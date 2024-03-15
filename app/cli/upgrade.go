package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"plandex/term"
	"plandex/version"
	"runtime"

	"github.com/Masterminds/semver"
	"github.com/inconshreveable/go-update"
)

func checkForUpgrade() {
	latestVersionURL := "https://plandex.ai/cli-version.txt"
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
	tag := fmt.Sprintf("cli/v%s", version)
	escapedTag := url.QueryEscape(tag)

	downloadURL := fmt.Sprintf("https://github.com/plandex-ai/plandex/releases/download/%s/plandex_%s_%s_%s.tar.gz", escapedTag, version, runtime.GOOS, runtime.GOARCH)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download the update: %w", err)
	}
	defer resp.Body.Close()

	// Create a temporary file to save the downloaded archive
	tempFile, err := os.CreateTemp("", "*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up file afterwards

	// Copy the response body to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save the downloaded archive: %w", err)
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek in temporary file: %w", err)
	}

	// Now, extract the binary from the tempFile
	gzr, err := gzip.NewReader(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Check if the current file is the binary
		if header.Typeflag == tar.TypeReg && (header.Name == "plandex" || header.Name == "plandex.exe") {
			err = update.Apply(tarReader, update.Options{})
			if err != nil {
				return fmt.Errorf("failed to apply update: %w", err)
			}
			break
		}
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
