package db

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/plandex/plandex/shared"
)

func GetPlanDiffs(orgId, planId string) (string, error) {
	planState, err := GetCurrentPlanState(CurrentPlanStateParams{
		OrgId:  orgId,
		PlanId: planId,
	})

	if err != nil {
		return "", fmt.Errorf("error getting current plan state: %v", err)
	}

	// create temp directory
	tempDirPath, err := os.MkdirTemp(getOrgDir(orgId), "tmp-diffs-*")

	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}

	defer func() {
		go os.RemoveAll(tempDirPath)
	}()

	// init a git repo in the temp dir
	err = initGitRepo(tempDirPath)

	if err != nil {
		return "", fmt.Errorf("error initializing git repo: %v", err)
	}

	files := planState.CurrentPlanFiles.Files

	// write the original files to the temp dir
	errCh := make(chan error, len(planState.ContextsByPath))
	hasAnyOriginal := false

	for path, context := range planState.ContextsByPath {
		go func(path string, context *shared.Context) {
			_, hasPath := files[path]
			if hasPath {
				hasAnyOriginal = true
				// ensure file directory exists
				err = os.MkdirAll(filepath.Dir(filepath.Join(tempDirPath, path)), 0755)
				if err != nil {
					errCh <- fmt.Errorf("error creating directory: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(tempDirPath, path), []byte(context.Body), 0644)
				if err != nil {
					errCh <- fmt.Errorf("error writing file: %v", err)
					return
				}
			}
			errCh <- nil
		}(path, context)
	}

	for range planState.ContextsByPath {
		err = <-errCh
		if err != nil {
			return "", fmt.Errorf("error writing original files to temp dir: %v", err)
		}
	}

	if hasAnyOriginal {
		// add and commit the files in the temp dir
		err := gitAdd(tempDirPath, ".")
		if err != nil {
			return "", fmt.Errorf("error adding files to git repository for dir: %s, err: %v", tempDirPath, err)
		}

		err = gitCommit(tempDirPath, "original files")
		if err != nil {
			return "", fmt.Errorf("error committing files to git repository for dir: %s, err: %v", tempDirPath, err)
		}
	}

	// write the current files to the temp dir
	errCh = make(chan error, len(files))

	for path, file := range files {
		go func(path, file string) {
			// ensure file directory exists
			err = os.MkdirAll(filepath.Dir(filepath.Join(tempDirPath, path)), 0755)
			if err != nil {
				errCh <- fmt.Errorf("error creating directory: %v", err)
				return
			}

			err = os.WriteFile(filepath.Join(tempDirPath, path), []byte(file), 0644)
			if err != nil {
				errCh <- fmt.Errorf("error writing file: %v", err)
				return
			}
			errCh <- nil
		}(path, file)
	}

	for range files {
		err = <-errCh
		if err != nil {
			return "", fmt.Errorf("error writing current files to temp dir: %v", err)
		}
	}

	err = gitAdd(tempDirPath, ".")
	if err != nil {
		return "", fmt.Errorf("error adding files to git repository for dir: %s, err: %v", tempDirPath, err)
	}

	res, err := exec.Command("git", "-C", tempDirPath, "diff", "--cached", "--color=always").CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("error getting diffs: %v", err)
	}

	return string(res), nil
}

func GetDiffsForBuild(original, updated string) (string, error) {
	// create temp directory
	tempDirPath, err := os.MkdirTemp("", "tmp-diffs-*")

	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}

	defer func() {
		go os.RemoveAll(tempDirPath)
	}()

	// write the original file to the temp dir
	err = os.WriteFile(filepath.Join(tempDirPath, "original"), []byte(original), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing original file: %v", err)
	}

	// write the updated file to the temp dir
	err = os.WriteFile(filepath.Join(tempDirPath, "updated"), []byte(updated), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing updated file: %v", err)
	}

	res, err := exec.Command("git", "-C", tempDirPath, "diff", "--no-color", "--no-index", "original", "updated").CombinedOutput()

	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok && exitError.ExitCode() == 1 {
			// Exit status 1 means diffs were found, which is expected
		} else {
			log.Printf("Error getting diffs: %v\n", err)
			log.Printf("Diff output: %s\n", res)
			return "", fmt.Errorf("error getting diffs: %v", err)
		}
	}

	return string(res), nil
}
