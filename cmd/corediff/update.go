package main

import (
	"fmt"
	"runtime"

	selfupdate "github.com/gwillem/go-selfupdate"
)

type updateArg struct{}

var (
	updateCmd     updateArg
	selfUpdateURL = fmt.Sprintf("https://sansec.io/downloads/%s-%s/corediff", runtime.GOOS, runtime.GOARCH)
)

func (u *updateArg) Execute(_ []string) error {
	fmt.Println("Checking for updates...")
	restarted, err := selfupdate.UpdateRestart(selfUpdateURL)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	if !restarted {
		fmt.Println("Already running latest version", corediffVersion)
	}
	return nil
}

func init() {
	cli.AddCommand("update", "Update corediff binary", "Download and install the latest corediff binary", &updateCmd)
}
