package db

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	maxGitRetries     = 5
	baseGitRetryDelay = 100 * time.Millisecond
)

func init() {
	// ensure git is available
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("error running git --version: %v", err))
	}
}

type GitRepo struct {
	orgId  string
	planId string
}

func InitGitRepo(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	return initGitRepo(dir)
}

func initGitRepo(dir string) error {
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

func getGitRepo(orgId, planId string) *GitRepo {
	return &GitRepo{
		orgId:  orgId,
		planId: planId,
	}
}

func (repo *GitRepo) GitAddAndCommit(branch, message string) error {
	log.Printf("[Git] GitAddAndCommit - orgId: %s, planId: %s, branch: %s, message: %s", repo.orgId, repo.planId, branch, message)
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitAdd(dir, ".")
	}, dir, fmt.Sprintf("GitAddAndCommit > gitAdd: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
	}

	err = gitWriteOperation(func() error {
		return gitCommit(dir, message)
	}, dir, fmt.Sprintf("GitAddAndCommit > gitCommit: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	// log.Println("[Git] GitAddAndCommit - finished, logging repo state")

	// repo.LogGitRepoState()

	return nil
}

func (repo *GitRepo) GitRewindToSha(branch, sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitRewindToSha(dir, sha)
	}, dir, fmt.Sprintf("GitRewindToSha > gitRewindToSha: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error rewinding git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func (repo *GitRepo) GetCurrentCommitSha() (sha string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting current commit SHA for dir: %s, err: %v", dir, err)
	}

	sha = strings.TrimSpace(string(output))
	return sha, nil
}

func (repo *GitRepo) GetCommitTime(branch, ref string) (time.Time, error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	// Use git show to get the commit timestamp
	cmd := exec.Command("git", "-C", dir, "show", "-s", "--format=%ct", ref)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting commit time for ref %s: %v", ref, err)
	}

	// Parse the Unix timestamp
	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing commit timestamp for ref %s: %v", ref, err)
	}

	// Convert Unix timestamp to time.Time
	commitTime := time.Unix(timestamp, 0)
	return commitTime, nil
}

func (repo *GitRepo) GitResetToSha(sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "reset", "--hard", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, dir, fmt.Sprintf("GitResetToSha > gitReset: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func (repo *GitRepo) GitCheckoutSha(sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "checkout", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, dir, fmt.Sprintf("GitCheckoutSha > gitCheckout: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func (repo *GitRepo) GetGitCommitHistory(branch string) (body string, shas []string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	body, shas, err = getGitCommitHistory(dir)
	if err != nil {
		return "", nil, fmt.Errorf("error getting git history for dir: %s, err: %v", dir, err)
	}

	return body, shas, nil
}

func (repo *GitRepo) GetLatestCommit(branch string) (sha, body string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	sha, body, err = getLatestCommit(dir)
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit for dir: %s, err: %v", dir, err)
	}

	return sha, body, nil
}

func (repo *GitRepo) GetLatestCommitShaBeforeTime(branch string, before time.Time) (sha string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	log.Printf("ADMIN - GetLatestCommitShaBeforeTime - dir: %s, before: %s", dir, before.Format("2006-01-02T15:04:05Z"))

	// Round up to the next second
	// roundedTime := before.Add(time.Second).Truncate(time.Second)

	gitFormattedTime := before.Format("2006-01-02 15:04:05+0000")

	// log.Printf("ADMIN - Git formatted time: %s", gitFormattedTime)

	cmd := exec.Command("git", "-C", dir, "log", "-n", "1",
		"--before="+gitFormattedTime,
		"--pretty=%h@@|@@%B@>>>@")
	log.Printf("ADMIN - Executing command: %s", cmd.String())
	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting latest commit before time for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	// log.Printf("ADMIN - git log res: %s", string(res))

	output := strings.TrimSpace(string(res))

	// history := processGitHistoryOutput(strings.TrimSpace(string(res)))

	// log.Printf("ADMIN - History: %v", history)

	if output == "" {
		return "", fmt.Errorf("no commits found before time: %s", before.Format("2006-01-02T15:04:05Z"))
	}

	sha = strings.Split(output, "@@|@@")[0]
	return sha, nil
}

func (repo *GitRepo) GitListBranches() ([]string, error) {
	orgId := repo.orgId
	planId := repo.planId

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

func (repo *GitRepo) GitCreateBranch(newBranch string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "checkout", "-b", newBranch).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error creating git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, dir, fmt.Sprintf("GitCreateBranch > gitCheckout: plan=%s branch=%s", planId, newBranch))

	if err != nil {
		return err
	}

	return nil
}

