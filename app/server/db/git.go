package db

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/lib/pq"
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

	res, err := exec.Command("git", "init", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository for dir: %s, err: %v, output: %s", dir, err, string(res))
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

	if currentBranch == branch {
		return nil
	}

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

// distributed locking to ensure only one user can write to a plan repo at a time
// multiple readers are allowed, but read locks block writes
// write lock is exclusive (blocks both reads and writes)
const lockTimeout = 60 * time.Second

func LockRepo(orgId, userId, planId, branch string, scope LockScope) (string, error) {
	return lockRepo(orgId, userId, planId, "", branch, scope, 0)
}

func LockRepoForBuild(orgId, userId, planId, planBuildId, branch string, scope LockScope) (string, error) {
	return lockRepo(orgId, userId, planId, planBuildId, branch, scope, 0)
}

func lockRepo(orgId, userId, planId, planBuildId, branch string, scope LockScope, numRetry int) (string, error) {
	tx, err := Conn.Begin()
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	query := "SELECT id, org_id, user_id, plan_id, plan_build_id, scope, branch, created_at FROM repo_locks WHERE plan_id = $1 FOR UPDATE"
	queryArgs := []interface{}{planId}

	var locks []*repoLock

	fn := func() error {
		rows, err := tx.Query(query, queryArgs...)
		if err != nil {
			return fmt.Errorf("error getting repo locks: %v", err)
		}

		defer rows.Close()

		var expiredLockIds []string

		now := time.Now()
		for rows.Next() {
			var lock repoLock
			if err := rows.Scan(&lock.Id, &lock.OrgId, &lock.UserId, &lock.PlanId, &lock.PlanBuildId, &lock.Scope, &lock.Branch, &lock.CreatedAt); err != nil {
				return fmt.Errorf("error scanning repo lock: %v", err)
			}
			if now.Sub(lock.CreatedAt) < lockTimeout {
				locks = append(locks, &lock)
			} else {
				expiredLockIds = append(expiredLockIds, lock.Id)
			}
		}

		if len(expiredLockIds) > 0 {
			query := "DELETE FROM repo_locks WHERE id = ANY($1)"

			args := make([]interface{}, len(expiredLockIds))
			for i, id := range expiredLockIds {
				args[i] = id
			}

			// Execute the query with the slice as a parameter
			_, err := tx.Exec(query, pq.Array(expiredLockIds))
			if err != nil {
				return fmt.Errorf("error removing expired locks: %v", err)
			}
		}

		return nil
	}
	if err := fn(); err != nil {
		return "", err
	}

	canAcquire := true
	canRetry := true

	// log.Println("locks:")
	// spew.Dump(locks)

	for _, lock := range locks {
		lockBranch := ""
		if lock.Branch != nil {
			lockBranch = *lock.Branch
		}

		if scope == LockScopeRead {
			canAcquireThisLock := lock.Scope == LockScopeRead && lockBranch == branch
			if !canAcquireThisLock {
				canAcquire = false
			}
		} else if scope == LockScopeWrite {
			canAcquire = false

			// if lock is for the same plan plan and branch, allow parallel writes
			if planId == lock.PlanId && branch == lockBranch {
				canAcquire = true
			}

			if lock.Scope == LockScopeWrite && lockBranch == branch {
				canRetry = false
			}
		} else {
			err = fmt.Errorf("invalid lock scope: %v", scope)
			return "", err
		}
	}

	if !canAcquire {
		if canRetry {
			// 10 second timeout
			if numRetry > 20 {
				err = fmt.Errorf("plan is currently being updated by another user")
				return "", err
			}
			time.Sleep(500 * time.Millisecond)
			return lockRepo(orgId, userId, planId, planBuildId, branch, scope, numRetry+1)
		}
		err = fmt.Errorf("plan is currently being updated by another user")
		return "", err
	}

	// Insert the new lock
	var lockPlanBuildId *string
	if planBuildId != "" {
		lockPlanBuildId = &planBuildId
	}

	var lockBranch *string
	if branch != "" {
		lockBranch = &branch
	}

	newLock := &repoLock{
		OrgId:       orgId,
		UserId:      userId,
		PlanId:      planId,
		PlanBuildId: lockPlanBuildId,
		Scope:       scope,
		Branch:      lockBranch,
	}
	// log.Println("newLock:")
	// spew.Dump(newLock)

	insertQuery := "INSERT INTO repo_locks (org_id, user_id, plan_id, plan_build_id, scope, branch) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id"
	err = tx.QueryRow(
		insertQuery,
		newLock.OrgId,
		newLock.UserId,
		newLock.PlanId,
		newLock.PlanBuildId,
		newLock.Scope,
		newLock.Branch,
	).Scan(&newLock.Id)
	if err != nil {
		return "", fmt.Errorf("error inserting new lock: %v", err)
	}

	if branch != "" {
		// checkout the branch
		err = gitCheckoutBranch(getPlanDir(orgId, planId), branch)
		if err != nil {
			return "", fmt.Errorf("error checking out branch: %v", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("error committing transaction: %v", err)
	}

	return newLock.Id, nil
}

func UnlockRepo(id string) error {
	query := "DELETE FROM repo_locks WHERE id = $1"
	_, err := Conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error removing lock: %v", err)
	}

	return nil
}
