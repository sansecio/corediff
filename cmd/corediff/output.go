package main

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	boldred   = color.New(color.FgHiRed, color.Bold).SprintFunc()
	grey      = color.New(color.FgHiBlack).SprintFunc()
	boldwhite = color.New(color.FgHiWhite).SprintFunc()
	warn      = color.New(color.FgYellow, color.Bold).SprintFunc()
	alarm     = color.New(color.FgHiWhite, color.BgHiRed, color.Bold).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()

	logLevel = 1
)

func logVerbose(a ...any) {
	if logLevel >= 3 {
		fmt.Println(a...)
	}
}

func init() {
	color.NoColor = false
}