func (repo *GitRepo) GitDeleteBranch(branchName string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "branch", "-D", branchName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error deleting git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, dir, fmt.Sprintf("GitDeleteBranch > gitBranch: plan=%s branch=%s", planId, branchName))

	if err != nil {
		return err
	}

	return nil
}

func (repo *GitRepo) GitClearUncommittedChanges(branch string) error {
	orgId := repo.orgId
	planId := repo.planId

	log.Printf("[Git] GitClearUncommittedChanges - orgId: %s, planId: %s, branch: %s", orgId, planId, branch)

	dir := getPlanDir(orgId, planId)

	// first do a lightweight git status to check if there are any uncommitted changes
	// prevents heavier operations below if there are no changes (the usual case)
	res, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking for uncommitted changes: %v, output: %s", err, string(res))
	}

	// If there's output, there are uncommitted changes
	hasChanges := strings.TrimSpace(string(res)) != ""

	if !hasChanges {
		log.Printf("[Git] GitClearUncommittedChanges - no changes to clear for plan %s", planId)
		return nil
	}

	err = gitWriteOperation(func() error {
		// Reset staged changes
		log.Printf("[Git] GitClearUncommittedChanges - resetting staged changes for plan %s", planId)
		res, err := exec.Command("git", "-C", dir, "reset", "--hard").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error resetting staged changes | err: %v, output: %s", err, string(res))
		}
		log.Printf("[Git] GitClearUncommittedChanges - reset staged changes finished for plan %s", planId)
		return nil
	}, dir, fmt.Sprintf("GitClearUncommittedChanges > gitReset: plan=%s", planId))

	if err != nil {
		return err
	}

	err = gitWriteOperation(func() error {
		// Clean untracked files
		log.Printf("[Git] GitClearUncommittedChanges - cleaning untracked files for plan %s", planId)
		res, err := exec.Command("git", "-C", dir, "clean", "-d", "-f").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error cleaning untracked files | err: %v, output: %s", err, string(res))
		}
		log.Printf("[Git] GitClearUncommittedChanges - clean untracked files finished for plan %s", planId)
		return nil
	}, dir, fmt.Sprintf("GitClearUncommittedChanges > gitClean: plan=%s", planId))

	return err
}

func (repo *GitRepo) GitCheckoutBranch(branch string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitCheckoutBranch(dir, branch)
	}, dir, fmt.Sprintf("GitCheckoutBranch > gitCheckout: plan=%s branch=%s", planId, branch))

	if err != nil {
		return err
	}

	return nil
}

func gitAdd(repoDir, path string) error {

	if err := gitRemoveIndexLockFileIfExists(repoDir); err != nil {
		return fmt.Errorf("error removing lock file before add: %v", err)
	}

	res, err := exec.Command("git", "-C", repoDir, "add", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}
	return nil
}

func gitCommit(repoDir, commitMsg string) error {
	if err := gitRemoveIndexLockFileIfExists(repoDir); err != nil {
		return fmt.Errorf("error removing lock file before commit: %v", err)
	}

	res, err := exec.Command("git", "-C", repoDir, "commit", "-m", commitMsg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}
	return nil

}

