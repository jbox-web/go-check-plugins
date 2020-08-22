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
	WriteTest bool `short:"w" long:"write-test" description:"run write test"`
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
	message := ""

	keys := make([]string, 0, len(resultCheck))

	for k := range resultCheck {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		result := resultCheck[k]
		message += fmt.Sprintf("\n%s: %s", k, result.Message)
	}

	return message
}

func run(args []string) *checkers.Checker {
	_, err := flags.ParseArgs(&opts, args)
	if err != nil {
		os.Exit(1)
	}

	disks, err := disk.Partitions(true)
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch disks info: %s", err))
	}

	fstab, err := fstab.ParseSystem()
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch fstab info: %s", err))
	}

	nfsHash := buildNFSHash(disks)

	var resultCheck map[string]result
	resultCheck = make(map[string]result)

	for _, fstabEntry := range fstab {
		if fstabEntry.IsNFS() {
			if mount, ok := nfsHash[fstabEntry.File]; ok {
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
								message += fmt.Sprintf("Path not writable (step: create file)")
								status = checkers.CRITICAL
							} else {
								if _, err := tmpfile.Write(content); err != nil {
									message += fmt.Sprintf("Path not writable (step: write file)")
									status = checkers.CRITICAL
								} else {
									if err := tmpfile.Close(); err != nil {
										message += fmt.Sprintf("Path not writable (step: close file)")
										status = checkers.CRITICAL
									} else {
										if err := os.Remove(tmpfile.Name()); err != nil {
											message += fmt.Sprintf("Path not writable (step: remove file)")
											status = checkers.CRITICAL
										} else {
											message += fmt.Sprintf("Path is writable")
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
			}
		} else {
			continue
		}
	}

	globalStatus := getGlobalStatus(resultCheck)
	globalMessage := buildFinalMessage(resultCheck)

	return checkers.NewChecker(globalStatus, globalMessage)
}
