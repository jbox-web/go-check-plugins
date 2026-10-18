package checkmountpoints

import (
	"os"
	"time"

	"path/filepath"
	"reflect"
	"testing"

	"github.com/d-tux/go-fstab"
	"github.com/jessevdk/go-flags"

	"github.com/mackerelio/checkers"
	"github.com/shirou/gopsutil/v4/disk"
)

// gopsutil v4 returns mount options as a list; flags map to "true".
func TestParseOptions(t *testing.T) {
	got := parseOptions([]string{"rw", "relatime", "vers=4.2", "proto=tcp"})
	want := map[string]string{"rw": "true", "relatime": "true", "vers": "4.2", "proto": "tcp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseOptions() = %v, want %v", got, want)
	}
}

func TestParseOptionsKeepsEqualsInValue(t *testing.T) {
	got := parseOptions([]string{"opt=a=b"})
	if got["opt"] != "a=b" {
		t.Fatalf(`parseOptions(["opt=a=b"])["opt"] = %q, want "a=b"`, got["opt"])
	}
}

func TestBuildNFSHashIncludesNFS3AndNFS4(t *testing.T) {
	got := buildNFSHash([]disk.PartitionStat{
		{Mountpoint: "/data/v3", Fstype: "nfs"},
		{Mountpoint: "/data/v4", Fstype: "nfs4"},
		{Mountpoint: "/", Fstype: "ext4"},
	})
	if _, ok := got["/data/v3"]; !ok {
		t.Error("NFS v3 mount missing from the hash")
	}
	if _, ok := got["/data/v4"]; !ok {
		t.Error("NFS v4 mount missing from the hash")
	}
	if _, ok := got["/"]; ok {
		t.Error("ext4 mount present in the hash")
	}
}

func TestGetGlobalStatus(t *testing.T) {
	cases := []struct {
		name     string
		statuses []checkers.Status
		want     checkers.Status
	}{
		{"all ok", []checkers.Status{checkers.OK, checkers.OK}, checkers.OK},
		{"one unknown", []checkers.Status{checkers.OK, checkers.UNKNOWN}, checkers.UNKNOWN},
		{"critical wins", []checkers.Status{checkers.UNKNOWN, checkers.CRITICAL}, checkers.CRITICAL},
	}
	for _, tc := range cases {
		results := map[string]result{}
		for i, s := range tc.statuses {
			results[string(rune('a'+i))] = result{Status: s}
		}
		if got := getGlobalStatus(results); got != tc.want {
			t.Errorf("%s: getGlobalStatus() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBuildFinalMessageWithoutResults(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildFinalMessage(empty) panicked: %v", r)
		}
	}()
	buildFinalMessage(map[string]result{})
}

func TestBuildFinalMessageAlignsColumns(t *testing.T) {
	got := buildFinalMessage(map[string]result{
		"/a":   {Message: "Path not mounted"},
		"/bcd": {Message: "Path mounted as ro"},
	})
	want := "\n\n/a   : Path not mounted\n/bcd : Path mounted as ro"
	if got != want {
		t.Fatalf("buildFinalMessage() = %q, want %q", got, want)
	}
}

func TestWriteTest(t *testing.T) {
	dir := t.TempDir()
	if got := writeTest(dir); got.Status != checkers.OK {
		t.Fatalf("writeTest(writable dir) = %+v, want OK", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("writeTest left %d file(s) behind (err %v)", len(entries), err)
	}
	got := writeTest(filepath.Join(dir, "missing"))
	if got.Status != checkers.CRITICAL || got.Message != "Path mounted as rw but not writable (step: create file)" {
		t.Fatalf("writeTest(missing dir) = %+v, want CRITICAL at the create step", got)
	}
}

func TestCheckMount(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name  string
		mount disk.PartitionStat
		write bool
		want  result
	}{
		{"read-only", disk.PartitionStat{Mountpoint: dir, Opts: []string{"ro"}}, true, result{"Path mounted as ro", checkers.OK}},
		{"rw, no write test", disk.PartitionStat{Mountpoint: dir, Opts: []string{"rw"}}, false, result{"Path mounted as rw but not tested", checkers.OK}},
		{"rw, write test", disk.PartitionStat{Mountpoint: dir, Opts: []string{"rw"}}, true, result{"Path mounted as rw and writable", checkers.OK}},
		{"missing path", disk.PartitionStat{Mountpoint: filepath.Join(dir, "gone"), Opts: []string{"rw"}}, true, result{"Path does not exist", checkers.CRITICAL}},
	}
	for _, tc := range cases {
		if got := checkMount(tc.mount, tc.write); got != tc.want {
			t.Errorf("%s: checkMount() = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestExtractNFSEntryIncludesNFS4(t *testing.T) {
	got := extractNFSEntry(fstab.Mounts{
		{File: "/data/v4", VfsType: "nfs4"},
		{File: "/data/v3", VfsType: "nfs"},
		{File: "/", VfsType: "ext4"},
	})
	want := []string{"/data/v3", "/data/v4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractNFSEntry() = %v, want %v", got, want)
	}
}

// A mountpoint that cannot be stat-ed (stale handle, I/O or permission
// error) is not healthy, even when it exists.
func TestCheckMountStatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })
	got := checkMount(disk.PartitionStat{Mountpoint: filepath.Join(locked, "share"), Opts: []string{"rw"}}, false)
	if got.Status != checkers.CRITICAL {
		t.Fatalf("checkMount(unreadable path) = %+v, want CRITICAL", got)
	}
}

func TestWithTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	got := withTimeout(10*time.Millisecond, func() result {
		<-block
		return result{"never", checkers.OK}
	})
	if got.Status != checkers.CRITICAL {
		t.Fatalf("withTimeout(blocked call) = %+v, want CRITICAL", got)
	}
	got = withTimeout(time.Second, func() result { return result{"done", checkers.OK} })
	if got != (result{"done", checkers.OK}) {
		t.Fatalf("withTimeout(fast call) = %+v, want its own result", got)
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(&flags.Error{Type: flags.ErrHelp}); got != 0 {
		t.Errorf("exitCode(--help) = %d, want 0", got)
	}
	if got := exitCode(&flags.Error{Type: flags.ErrUnknownFlag}); got != 3 {
		t.Errorf("exitCode(unknown flag) = %d, want 3 (UNKNOWN)", got)
	}
}
