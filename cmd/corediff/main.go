package main

import (
	"fmt"
	"os"
	"runtime"

	buildversion "github.com/gwillem/go-buildversion"
	"github.com/jessevdk/go-flags"
)

const defaultCmd = "scan"

type globalOpt struct {
	Verbose  []bool `short:"v" long:"verbose" description:"Verbose output (-v info, -vv per-file details)"`
	Version  bool   `long:"version" description:"Print version and exit"`
	Parallel int    `short:"p" long:"parallel" description:"Parallel workers (default: number of CPUs)" default:"0"`
}

// parallelLimit returns the configured parallelism, defaulting to GOMAXPROCS.
func parallelLimit() int {
	if globalOpts.Parallel > 0 {
		return globalOpts.Parallel
	}
	return runtime.GOMAXPROCS(0)
}

var (
	globalOpts      globalOpt
	cli             = flags.NewParser(&globalOpts, flags.Default)
	corediffVersion = buildversion.String()
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("corediff", corediffVersion)
		return
	}
	ensureDefaultCommand(cli, defaultCmd)
	cli.SubcommandsOptional = false
	_, _ = cli.Parse()
}

func ensureDefaultCommand(p *flags.Parser, cmd string) {
	if len(os.Args) < 2 {
		return
	}
	for _, c := range p.Commands() {
		if c.Name == os.Args[1] {
			return
		}
	}
	os.Args = append([]string{os.Args[0], cmd}, os.Args[1:]...)
}
