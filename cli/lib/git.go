package lib

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"plandex/types"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var gitMutex sync.Mutex

func InitGitRepo(dir string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "init", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return GitAddAndCommit(dir, "New plan", false)
}

func AddGitSubmodule(rootDir, submoduleDir string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	// Calculate the relative path from rootDir to submoduleDir
	relPath, err := filepath.Rel(rootDir, submoduleDir)
	if err != nil {
		return fmt.Errorf("error computing relative path from %s to %s: %v", rootDir, submoduleDir, err)
	}

	// Ensure relative path starts with ./ or ../
	if !strings.HasPrefix(relPath, "./") && !strings.HasPrefix(relPath, "../") {
		relPath = "./" + relPath
	}

	res, err := exec.Command("git", "-C", rootDir, "submodule", "add", relPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding git submodule for dir: %s, err: %v, output: %s", submoduleDir, err, string(res))
	}

	return nil
}

func UpdateAndInitSubmodules(rootDir string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "-C", rootDir, "submodule", "update", "--init", "--recursive").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error updating git submodules for dir: %s, err: %v, output: %s", rootDir, err, string(res))
	}

	return nil
}

func GitAddAndCommit(dir, message string, lockMutex bool) error {
	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	err := GitAdd(dir, ".", false)
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
	}

	err = GitCommit(dir, message, false)
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func GitAdd(repoDir, path string, lockMutex bool) error {
	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	res, err := exec.Command("git", "-C", repoDir, "add", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}

	return nil
}

func GitCommit(repoDir, commitMsg string, lockMutex bool) error {
	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	res, err := exec.Command("git", "-C", repoDir, "commit", "-m", commitMsg, "--allow-empty").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}

	return nil
}

func CheckUncommittedChanges() (bool, error) {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	// Check if there are any changes
	res, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error checking for uncommitted changes: %v, output: %s", err, string(res))
	}

	// If there's output, there are uncommitted changes
	return strings.TrimSpace(string(res)) != "", nil
}

func GitStashCreate(message string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "stash", "push", "-m", message).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating git stash: %v, output: %s", err, string(res))
	}

	return nil
}

// this matches output for git version 2.39.3
// need to test on other versions and check for more variations
// there isn't any structured way to get stash conflicts from git, unfortunately
const PopStashConflictMsg = "overwritten by merge"
const ConflictMsgFilesEnd = "commit your changes"

func GitStashPop(conflictStrategy types.PlanOutdatedStrategy) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "stash", "pop").CombinedOutput()

	if err != nil {
		if strings.Contains(string(res), PopStashConflictMsg) {
			if conflictStrategy != types.PlanOutdatedStrategyOverwrite {
				return fmt.Errorf("conflict popping git stash with unsupported conflict strategy: %s", conflictStrategy)
			}

			// Parse the output to find which files have conflicts
			conflictFiles := parseConflictFiles(string(res))
			for _, file := range conflictFiles {
				// Reset each conflicting file individually
				err = exec.Command("git", "checkout", "--ours", file).Run()
				if err != nil {
					return fmt.Errorf("error resetting file %s: %v", file, err)
				}
			}
			err = exec.Command("git", "stash", "drop").Run()
			if err != nil {
				return fmt.Errorf("error dropping git stash: %v", err)
			}
			return nil
		} else {
			return fmt.Errorf("error popping git stash: %v", err)
		}
	}

	return nil
}

func parseConflictFiles(gitOutput string) []string {
	var conflictFiles []string
	lines := strings.Split(gitOutput, "\n")

	inFilesSection := false

	for _, line := range lines {
		if inFilesSection {
			file := strings.TrimSpace(line)
			if file == "" {
				continue
			}
			conflictFiles = append(conflictFiles, strings.TrimSpace(line))
		} else if strings.Contains(line, PopStashConflictMsg) {
			inFilesSection = true
		} else if strings.Contains(line, ConflictMsgFilesEnd) {
			break
		}
	}
	return conflictFiles
}

func CwdIsGitRepo() bool {
	isGitRepo := false
	if IsCommandAvailable("git") {
		// check whether we're in a git repo
		_, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
		if err == nil {
			isGitRepo = true
		}
	}

	return isGitRepo
}

func gitCommitRootUpdate(commitMsg string, lockMutex bool) error {
	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	err := GitAdd(CurrentPlanDir, ".", false)
	if err != nil {
		return fmt.Errorf("failed to root plan dir changes: %s", err)
	}

	// Commit these staged submodule changes in the root repo
	err = GitCommit(CurrentPlanDir, commitMsg, false)
	if err != nil {
		return fmt.Errorf("failed to commit submodule updates in root dir: %s", err)
	}

	return nil

}

func GitCommitContextUpdate(commitMsg string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	err := GitAddAndCommit(ContextSubdir, commitMsg, false)

	if err != nil {
		return fmt.Errorf("failed to commit context: %v", err)
	}

	err = GitAdd(CurrentPlanDir, ContextSubdir, false)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in context dir: %s", err)
	}

	return gitCommitRootUpdate(commitMsg, false)
}

func GitCommitConvoUpdate(commitMsg string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	err := GitAddAndCommit(ConversationSubdir, commitMsg, false)

	if err != nil {
		return fmt.Errorf("failed to commit convo update: %v", err)
	}

	err = GitAdd(CurrentPlanDir, ConversationSubdir, false)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in convo dir: %s", err)
	}

	return gitCommitRootUpdate(commitMsg, false)
}

func GitCommitPlanUpdate(commitMsg string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	err := GitAddAndCommit(ResultsSubdir, commitMsg, false)
	if err != nil {
		return fmt.Errorf("failed to commit files to plan dir: %s", err)
	}

	return gitCommitRootUpdate(commitMsg, false)
}

// GetGitCommitHistory retrieves the git commit history for the given directory.
// It returns a formatted string with each commit's short SHA, timestamp, and message.
func GetGitCommitHistory(dir string) (string, error) {
	gitMutex.Lock()
	defer gitMutex.Unlock()

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

// GetLatestCommit retrieves the latest commit SHA, timestamp, and message for the given directory.
// It returns the short SHA along with a formatted commit message.
func GetLatestCommit(dir string) (string, string, error) {
	gitMutex.Lock()
	defer gitMutex.Unlock()

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

// GitRewindSteps reverts the repository by a specified number of steps.
func GitRewindSteps(dir string, steps int, isRoot bool) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "-C", dir, "reset", "--hard",
		fmt.Sprintf("HEAD~%d", steps)).CombinedOutput()

	if err != nil {
		return fmt.Errorf("error executing git reset for dir: %s, steps: %d, err: %v, output: %s", dir, steps, err, string(res))
	}

	if isRoot {
		// Update submodules
		err = updateSubmodules(dir)
		if err != nil {
			return err
		}
	}

	return nil
}

// GitRewindToSHA reverts the repository to a specific commit SHA.
func GitRewindToSHA(dir, sha string, isRoot bool) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "-C", dir, "reset", "--hard",
		sha).CombinedOutput()

	if err != nil {
		return fmt.Errorf("error executing git reset for dir: %s, sha: %s, err: %v, output: %s", dir, sha, err, string(res))
	}

	if isRoot {
		err = updateSubmodules(dir)
		if err != nil {
			return err
		}
	}

	return nil
}

// git mutex will already be locked when this function is called
func updateSubmodules(dir string) error {
	res, err := exec.Command("git", "-C", dir, "submodule", "update", "--recursive", "--init").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error updating git submodules for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return nil
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
