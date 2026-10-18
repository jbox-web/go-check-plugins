package checkmountpoints

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/d-tux/go-fstab"
	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
	"github.com/shirou/gopsutil/v4/disk"
)

var opts struct {
	WriteTest   bool     `short:"w" long:"write-test" description:"run write test"`
	MountPoints []string `short:"m" description:"list of mountpoints to check. Can be repeated multiple times"`
	Timeout     float64  `short:"t" long:"timeout" default:"10" description:"Seconds before a mountpoint that does not answer is reported CRITICAL"`
}

type result struct {
	Message string
	Status  checkers.Status
}

// Do the plugin
func Do() {
	ckr := run(os.Args[1:])
	ckr.Name = "MountPoints"
	ckr.Exit()
}

// buildNFSHash indexes the mounted NFS partitions (v3 and v4) by mountpoint.
func buildNFSHash(disks []disk.PartitionStat) map[string]disk.PartitionStat {
	partitionsHash := make(map[string]disk.PartitionStat)

	for _, d := range disks {
		if d.Fstype == "nfs" || d.Fstype == "nfs4" {
			partitionsHash[d.Mountpoint] = d
		}
	}

	return partitionsHash
}

// parseOptions maps mount options to their value, "true" for bare flags.
// gopsutil v4 already splits the options list.
func parseOptions(opts []string) map[string]string {
	parsedOptions := make(map[string]string)

	for _, opt := range opts {
		key, value, found := strings.Cut(opt, "=")
		if !found {
			value = "true"
		}
		parsedOptions[key] = value
	}

	return parsedOptions
}

// getGlobalStatus returns the worst status: CRITICAL, then UNKNOWN, then OK.
func getGlobalStatus(resultCheck map[string]result) checkers.Status {
	globalStatus := checkers.OK

	for _, r := range resultCheck {
		switch r.Status {
		case checkers.CRITICAL:
			return checkers.CRITICAL
		case checkers.UNKNOWN:
			globalStatus = checkers.UNKNOWN
		}
	}

	return globalStatus
}

// buildFinalMessage lists one result per mountpoint, sorted, colons aligned.
func buildFinalMessage(resultCheck map[string]result) string {
	message := "\n"

	keys := make([]string, 0, len(resultCheck))
	maxLength := 0
	for k := range resultCheck {
		keys = append(keys, k)
		maxLength = max(maxLength, len(k))
	}

	sort.Strings(keys)

	for _, k := range keys {
		padding := (maxLength - len(k)) + 1
		message += fmt.Sprintf("\n%s%-*s: %s", k, padding, " ", resultCheck[k].Message)
	}

	return message
}

func extractNFSEntry(mounts fstab.Mounts) []string {
	keys := make([]string, 0, len(mounts))

	for _, fstabEntry := range mounts {
		// go-fstab's IsNFS() only matches "nfs", not "nfs4".
		if fstabEntry.VfsType == "nfs" || fstabEntry.VfsType == "nfs4" {
			keys = append(keys, fstabEntry.File)
		}
	}
	sort.Strings(keys)
	return keys
}

// writeTest creates, writes, closes and removes a file in dir, and reports
// the first step that fails.
func writeTest(dir string) result {
	tmpfile, err := os.CreateTemp(dir, "checkmountpoints")
	if err != nil {
		return result{"Path mounted as rw but not writable (step: create file)", checkers.CRITICAL}
	}
	if _, err := tmpfile.Write([]byte("temporary file's content")); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return result{"Path mounted as rw but not writable (step: write file)", checkers.CRITICAL}
	}
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return result{"Path mounted as rw but not writable (step: close file)", checkers.CRITICAL}
	}
	if err := os.Remove(tmpfile.Name()); err != nil {
		return result{"Path mounted as rw but not writable (step: remove file)", checkers.CRITICAL}
	}
	return result{"Path mounted as rw and writable", checkers.OK}
}

// checkMount checks one mountpoint that is both in fstab and mounted.
func checkMount(mount disk.PartitionStat, write bool) result {
	if _, err := os.Stat(mount.Mountpoint); os.IsNotExist(err) {
		return result{"Path does not exist", checkers.CRITICAL}
	} else if err != nil {
		return result{fmt.Sprintf("Path not accessible: %s", err), checkers.CRITICAL}
	}
	if _, ok := parseOptions(mount.Opts)["rw"]; !ok {
		return result{"Path mounted as ro", checkers.OK}
	}
	if !write {
		return result{"Path mounted as rw but not tested", checkers.OK}
	}
	return writeTest(mount.Mountpoint)
}

func run(args []string) *checkers.Checker {
	if _, err := flags.ParseArgs(&opts, args); err != nil {
		os.Exit(exitCode(err))
	}

	if len(opts.MountPoints) == 0 {
		return checkers.Unknown("No mountpoint to check: pass at least one -m")
	}

	disks, err := disk.Partitions(true)
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch disks info: %s", err))
	}

	nfsHash := buildNFSHash(disks)

	mounts, err := fstab.ParseSystem()
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch fstab info: %s", err))
	}

	nfsFstab := extractNFSEntry(mounts)

	resultCheck := make(map[string]result)

	for _, entry := range opts.MountPoints {
		if !contains(nfsFstab, entry) {
			resultCheck[entry] = result{"Path not found in fstab", checkers.UNKNOWN}
			continue
		}
		mount, ok := nfsHash[entry]
		if !ok {
			resultCheck[entry] = result{"Path not mounted", checkers.CRITICAL}
			continue
		}
		timeout := time.Duration(opts.Timeout * float64(time.Second))
		resultCheck[mount.Mountpoint] = withTimeout(timeout, func() result {
			return checkMount(mount, opts.WriteTest)
		})
	}

	return checkers.NewChecker(getGlobalStatus(resultCheck), buildFinalMessage(resultCheck))
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// withTimeout runs fn, giving up after d: a hard NFS mount whose server is
// gone blocks stat and write calls forever. The blocked goroutine is left
// behind; the plugin exits right after.
func withTimeout(d time.Duration, fn func() result) result {
	done := make(chan result, 1)
	go func() { done <- fn() }()
	select {
	case r := <-done:
		return r
	case <-time.After(d):
		return result{fmt.Sprintf("Path not responding after %s", d), checkers.CRITICAL}
	}
}

// exitCode turns a flag parsing failure into the plugin's exit code: 0 for
// --help, UNKNOWN for a bad command line (exit 1 would read as WARNING).
func exitCode(err error) int {
	if flags.WroteHelp(err) {
		return 0
	}
	return int(checkers.UNKNOWN)
}
