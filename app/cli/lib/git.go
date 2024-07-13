package lib

import (
	"fmt"
	"log"
	"os/exec"
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

	err = GitCommit(dir, message, nil, false)
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func GitAddAndCommitPaths(dir, message string, paths []string, lockMutex bool) error {
	if len(paths) == 0 {
		return nil
	}

	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	for _, path := range paths {
		err := GitAdd(dir, path, false)
		if err != nil {
			return fmt.Errorf("error adding file %s to git repository for dir: %s, err: %v", path, dir, err)
		}
	}

	err := GitCommit(dir, message, paths, false)
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

func GitCommit(repoDir, commitMsg string, paths []string, lockMutex bool) error {
	if lockMutex {
		gitMutex.Lock()
		defer gitMutex.Unlock()
	}

	args := []string{"-C", repoDir, "commit", "-m", commitMsg, "--allow-empty"}

	if len(paths) > 0 {
		args = append(args, paths...)
	}

	res, err := exec.Command("git", args...).CombinedOutput()
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

	res, err := exec.Command("git", "stash", "push", "--include-untracked", "-m", message).CombinedOutput()
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

func GitStashPop(forceOverwrite bool) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "stash", "pop").CombinedOutput()

	// we should no longer have conflicts since we are forcing an update before
	// running the 'apply' command as well as resetting any files with uncommitted change
	// still leaving this though in case something goes wrong

	if err != nil {
		log.Println("Error popping git stash:", string(res))

		if strings.Contains(string(res), PopStashConflictMsg) {
			log.Println("Conflicts detected")

			if !forceOverwrite {
				return fmt.Errorf("conflict popping git stash: %s", string(res))
			}

			// Parse the output to find which files have conflicts
			conflictFiles := parseConflictFiles(string(res))

			log.Println("Conflicting files:", conflictFiles)

			for _, file := range conflictFiles {
				// Reset each conflicting file individually
				checkoutRes, err := exec.Command("git", "checkout", "--ours", file).CombinedOutput()
				if err != nil {
					return fmt.Errorf("error resetting file %s: %v", file, string(checkoutRes))
				}
			}
			dropRes, err := exec.Command("git", "stash", "drop").CombinedOutput()
			if err != nil {
				return fmt.Errorf("error dropping git stash: %v", string(dropRes))
			}
			return nil
		} else {
			log.Println("No conflicts detected")

			return fmt.Errorf("error popping git stash: %v", string(res))
		}
	}

	return nil
}

func GitClearUncommittedChanges() error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	// Reset staged changes
	res, err := exec.Command("git", "reset", "--hard").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error resetting staged changes | err: %v, output: %s", err, string(res))
	}

	// Clean untracked files
	res, err = exec.Command("git", "clean", "-d", "-f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error cleaning untracked files | err: %v, output: %s", err, string(res))
	}

	return nil
}

func GitFileHasUncommittedChanges(path string) (bool, error) {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "status", "--porcelain", path).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error checking for uncommitted changes for file %s | err: %v, output: %s", path, err, string(res))
	}

	return strings.TrimSpace(string(res)) != "", nil
}

func GitCheckoutFile(path string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "checkout", path).CombinedOutput()
	if err != nil {
		log.Println("Error checking out file:", string(res))

		return fmt.Errorf("error checking out file %s | err: %v, output: %s", path, err, string(res))
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
