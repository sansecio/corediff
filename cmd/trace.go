package main

import (
	"fmt"
	"os"
)

type traceArg struct{}

var traceArgs traceArg

func init() {
	_, _ = cli.AddCommand("trace", "Trace files or stdin", "Trace files or stdin", &traceArgs)
}

func (t *traceArg) Execute(args []string) error {
	trace := func(line []byte) {
		fmt.Printf("%016x %s\n", hash(normalizeLine(line)), line)
	}

	if len(args) == 0 || args[0] == "-" {
		return parseFH(os.Stdin, trace)
	}

	for _, p := range args {
		err := parseFile(p, trace)
		if err != nil {
			return err
		}
	}
	return nil
}
