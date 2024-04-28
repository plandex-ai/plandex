package db

import (
	"fmt"
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