func gitCheckoutBranch(repoDir, branch string) error {
	log.Printf("[Git] gitCheckoutBranch - repoDir: %s, branch: %s", repoDir, branch)
	if err := gitRemoveIndexLockFileIfExists(repoDir); err != nil {
		return fmt.Errorf("error removing lock file before checkout: %v", err)
	}

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
	log.Printf("[Git] gitCheckoutBranch - currentBranch: %s", currentBranch)

	if currentBranch == branch {
		log.Printf("[Git] gitCheckoutBranch - already on branch %s, skipping", branch)
		return nil
	}

	log.Println("[Git] gitCheckoutBranch - checking out branch:", branch)
	res, err := exec.Command("git", "-C", repoDir, "checkout", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking out git branch for dir: %s, err: %v, output: %s", repoDir, err, string(res))
	}
	return nil
}

func gitRewindToSha(repoDir, sha string) error {
	res, err := exec.Command("git", "-C", repoDir, "reset", "--hard", sha).CombinedOutput()
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
		return "", "", fmt.Errorf("error getting git history for dir: %s, err: %v", dir, err)
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
		return "", nil, fmt.Errorf("error getting git history for dir: %s, err: %v", dir, err)
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

func removeLockFile(lockFilePath string) error {
	_, err := os.Stat(lockFilePath)
	exists := err == nil
	// log.Println("index.lock file exists:", exists)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking lock file: %v", err)
	}

	attempts := 0
	for exists {
		if attempts > 10 {
			return fmt.Errorf("error removing index.lock file: %v after %d attempts", err, attempts)
		}

		log.Printf("[Git] removeLockFile - removing index.lock file: %s, attempt: %d", lockFilePath, attempts)

		if err := os.Remove(lockFilePath); err != nil {
			if os.IsNotExist(err) {
				log.Printf("[Git] removeLockFile - %s file not found, skipping removal", lockFilePath)
				return nil
			}

			return fmt.Errorf("error removing lock file: %v", err)
		}

		_, err = os.Stat(lockFilePath)
		exists = err == nil

		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error checking lock file: %v", err)
		}

		log.Printf("[Git] removeLockFile - after removal, %s file exists: %t", lockFilePath, exists)
		if exists {
			log.Printf("[Git] removeLockFile - %s file still exists, retrying after delay", lockFilePath)
		} else {
			log.Printf("[Git] removeLockFile - %s file removed successfully", lockFilePath)
			return nil
		}

		attempts++
		time.Sleep(20 * time.Millisecond)
	}

	return nil
}

func gitRemoveIndexLockFileIfExists(repoDir string) error {
	log.Printf("[Git] gitRemoveIndexLockFileIfExists - repoDir: %s", repoDir)

	paths := []string{
		filepath.Join(repoDir, ".git", "index.lock"),
		filepath.Join(repoDir, ".git", "refs", "heads", "HEAD.lock"),
		filepath.Join(repoDir, ".git", "HEAD.lock"),
	}

	errCh := make(chan error, len(paths))

	for _, path := range paths {
		go func(path string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in gitRemoveIndexLockFileIfExists: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in gitRemoveIndexLockFileIfExists: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			if err := removeLockFile(path); err != nil {
				errCh <- err
				return
			}
			errCh <- nil
		}(path)
	}

	errs := []error{}
	for i := 0; i < len(paths); i++ {
		err := <-errCh
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("error removing lock files: %v", errs)
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

func gitWriteOperation(operation func() error, repoDir, label string) error {
	log.Printf("[Git] gitWriteOperation - label: %s", label)
	var err error
	for attempt := 0; attempt < maxGitRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseGitRetryDelay // Exponential backoff
			time.Sleep(delay)
			log.Printf("Retry attempt %d for git operation %s (delay: %v)\n", attempt+1, label, delay)
		}

		err = operation()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if strings.Contains(err.Error(), "index.lock") || strings.Contains(err.Error(), "cannot lock ref") {
			log.Printf("Git lock file error detected for %s, will retry: %v\n", label, err)
			err = gitRemoveIndexLockFileIfExists(repoDir)
			if err != nil {
				log.Printf("error removing lock files: %v", err)
			}
			continue
		}

		// Non-retryable error
		return err
	}
	return fmt.Errorf("operation %s failed after %d attempts: %v", label, maxGitRetries, err)
}

