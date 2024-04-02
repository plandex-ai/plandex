package db

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

func init() {
	// ensure git is available
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("error running git --version: %v", err))
	}
}

func InitGitRepo(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)

	// Set the default branch name to 'main' for the new repository
	res, err := exec.Command("git", "-C", dir, "init", "-b", "main").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository with 'main' as default branch for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	// Configure user name and email for the repository
	if err := setGitConfig(dir, "user.email", "server@plandex.ai"); err != nil {
		return err
	}
	if err := setGitConfig(dir, "user.name", "Plandex"); err != nil {
		return err
	}

	return nil
}

func GitAddAndCommit(orgId, planId, branch, message string) error {
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

// func GitAddAndAmendCommit(orgId, planId, branch, addMessage string) error {
// 	dir := getPlanDir(orgId, planId)

// 	err := gitAdd(dir, ".")
// 	if err != nil {
// 		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
// 	}

// 	// Get the latest commit message
// 	_, latestCommitMsg, err := getLatestCommit(dir)
// 	if err != nil {
// 		return fmt.Errorf("error getting latest commit message for dir: %s, err: %v", dir, err)
// 	}

// 	// Amend the latest commit with the new message
// 	message := latestCommitMsg + "\n\n" + addMessage
// 	res, err := exec.Command("git", "-C", dir, "commit", "--amend", "-m", message).CombinedOutput()

// 	if err != nil {
// 		return fmt.Errorf("error amending commit for dir: %s, err: %v, output: %s", dir, err, string(res))
// 	}

// 	return nil
// }

func GitRewindToSha(orgId, planId, branch, sha string) error {
	dir := getPlanDir(orgId, planId)

	err := gitRewindToSha(dir, sha)
	if err != nil {
		return fmt.Errorf("error rewinding git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func GetGitCommitHistory(orgId, planId, branch string) (body string, shas []string, err error) {
	dir := getPlanDir(orgId, planId)

	body, shas, err = getGitCommitHistory(dir)
	if err != nil {
		return "", nil, fmt.Errorf("error getting git history for dir: %s, err: %v", dir, err)
	}

	return body, shas, nil
}

func GetLatestCommit(orgId, planId, branch string) (sha, body string, err error) {
	dir := getPlanDir(orgId, planId)

	sha, body, err = getLatestCommit(dir)
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit for dir: %s, err: %v", dir, err)
	}

	return sha, body, nil
}

func GitListBranches(orgId, planId string) ([]string, error) {
	dir := getPlanDir(orgId, planId)

	var out bytes.Buffer
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error getting git branches for dir: %s, err: %v", dir, err)
	}

	branches := strings.Split(strings.TrimSpace(out.String()), "\n")

	if len(branches) == 0 || (len(branches) == 1 && branches[0] == "") {
		return []string{"main"}, nil
	}

	return branches, nil
}

func GitCreateBranch(orgId, planId, branch, newBranch string) error {
	dir := getPlanDir(orgId, planId)

	res, err := exec.Command("git", "-C", dir, "checkout", "-b", newBranch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return nil
}

func GitDeleteBranch(orgId, planId, branchName string) error {
	dir := getPlanDir(orgId, planId)

	res, err := exec.Command("git", "-C", dir, "branch", "-D", branchName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return nil
}

func GitClearUncommittedChanges(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)

	// Reset staged changes
	res, err := exec.Command("git", "-C", dir, "reset", "--hard").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error resetting staged changes | err: %v, output: %s", err, string(res))
	}

	// Clean untracked files
	res, err = exec.Command("git", "-C", dir, "clean", "-d", "-f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error cleaning untracked files | err: %v, output: %s", err, string(res))
	}

	return nil
}

func gitCheckoutBranch(repoDir, branch string) error {
	// get current branch and only checkout if it's not the same
	// trying to check out the same branch will result in an error
	var out bytes.Buffer
	cmd := exec.Command("git", "-C", repoDir, "branch", "--show-current")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error getting current git branch for dir: %s, err: %v", repoDir, err)
	}

	currentBranch := strings.TrimSpace(out.String())

	log.Println("currentBranch:", currentBranch)

	if currentBranch == branch {
		return nil
	}

	log.Println("checking out branch:", branch)

	res, err := exec.Command("git", "-C", repoDir, "checkout", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking out git branch for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}

	return nil
}

func gitRewindToSha(repoDir, sha string) error {
	res, err := exec.Command("git", "-C", repoDir, "reset", "--hard",
		sha).CombinedOutput()

	if err != nil {
		return fmt.Errorf("error executing git reset for dir: %s, sha: %s, err: %v, output: %s", repoDir, sha, err, string(res))
	}

	return nil
}

func getLatestCommit(dir string) (sha, body string, err error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "log", "--pretty=%h@@|@@%at@@|@@%B@>>>@")
	cmd.Dir = dir
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("error getting git history for dir: %s, err: %v",
			dir, err)
	}

	// Process the log output to get it in the desired format.
	history := processGitHistoryOutput(strings.TrimSpace(out.String()))

	first := history[0]

	sha = first[0]
	body = first[1]

	return sha, body, nil
}

func getGitCommitHistory(dir string) (body string, shas []string, err error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "log", "--pretty=%h@@|@@%at@@|@@%B@>>>@")
	cmd.Dir = dir
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", nil, fmt.Errorf("error getting git history for dir: %s, err: %v",
			dir, err)
	}

	// Process the log output to get it in the desired format.
	history := processGitHistoryOutput(strings.TrimSpace(out.String()))

	var output []string
	for _, el := range history {
		shas = append(shas, el[0])
		output = append(output, el[1])
	}

	return strings.Join(output, "\n\n"), shas, nil
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

			dt := time.Unix(timestamp, 0).UTC()
			formattedTs := dt.Format("Mon Jan 2, 2006 | 3:04:05pm MST")

			// Prepare the header with colors.
			headerColor := color.New(color.FgCyan, color.Bold)
			dateColor := color.New(color.FgCyan)

			// Combine sha, formatted timestamp, and message header into one string.
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

func gitRemoveIndexLockFileIfExists(repoDir string) error {
	// Remove the lock file if it exists
	lockFilePath := filepath.Join(repoDir, ".git", "index.lock")
	_, err := os.Stat(lockFilePath)

	if err == nil {
		if err := os.Remove(lockFilePath); err != nil {
			return fmt.Errorf("error removing lock file: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking lock file: %v", err)
	}

	return nil
}

func setGitConfig(repoDir, key, value string) error {
	res, err := exec.Command("git", "-C", repoDir, "config", key, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting git config %s to %s for dir: %s, err: %v, output: %s", key, value, repoDir, err, string(res))
	}
	return nil
}
