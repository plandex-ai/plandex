package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"plandex/term"
	"plandex/version"

	"github.com/Masterminds/semver"
	"github.com/inconshreveable/go-update"
)

func checkForUpgrade() {
	latestVersionURL := "https://example.com/plandex/latest-version" // Placeholder URL
	resp, err := http.Get(latestVersionURL)
	if err != nil {
		log.Println("Error checking latest version:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	latestVersion, err := semver.NewVersion(string(body))
	if err != nil {
		log.Println("Error parsing latest version:", err)
		return
	}

	currentVersion, err := semver.NewVersion(version.Version)
	if err != nil {
		log.Println("Error parsing current version:", err)
		return
	}

	if latestVersion.GreaterThan(currentVersion) {
		fmt.Println("A new version of Plandex is available:", latestVersion)
		confirmed, err := term.ConfirmYesNo("Do you want to upgrade to the latest version?")
		if err != nil {
			log.Println("Error reading input:", err)
			return
		}

		if confirmed {
			err := doUpgrade(latestVersion.String())
			if err != nil {
				log.Println("Upgrade failed:", err)
				return
			}
			fmt.Println("Upgrade successful. Restarting Plandex...")
			restartPlandex()
		}
	}
}

func doUpgrade(version string) error {
	// Placeholder for download URL. This should point to the actual binary location.
	downloadURL := fmt.Sprintf("https://github.com/plandex-ai/plandex/releases/download/%s/plandex_%s_%s_%s.tar.gz", version, version, "os", "arch")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download the update: %w", err)
	}
	defer resp.Body.Close()

	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		return fmt.Errorf("failed to apply the update: %w", err)
	}

	return nil
}

func restartPlandex() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to determine executable path: %v", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Failed to restart: %v", err)
	}
	os.Exit(0)
}
