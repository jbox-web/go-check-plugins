package checkswap

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
	"github.com/shirou/gopsutil/v4/mem"
)

var opts struct {
	Warning  string `short:"w" long:"warning" default:"95" description:"Sets warning value for Swap Usage. Default is 95%"`
	Critical string `short:"c" long:"critical" default:"98" description:"Sets critical value for Swap Usage. Default is 98%"`
}

// Do the plugin
func Do() {
	ckr := run(os.Args[1:])
	ckr.Name = "Swap"
	ckr.Exit()
}

// checkStatus compares the used percentage with the warning and critical
// thresholds, given as the raw command-line strings.
func checkStatus(usedPercent float64, warning, critical string) (checkers.Status, error) {
	warnThreshold, err := strconv.ParseFloat(warning, 64)
	if err != nil {
		return checkers.UNKNOWN, fmt.Errorf("invalid warning threshold %q: %w", warning, err)
	}
	critThreshold, err := strconv.ParseFloat(critical, 64)
	if err != nil {
		return checkers.UNKNOWN, fmt.Errorf("invalid critical threshold %q: %w", critical, err)
	}

	// Critical first: a usage above the critical threshold is above the
	// warning one too.
	switch {
	case usedPercent >= critThreshold:
		return checkers.CRITICAL, nil
	case usedPercent >= warnThreshold:
		return checkers.WARNING, nil
	}
	return checkers.OK, nil
}

func run(args []string) *checkers.Checker {
	_, err := flags.ParseArgs(&opts, args)
	if err != nil {
		os.Exit(exitCode(err))
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Failed to fetch swap info: %s", err))
	}

	checkState, err := checkStatus(swap.UsedPercent, opts.Warning, opts.Critical)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	total := humanize.Bytes(swap.Total)
	used := humanize.Bytes(swap.Used)
	free := humanize.Bytes(swap.Free)
	percent := humanize.FtoaWithDigits(swap.UsedPercent, 2)

	message := fmt.Sprintf("Total: %s - Used: %s (%s%%) - Free: %s", total, used, percent, free)
	return checkers.NewChecker(checkState, message)
}

// exitCode turns a flag parsing failure into the plugin's exit code: 0 for
// --help, UNKNOWN for a bad command line (exit 1 would read as WARNING).
func exitCode(err error) int {
	if flags.WroteHelp(err) {
		return 0
	}
	return int(checkers.UNKNOWN)
}
