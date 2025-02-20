package db

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

func GitAddAndCommit(orgId, planId, branch, message string) error {
	err := assertWriteOrRootLockSet(planId, branch)
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		return gitAdd(dir, ".")
	}, fmt.Sprintf("GitAddAndCommit > gitAdd: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
	}

	err = gitWriteOperation(func() error {
		return gitCommit(dir, message)
	}, fmt.Sprintf("GitAddAndCommit > gitCommit: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func GitRewindToSha(orgId, planId, branch, sha string) error {
	err := assertWriteOrRootLockSet(planId, branch)
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		return gitRewindToSha(dir, sha)
	}, fmt.Sprintf("GitRewindToSha > gitRewindToSha: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error rewinding git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func GetCurrentCommitSha(orgId, planId string) (sha string, err error) {
	err = assertAnyPlanLockSet(planId, "")
	if err != nil {
		return "", err
	}

	dir := getPlanDir(orgId, planId)

	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting current commit SHA for dir: %s, err: %v", dir, err)
	}

	sha = strings.TrimSpace(string(output))
	return sha, nil
}

func GetCommitTime(orgId, planId, branch, ref string) (time.Time, error) {
	err := assertAnyPlanLockSet(planId, branch)
	if err != nil {
		return time.Time{}, err
	}

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

func GitResetToSha(orgId, planId, sha string) error {
	err := assertWriteOrRootLockSet(planId, "")
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "reset", "--hard", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, fmt.Sprintf("GitResetToSha > gitReset: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func GitCheckoutSha(orgId, planId, sha string) error {
	err := assertWriteOrRootLockSet(planId, "")
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "checkout", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, fmt.Sprintf("GitCheckoutSha > gitCheckout: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func GetGitCommitHistory(orgId, planId, branch string) (body string, shas []string, err error) {
	err = assertAnyPlanLockSet(planId, branch)
	if err != nil {
		return "", nil, err
	}

	dir := getPlanDir(orgId, planId)

	body, shas, err = getGitCommitHistory(dir)
	if err != nil {
		return "", nil, fmt.Errorf("error getting git history for dir: %s, err: %v", dir, err)
	}

	return body, shas, nil
}

func GetLatestCommit(orgId, planId, branch string) (sha, body string, err error) {
	err = assertAnyPlanLockSet(planId, branch)
	if err != nil {
		return "", "", err
	}

	dir := getPlanDir(orgId, planId)

	sha, body, err = getLatestCommit(dir)
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit for dir: %s, err: %v", dir, err)
	}

	return sha, body, nil
}

func GetLatestCommitShaBeforeTime(orgId, planId, branch string, before time.Time) (sha string, err error) {
	err = assertAnyPlanLockSet(planId, branch)
	if err != nil {
		return "", err
	}

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

func GitListBranches(orgId, planId string) ([]string, error) {
	err := assertAnyPlanLockSet(planId, "")
	if err != nil {
		return nil, err
	}

	dir := getPlanDir(orgId, planId)

	var out bytes.Buffer
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = dir
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error getting git branches for dir: %s, err: %v", dir, err)
	}

	branches := strings.Split(strings.TrimSpace(out.String()), "\n")

	if len(branches) == 0 || (len(branches) == 1 && branches[0] == "") {
		return []string{"main"}, nil
	}

	return branches, nil
}

func GitCreateBranch(orgId, planId, newBranch string) error {
	err := assertWriteOrRootLockSet(planId, "")
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "checkout", "-b", newBranch).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error creating git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, fmt.Sprintf("GitCreateBranch > gitCheckout: plan=%s branch=%s", planId, newBranch))

	if err != nil {
		return err
	}

	return nil
}

func GitDeleteBranch(orgId, planId, branchName string) error {
	err := assertWriteOrRootLockSet(planId, "")
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "branch", "-D", branchName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error deleting git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, fmt.Sprintf("GitDeleteBranch > gitBranch: plan=%s branch=%s", planId, branchName))

	if err != nil {
		return err
	}

	return nil
}

func GitClearUncommittedChanges(orgId, planId, branch string) error {
	err := assertWriteOrRootLockSet(planId, branch)
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		// Reset staged changes
		res, err := exec.Command("git", "-C", dir, "reset", "--hard").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error resetting staged changes | err: %v, output: %s", err, string(res))
		}
		return nil
	}, fmt.Sprintf("GitClearUncommittedChanges > gitReset: plan=%s", planId))

	if err != nil {
		return err
	}

	err = gitWriteOperation(func() error {
		// Clean untracked files
		res, err := exec.Command("git", "-C", dir, "clean", "-d", "-f").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error cleaning untracked files | err: %v, output: %s", err, string(res))
		}

		return nil
	}, fmt.Sprintf("GitClearUncommittedChanges > gitClean: plan=%s", planId))

	return err
}

func GitCheckoutBranch(orgId, planId, branch string) error {
	// even though we use 'gitWriteOperation' below, we also check out a branch for reads, so we just need to assert that any lock is set
	err := assertAnyPlanLockSet(planId, branch)
	if err != nil {
		return err
	}

	dir := getPlanDir(orgId, planId)

	err = gitWriteOperation(func() error {
		return gitCheckoutBranch(dir, branch)
	}, fmt.Sprintf("GitCheckoutBranch > gitCheckout: plan=%s branch=%s", planId, branch))

	if err != nil {
		return err
	}

	return nil
}

