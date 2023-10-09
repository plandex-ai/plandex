package lib

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var gitMutex sync.Mutex

func InitGitRepo(dir string) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()

	res, err := exec.Command("git", "init", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	return GitAddAndCommit(dir, "Initial commit")
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

func GitAddAndCommit(dir, message string) error {
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

func gitCommitPlanUpdate(commitMsg string) error {

	err := GitAddAndCommit(ConversationSubdir, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit files to conversation dir: %s\n", err)
	}

	err = GitAddAndCommit(PlanSubdir, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit files to plan dir: %s\n", err)
	}

	// Stage changes in the submodules in the root repo
	err = GitAdd(CurrentPlanRootDir, ConversationSubdir, true)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in conversation dir: %s\n", err)
	}

	err = GitAdd(CurrentPlanRootDir, PlanSubdir, true)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in plan dir: %s\n", err)
	}

	// Commit these staged submodule changes in the root repo
	err = GitCommit(CurrentPlanRootDir, commitMsg, true)
	if err != nil {
		return fmt.Errorf("failed to commit submodule updates in root dir: %s\n", err)
	}

	return nil
}
