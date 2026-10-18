package checkfileage

import (
	"fmt"
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

// Durations used to break an age down, largest first: a month is 30 days and
// a year 365 days.
var units = []struct {
	name    string
	seconds int64
}{
	{"year", 365 * 24 * 3600},
	{"month", 30 * 24 * 3600},
	{"week", 7 * 24 * 3600},
	{"day", 24 * 3600},
	{"hour", 3600},
	{"minute", 60},
	{"second", 1},
}

// secondsToHuman spells an age out, from its largest non-zero unit down to
// seconds, e.g. "1 hour 0 minute 5 seconds ".
func secondsToHuman(input int64) (result string) {
	remaining := input
	started := false
	for _, u := range units {
		count := remaining / u.seconds
		remaining %= u.seconds
		if count > 0 || started || u.name == "second" {
			started = true
			result += plural(int(count), u.name)
		}
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
		os.Exit(exitCode(err))
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

// exitCode turns a flag parsing failure into the plugin's exit code: 0 for
// --help, UNKNOWN for a bad command line (exit 1 would read as WARNING).
func exitCode(err error) int {
	if flags.WroteHelp(err) {
		return 0
	}
	return int(checkers.UNKNOWN)
}
