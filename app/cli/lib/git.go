package lib

import (
	"fmt"
	"os/exec"
	"plandex/types"
	"strings"
	"sync"
)

var gitMutex sync.Mutex

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

	res, err := exec.Command("git", "stash", "push", "-u", "-m", message).CombinedOutput()
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

func GitStashPop(conflictStrategy string) error {
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

func GitClearUncommittedChanges() error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "checkout", ".").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error clearing uncommitted changes: %v, output: %s", err, string(res))
	}

	return nil
}
