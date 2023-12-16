package db

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

func init() {
	// ensure git is available
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("Error running git --version: %v", err))
	}
}

func InitGitRepo(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)

	res, err := exec.Command("git", "init", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return nil
}

func GitAddAndCommit(orgId, planId, message string) error {
	dir := getPlanDir(orgId, planId)

	err := gitAdd(dir, ".")
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
	}

	err = gitCommit(dir, message)
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func gitAdd(repoDir, path string) error {
	res, err := exec.Command("git", "-C", repoDir, "add", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}

	return nil
}

func gitCommit(repoDir, commitMsg string) error {
	res, err := exec.Command("git", "-C", repoDir, "commit", "-m", commitMsg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}

	return nil
}

func GitRewindToSHA(repoDir, sha string) error {
	res, err := exec.Command("git", "-C", repoDir, "reset", "--hard",
		sha).CombinedOutput()

	if err != nil {
		return fmt.Errorf("error executing git reset for dir: %s, sha: %s, err: %v, output: %s", repoDir, sha, err, string(res))
	}

	return nil
}

func GetLatestCommit(dir string) (string, string, error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "log", "--pretty=%h@@|@@%at@@|@@%B@>>>@")
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("error getting git history for dir: %s, err: %v",
			dir, err)
	}

	// Process the log output to get it in the desired format.
	history := processGitHistoryOutput(strings.TrimSpace(out.String()))

	first := history[0]

	return first[0], first[1], nil
}

func GetGitCommitHistory(dir string) (string, error) {

	var out bytes.Buffer
	cmd := exec.Command("git", "log", "--pretty=%h@@|@@%at@@|@@%B@>>>@")
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error getting git history for dir: %s, err: %v",
			dir, err)
	}

	// Process the log output to get it in the desired format.
	history := processGitHistoryOutput(strings.TrimSpace(out.String()))

	var output []string
	for _, el := range history {
		output = append(output, el[1])
	}

	return strings.Join(output, "\n\n"), nil
}

// processGitHistoryOutput processes the raw output from the git log command and returns a formatted string.
func processGitHistoryOutput(raw string) [][2]string {
	var history [][2]string
	entries := strings.Split(raw, "@>>>@") // Split entries using the custom separator.

	for _, entry := range entries {
		// First clean up any leading/trailing whitespace or newlines from each entry.
		entry = strings.TrimSpace(entry)

		// Now split the cleaned entry into its parts.
		parts := strings.Split(entry, "@@|@@")
		if len(parts) == 3 {
			sha := parts[0]
			timestampStr := parts[1]
			message := strings.TrimSpace(parts[2]) // Trim whitespace from message as well.

			// Extract and format timestamp.
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				continue // Skip entries with invalid timestamps.
			}

			dt := time.Unix(timestamp, 0).Local()
			formattedTs := dt.Format("Mon Jan 2, 2006 | 3:04:05pm MST")
			if dt.Day() == time.Now().Day() {
				formattedTs = dt.Format("Today | 3:04:05pm MST")
			} else if dt.Day() == time.Now().AddDate(0, 0, -1).Day() {
				formattedTs = dt.Format("Yesterday | 3:04:05pm MST")
			}

			// Prepare the header with colors.
			headerColor := color.New(color.FgCyan, color.Bold)
			dateColor := color.New(color.FgCyan)

			// Combine SHA, formatted timestamp, and message header into one string.
			header := fmt.Sprintf("%s | %s", headerColor.Sprintf("📝 Update %s", sha), dateColor.Sprintf("%s", formattedTs))

			// Combine header and message with a newline only if the message is not empty.
			fullEntry := header
			if message != "" {
				fullEntry += "\n" + message
			}

			history = append(history, [2]string{sha, fullEntry})
		}
	}

	return history
}
