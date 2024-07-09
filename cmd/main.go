package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

const defaultcmd = "scan"

type globalOpt struct {
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
}

var (
	globalOpts globalOpt
	cli        = flags.NewParser(&globalOpts, flags.Default)
)

func main() {
	ensureDefaultCommand(cli, defaultcmd)
	cli.SubcommandsOptional = false
	_, _ = cli.Parse()
}

func ensureDefaultCommand(p *flags.Parser, defaultCmd string) {
	if len(os.Args) < 2 {
		// os.Args = append(os.Args, defaultCmd)
		return
	}

	for _, c := range p.Commands() {
		if c.Name == os.Args[1] {
			return
		}
	}

	os.Args = append([]string{os.Args[0], defaultCmd}, os.Args[1:]...)
}