func withRetry(operation func() error) error {
	var err error
	for attempt := 0; attempt < maxGitRetries; attempt++ {
		// Remove any existing lock file before each attempt
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseGitRetryDelay // Exponential backoff
			time.Sleep(delay)
			log.Printf("Retry attempt %d for git operation (delay: %v)\n", attempt+1, delay)
		}

		err = operation()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if strings.Contains(err.Error(), "index.lock") {
			log.Printf("Git lock file error detected, will retry: %v\n", err)
			continue
		}

		// Non-retryable error
		return err
	}
	return fmt.Errorf("operation failed after %d attempts: %v", maxGitRetries, err)
}

func gitAdd(repoDir, path string) error {
	return withRetry(func() error {
		if err := gitRemoveIndexLockFileIfExists(repoDir); err != nil {
			return fmt.Errorf("error removing lock file before add: %v", err)
		}

		res, err := exec.Command("git", "-C", repoDir, "add", path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error adding files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
		}
		return nil
	})
}

func gitCommit(repoDir, commitMsg string) error {
	return withRetry(func() error {
		if err := gitRemoveIndexLockFileIfExists(repoDir); err != nil {
			return fmt.Errorf("error removing lock file before commit: %v", err)
		}

		res, err := exec.Command("git", "-C", repoDir, "commit", "-m", commitMsg).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error committing files to git repository for dir: %s, err: %v, output: %s", repoDir, err, string(res))
		}
		return nil
	})
}

func gitCheckoutBranch(repoDir, branch string) error {
	return withRetry(func() error {
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
	})
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
	log.Println("index.lock file exists:", exists)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking lock file: %v", err)
	}

	attempts := 0
	for exists {
		if attempts > 10 {
			return fmt.Errorf("error removing index.lock file: %v after %d attempts", err, attempts)
		}

		log.Println("removing index.lock file:", lockFilePath, "attempt:", attempts)

		if err := os.Remove(lockFilePath); err != nil {
			if os.IsNotExist(err) {
				log.Println("index.lock file not found, skipping removal")
				return nil
			}

			return fmt.Errorf("error removing lock file: %v", err)
		}

		_, err = os.Stat(lockFilePath)
		exists = err == nil

		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error checking lock file: %v", err)
		}

		log.Println("after removal, index.lock file exists:", exists)
		if exists {
			log.Println("index.lock file still exists, retrying after delay")
		} else {
			log.Println("index.lock file removed successfully")
			return nil
		}

		attempts++
		time.Sleep(20 * time.Millisecond)
	}

	return nil
}

func gitRemoveIndexLockFileIfExists(repoDir string) error {
	paths := []string{
		filepath.Join(repoDir, ".git", "index.lock"),
		filepath.Join(repoDir, ".git", "refs", "heads", "HEAD.lock"),
	}

	errCh := make(chan error, len(paths))

	for _, path := range paths {
		go func(path string) {
			if err := removeLockFile(path); err != nil {
				errCh <- err
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

func gitWriteOperation(operation func() error, label string) error {
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
		if strings.Contains(err.Error(), "index.lock") {
			log.Printf("Git lock file error detected for %s, will retry: %v\n", label, err)
			continue
		}

		// Non-retryable error
		return err
	}
	return fmt.Errorf("operation %s failed after %d attempts: %v", label, maxGitRetries, err)
}

func assertWriteOrRootLockSet(planId, branch string) error {
	planLocksMu.Lock()
	defer planLocksMu.Unlock()

	lock, ok := planLocks[planId]
	if !ok || lock.currentBranch != branch || (!lock.hasWriter && lock.currentBranch != "") {
		stack := debug.Stack()

		if !ok {
			log.Printf("Plan %s branch %s NO LOCK SET for git repo write operation\n%s", planId, branch, formatStackTraceLong(stack))

			return fmt.Errorf("plan %s branch %s NO LOCK SET for git repo write operation", planId, branch)
		} else if lock.currentBranch != branch {
			log.Printf("lock.currentBranch: %s, branch: %s", lock.currentBranch, branch)
			log.Printf("Plan %s branch %s WRONG LOCK SET for git repo write operation\n%s", planId, branch, formatStackTraceLong(stack))

			return fmt.Errorf("plan %s branch %s WRONG LOCK SET for git repo write operation", planId, branch)
		} else if !lock.hasWriter {
			log.Printf("lock.hasWriter: %t", lock.hasWriter)
			log.Printf("Plan %s branch %s NO WRITER LOCK SET for git repo write operation\n%s", planId, branch, formatStackTraceLong(stack))

			return fmt.Errorf("plan %s branch %s NO WRITER LOCK SET for git repo write operation", planId, branch)
		}

	}

	return nil
}

func assertAnyPlanLockSet(planId, branch string) error {
	planLocksMu.Lock()
	defer planLocksMu.Unlock()

	lock, ok := planLocks[planId]
	if !ok {
		stack := debug.Stack()
		log.Printf("Plan %s branch %s NO LOCK SET for git repo operation\n%s", planId, branch, formatStackTraceLong(stack))
		return fmt.Errorf("plan %s NO LOCK SET", planId)
	}

	// if the lock is set on a branch rather than a plan lock, require the branch to match
	if lock.currentBranch != "" && lock.currentBranch != branch {
		stack := debug.Stack()
		log.Printf("Plan %s branch %s WRONG LOCK SET for git repo operation\n%s", planId, branch, formatStackTraceLong(stack))
		return fmt.Errorf("plan %s WRONG LOCK SET", planId)
	}

	// otherwise it's a root plan lock, so we allow any branch
	return nil
}
