package checkfileage

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
)

// Do the plugin
func Do() {
	ckr := run(os.Args[1:])
	ckr.Name = "FileAge"
	ckr.Exit()
}

type monitor struct {
	warningAge   int64
	warningSize  int64
	criticalAge  int64
	criticalSize int64
}

func (m monitor) hasWarningAge() bool {
	return m.warningAge != 0
}

func (m monitor) hasWarningSize() bool {
	return m.warningSize != 0
}

func (m monitor) CheckWarning(age, size int64) bool {
	return (m.hasWarningAge() && m.warningAge < age) ||
		(m.hasWarningSize() && m.warningSize > size)
}

func (m monitor) hasCriticalAge() bool {
	return m.criticalAge != 0
}

func (m monitor) hasCriticalSize() bool {
	return m.criticalSize != 0
}

func (m monitor) CheckCritical(age, size int64) bool {
	return (m.hasCriticalAge() && m.criticalAge < age) ||
		(m.hasCriticalSize() && m.criticalSize > size)
}

func newMonitor(warningAge, warningSize, criticalAge, criticalSize int64) *monitor {
	return &monitor{
		warningAge:   warningAge,
		warningSize:  warningSize,
		criticalAge:  criticalAge,
		criticalSize: criticalSize,
	}
}

func plural(count int, singular string) (result string) {
	if (count == 1) || (count == 0) {
		result = strconv.Itoa(count) + " " + singular + " "
	} else {
		result = strconv.Itoa(count) + " " + singular + "s "
	}
	return
}

func secondsToHuman(input int64) (result string) {
	years := math.Floor(float64(input) / 60 / 60 / 24 / 7 / 30 / 12)
	seconds := input % (60 * 60 * 24 * 7 * 30 * 12)
	months := math.Floor(float64(seconds) / 60 / 60 / 24 / 7 / 30)
	seconds = input % (60 * 60 * 24 * 7 * 30)
	weeks := math.Floor(float64(seconds) / 60 / 60 / 24 / 7)
	seconds = input % (60 * 60 * 24 * 7)
	days := math.Floor(float64(seconds) / 60 / 60 / 24)
	seconds = input % (60 * 60 * 24)
	hours := math.Floor(float64(seconds) / 60 / 60)
	seconds = input % (60 * 60)
	minutes := math.Floor(float64(seconds) / 60)
	seconds = input % 60

	if years > 0 {
		result = plural(int(years), "year") + plural(int(months), "month") + plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if months > 0 {
		result = plural(int(months), "month") + plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if weeks > 0 {
		result = plural(int(weeks), "week") + plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if days > 0 {
		result = plural(int(days), "day") + plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if hours > 0 {
		result = plural(int(hours), "hour") + plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else if minutes > 0 {
		result = plural(int(minutes), "minute") + plural(int(seconds), "second")
	} else {
		result = plural(int(seconds), "second")
	}

	return
}

var opts struct {
	File          string `short:"f" long:"file" required:"true" description:"monitor file name"`
	WarningAge    int64  `short:"w" long:"warning-age" default:"240" description:"warning if more old than"`
	WarningSize   int64  `short:"W" long:"warning-size" description:"warning if file size less than"`
	CriticalAge   int64  `short:"c" long:"critical-age" default:"600" description:"critical if more old than"`
	CriticalSize  int64  `short:"C" long:"critical-size" default:"0" description:"critical if file size less than"`
	IgnoreMissing bool   `short:"i" long:"ignore-missing" description:"skip alert if file doesn't exist"`
}

func run(args []string) *checkers.Checker {
	_, err := flags.ParseArgs(&opts, args)
	if err != nil {
		os.Exit(1)
	}

	stat, err := os.Stat(opts.File)
	if err != nil {
		if opts.IgnoreMissing {
			return checkers.Ok("No such file, but ignore missing is set.")
		}
		return checkers.Unknown(err.Error())
	}

	monitor := newMonitor(opts.WarningAge, opts.WarningSize, opts.CriticalAge, opts.CriticalSize)

	result := checkers.OK

	mtime := stat.ModTime()
	age := time.Now().Unix() - mtime.Unix()
	size := stat.Size()

	if monitor.CheckWarning(age, size) {
		result = checkers.WARNING
	}

	if monitor.CheckCritical(age, size) {
		result = checkers.CRITICAL
	}

	duration := strings.TrimSpace(secondsToHuman(age))
	msg := fmt.Sprintf("%s is %d seconds old (%s) and %d bytes.", opts.File, age, duration, size)
	return checkers.NewChecker(result, msg)
}