// LogGitRepoState prints out useful debug info about the current git repository:
//   - The currently checked-out branch
//   - The last few commits
//   - The status (untracked changes, etc.)
//   - A directory listing of refs/heads
//   - A directory listing of .git/ (to spot any leftover lock files or HEAD files)
func (repo *GitRepo) LogGitRepoState() {
	repoDir := getPlanDir(repo.orgId, repo.planId)

	log.Println("[DEBUG] --- Git Repo State ---")

	// 1. Current branch
	out, err := exec.Command("git", "-C", repoDir, "branch", "--show-current").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git branch --show-current`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Current branch: %s", string(out))
	}

	// 2. Recent commits
	out, err = exec.Command("git", "-C", repoDir, "log", "--oneline", "-5").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git log --oneline -5`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Recent commits:\n%s", string(out))
	}

	// 3. Git status
	out, err = exec.Command("git", "-C", repoDir, "status", "--short", "--branch").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git status`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Git status:\n%s", string(out))
	}

	// 4. Show all refs (to see if `.git/refs/heads/HEAD` exists)
	out, err = exec.Command("git", "-C", repoDir, "show-ref").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git show-ref`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] All refs:\n%s", string(out))
	}

	// 5. Directory listing of .git/refs/heads
	headsDir := filepath.Join(repoDir, ".git", "refs", "heads")
	out, err = exec.Command("ls", "-l", headsDir).CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error listing heads dir: %s, err: %v, output: %s", headsDir, err, string(out))
	} else {
		log.Printf("[DEBUG] .git/refs/heads contents:\n%s", string(out))
	}

	// 5a. If there's actually a HEAD file in `.git/refs/heads`, cat it.
	headRefPath := filepath.Join(headsDir, "HEAD")
	if _, err := os.Stat(headRefPath); err == nil {
		// The file `.git/refs/heads/HEAD` exists, which is unusual
		log.Printf("[DEBUG] Found .git/refs/heads/HEAD. Dumping contents:")
		catOut, _ := exec.Command("cat", headRefPath).CombinedOutput()
		log.Printf("[DEBUG] .git/refs/heads/HEAD contents:\n%s", string(catOut))
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for .git/refs/heads/HEAD: %v", err)
	}

	// 6. Directory listing of .git/ in case there's HEAD.lock or index.lock
	gitDir := filepath.Join(repoDir, ".git")
	out, err = exec.Command("ls", "-l", gitDir).CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error listing .git dir: %s, err: %v, output: %s", gitDir, err, string(out))
	} else {
		log.Printf("[DEBUG] .git/ contents:\n%s", string(out))
	}

	// 6a. If there's a .git/HEAD file, cat it
	headFilePath := filepath.Join(gitDir, "HEAD")
	if _, err := os.Stat(headFilePath); err == nil {
		log.Printf("[DEBUG] .git/HEAD file exists. Dumping contents:")
		catOut, _ := exec.Command("cat", headFilePath).CombinedOutput()
		log.Printf("[DEBUG] .git/HEAD contents:\n%s", string(catOut))
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for .git/HEAD: %v", err)
	}

	// 6b. Check for HEAD.lock or index.lock specifically
	headLockPath := filepath.Join(gitDir, "HEAD.lock")
	if _, err := os.Stat(headLockPath); err == nil {
		log.Printf("[DEBUG] HEAD.lock file exists at: %s", headLockPath)
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for HEAD.lock: %v", err)
	}

	indexLockPath := filepath.Join(gitDir, "index.lock")
	if _, err := os.Stat(indexLockPath); err == nil {
		log.Printf("[DEBUG] index.lock file exists at: %s", indexLockPath)
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for index.lock: %v", err)
	}

	log.Println("[DEBUG] --- End Git Repo State ---")
}
