package checkswap

import (
  "fmt"
  "github.com/dustin/go-humanize"
  "github.com/jessevdk/go-flags"
  "github.com/mackerelio/checkers"
  "github.com/shirou/gopsutil/mem"
  "os"
  "strconv"
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

func run(args []string) *checkers.Checker {
  _, err := flags.ParseArgs(&opts, args)
  if err != nil {
    os.Exit(1)
  }

  swap, err := mem.SwapMemory()
  if err != nil {
    return checkers.Unknown(fmt.Sprintf("Failed to fetch swap info: %s", err))
  }

  var checkState checkers.Status

  warnThreshold, err := strconv.ParseFloat(opts.Warning, 64)
  critThreshold, err := strconv.ParseFloat(opts.Critical, 64)

  if swap.UsedPercent >= warnThreshold {
    checkState = checkers.WARNING
  } else if swap.UsedPercent >= critThreshold {
    checkState = checkers.CRITICAL
  } else {
    checkState = checkers.OK
  }

  total := humanize.Bytes(swap.Total)
  used := humanize.Bytes(swap.Used)
  free := humanize.Bytes(swap.Free)
  percent := humanize.FtoaWithDigits(swap.UsedPercent, 2)

  message := fmt.Sprintf("Total: %s - Used: %s (%s%%) - Free: %s", total, used, percent, free)
  return checkers.NewChecker(checkState, message)
}
