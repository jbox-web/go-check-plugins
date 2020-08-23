package checkmountpoints

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
	"github.com/n-rodriguez/go-fstab"
	"github.com/shirou/gopsutil/disk"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

var opts struct {
	WriteTest   bool     `short:"w" long:"write-test" description:"run write test"`
	MountPoints []string `short:"m" description:"list of mountpoints to check. Can be repeated multiple times"`
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

func buildNFSHash(disks []disk.PartitionStat) map[string]disk.PartitionStat {
	var partitionsHash map[string]disk.PartitionStat
	partitionsHash = make(map[string]disk.PartitionStat)

	for _, disk := range disks {
		if disk.Fstype == "nfs4" {
			partitionsHash[disk.Mountpoint] = disk
		} else {
			continue
		}
	}

	return partitionsHash
}

func parseOptions(opts string) map[string]string {
	var parsedOptions map[string]string
	parsedOptions = make(map[string]string)

	splitOpts := strings.Split(opts, ",")

	for _, opt := range splitOpts {
		splitOpt := strings.Split(opt, "=")
		if len(splitOpt) == 1 {
			parsedOptions[splitOpt[0]] = "true"
		} else {
			parsedOptions[splitOpt[0]] = splitOpt[1]
		}
	}

	return parsedOptions
}

func getGlobalStatus(resultCheck map[string]result) checkers.Status {
	var globalStatus checkers.Status
	globalStatus = checkers.OK

	for _, result := range resultCheck {
		if result.Status == checkers.CRITICAL {
			globalStatus = checkers.CRITICAL
			break
		}
	}

	return globalStatus
}

func buildFinalMessage(resultCheck map[string]result) string {
	message := "\n"

	keys := make([]string, 0, len(resultCheck))

	for k := range resultCheck {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	l := longest(keys)[0]
	maxLength := len(l)

	for _, k := range keys {
		result := resultCheck[k]
		padding := (maxLength - len(k)) + 1
		message += fmt.Sprintf("\n%s%-*s: %s", k, padding, " ", result.Message)
	}

	return message
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// See: https://stackoverflow.com/a/54832590
func longest(a []string) []string {
	var l []string
	if len(a) > 0 {
		l = append(l, a[0])
		a = a[1:]
	}
	for _, s := range a {
		if len(l[0]) <= len(s) {
			if len(l[0]) < len(s) {
				l = l[:0]
			}
			l = append(l, s)
		}
	}
	return append([]string(nil), l...)
}

func extractNFSEntry(fstab fstab.Mounts) []string {
	keys := make([]string, 0, len(fstab))

	for _, fstabEntry := range fstab {
		if fstabEntry.IsNFS() {
			keys = append(keys, fstabEntry.File)
		} else {
			continue
		}
	}
	sort.Strings(keys)
	return keys
}

func run(args []string) *checkers.Checker {
	args, err := flags.ParseArgs(&opts, args)
	if err != nil {
		os.Exit(1)
	}

	mounts := opts.MountPoints

	disks, err := disk.Partitions(true)
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch disks info: %s", err))
	}

	nfsHash := buildNFSHash(disks)

	fstab, err := fstab.ParseSystem()
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch fstab info: %s", err))
	}

	nfsFstab := extractNFSEntry(fstab)

	var resultCheck map[string]result
	resultCheck = make(map[string]result)

	for _, entry := range mounts {
		if contains(nfsFstab, entry) {
			if mount, ok := nfsHash[entry]; ok {
				message := ""
				var status checkers.Status

				if _, err := os.Stat(mount.Mountpoint); os.IsNotExist(err) {
					message += fmt.Sprintf("Path does not exist")
					status = checkers.CRITICAL
				} else {
					parsedOptions := parseOptions(mount.Opts)
					if _, ok := parsedOptions["rw"]; ok {
						if opts.WriteTest {
							content := []byte("temporary file's content")
							tmpfile, err := ioutil.TempFile(mount.Mountpoint, "checkmountpoints")

							if err != nil {
								message += fmt.Sprintf("Path mounted as rw but not writable (step: create file)")
								status = checkers.CRITICAL
							} else {
								if _, err := tmpfile.Write(content); err != nil {
									message += fmt.Sprintf("Path mounted as rw but not writable (step: write file)")
									status = checkers.CRITICAL
								} else {
									if err := tmpfile.Close(); err != nil {
										message += fmt.Sprintf("Path mounted as rw but not writable (step: close file)")
										status = checkers.CRITICAL
									} else {
										if err := os.Remove(tmpfile.Name()); err != nil {
											message += fmt.Sprintf("Path mounted as rw but not writable (step: remove file)")
											status = checkers.CRITICAL
										} else {
											message += fmt.Sprintf("Path mounted as rw and writable")
											status = checkers.OK
										}
									}
								}
							}
						} else {
							message += fmt.Sprintf("Path mounted as rw but not tested")
						}
					} else {
						message += fmt.Sprintf("Path mounted as ro")
					}
				}

				resultCheck[mount.Mountpoint] = result{Message: message, Status: status}
			} else {
				message := ""
				var status checkers.Status
				message += fmt.Sprintf("Path not mounted")
				status = checkers.CRITICAL
				resultCheck[entry] = result{Message: message, Status: status}
			}
		} else {
			message := ""
			var status checkers.Status
			message += fmt.Sprintf("Path not found in fstab")
			status = checkers.UNKNOWN
			resultCheck[entry] = result{Message: message, Status: status}
		}
	}

	globalStatus := getGlobalStatus(resultCheck)
	globalMessage := buildFinalMessage(resultCheck)

	return checkers.NewChecker(globalStatus, globalMessage)
}
