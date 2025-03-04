package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"plandex-cli/term"
	"plandex-cli/version"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
)

func checkForUpgrade() {
	if os.Getenv("PLANDEX_SKIP_UPGRADE") != "" {
		return
	}

	if version.Version == "development" {
		return
	}

	term.StartSpinner("")
	defer term.StopSpinner()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	latestVersionURL := "https://plandex.ai/v2/cli-version.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestVersionURL, nil)
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
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

	versionStr := string(body)
	versionStr = strings.TrimSpace(versionStr)

	latestVersion, err := semver.NewVersion(versionStr)
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
		term.StopSpinner()
		fmt.Println("A new version of Plandex is available:", color.New(color.Bold, term.ColorHiGreen).Sprint(versionStr))
		fmt.Printf("Current version: %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(version.Version))
		confirmed, err := term.ConfirmYesNo("Upgrade to the latest version?")
		if err != nil {
			log.Println("Error reading input:", err)
			return
		}

		if confirmed {
			term.ResumeSpinner()
			err := doUpgrade(latestVersion.String())
			if err != nil {
				term.OutputErrorAndExit("Failed to upgrade: %v", err)
				return
			}
			term.StopSpinner()
			restartPlandex()
		} else {
			fmt.Println("Note: set PLANDEX_SKIP_UPGRADE=1 to stop upgrade prompts")
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
				if errors.Is(err, fs.ErrPermission) {
					return fmt.Errorf("failed to apply update due to permission error; please try running your command again with 'sudo': %w", err)
				}
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
		term.OutputErrorAndExit("Failed to determine executable path: %v", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		term.OutputErrorAndExit("Failed to restart: %v", err)
	}

	err = cmd.Wait()

	// If the process exited with an error, exit with the same error code
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else if err != nil {
		term.OutputErrorAndExit("Failed to restart: %v", err)
	}

	os.Exit(0)
}
