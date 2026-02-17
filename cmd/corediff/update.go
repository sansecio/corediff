package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	selfupdate "github.com/gwillem/go-selfupdate"
)

type updateArg struct{}

var (
	updateCmd     updateArg
	selfUpdateURL = fmt.Sprintf("https://sansec.io/downloads/%s-%s/corediff", runtime.GOOS, runtime.GOARCH)
)

func (u *updateArg) Execute(_ []string) error {
	fmt.Println("Checking for updates...")
	updated, err := selfupdate.Update(selfUpdateURL)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	if !updated {
		fmt.Println("Already running latest version", corediffVersion)
		return nil
	}

	newVersion, err := getInstalledVersion()
	if err != nil {
		fmt.Printf("Updated from %s (could not determine new version: %v)\n", corediffVersion, err)
	} else {
		fmt.Printf("Updated %s -> %s\n", corediffVersion, newVersion)
	}
	return nil
}

// getInstalledVersion runs the updated binary with --version to get its version string.
func getInstalledVersion() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(exe, "--version").Output()
	if err != nil {
		return "", err
	}
	// output is "corediff v1.2.3\n"
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "corediff")), nil
}

func init() {
	cli.AddCommand("update", "Update corediff binary", "Download and install the latest corediff binary", &updateCmd)
}
