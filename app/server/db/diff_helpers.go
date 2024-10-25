package db

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

func GetPlanDiffs(orgId, planId string, plain bool) (string, error) {
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

	colorArg := "--color=always"
	if plain {
		colorArg = "--no-color"
	}
	res, err := exec.Command("git", "-C", tempDirPath, "diff", "--cached", colorArg).CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("error getting diffs: %v", err)
	}

	return string(res), nil
}

func GetDiffs(original, updated string) (string, error) {
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

	cmd := exec.Command("git", "-C", tempDirPath, "diff", "--no-color", "--no-index", "original", "updated")

	res, err := cmd.CombinedOutput()

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

type change struct {
	Old    string
	New    string
	Line   int
	Length int
}

func GetDiffReplacements(original, updated string) ([]*shared.Replacement, error) {
	diff, err := GetDiffs(original, updated)
	if err != nil {
		return nil, fmt.Errorf("error getting git diffs: %v", err)
	}

	var changes []*change
	scanner := bufio.NewScanner(strings.NewReader(diff))

	var currentHunk *change
	var oldLines, newLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse hunk header
		if strings.HasPrefix(line, "@@") {
			// If we have a previous hunk, process it
			if currentHunk != nil {
				change := processHunk(oldLines, newLines, currentHunk.Line)
				if change != nil {
					changes = append(changes, change)
				}
			}

			// Parse the new hunk header
			lineInfo := strings.Split(line, " ")[1:] // Skip @@ part
			oldInfo := strings.Split(lineInfo[0], ",")
			startLine, _ := strconv.Atoi(strings.TrimPrefix(oldInfo[0], "-"))

			currentHunk = &change{
				Line: startLine,
			}
			oldLines = []string{}
			newLines = []string{}
			continue
		}

		if currentHunk == nil {
			continue // Skip until we find a hunk
		}

		// Process the lines within a hunk
		switch {
		case strings.HasPrefix(line, "-"):
			oldLines = append(oldLines, strings.TrimPrefix(line, "-"))
		case strings.HasPrefix(line, "+"):
			newLines = append(newLines, strings.TrimPrefix(line, "+"))
		case strings.HasPrefix(line, " "):
			// Context lines - add to both
			line = strings.TrimPrefix(line, " ")
			oldLines = append(oldLines, line)
			newLines = append(newLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning diff: %v", err)
	}

	// Process the last hunk if exists
	if currentHunk != nil {
		change := processHunk(oldLines, newLines, currentHunk.Line)
		if change != nil {
			changes = append(changes, change)
		}
	}

	replacements := make([]*shared.Replacement, len(changes))
	for i, change := range changes {
		replacements[i] = &shared.Replacement{
			Id:  uuid.New().String(),
			Old: change.Old,
			New: change.New,
		}
	}

	return replacements, nil
}

func processHunk(oldLines, newLines []string, startLine int) *change {
	if len(oldLines) == 0 && len(newLines) == 0 {
		return nil
	}

	return &change{
		Old:    strings.Join(oldLines, "\n"),
		New:    strings.Join(newLines, "\n"),
		Line:   startLine,
		Length: len(oldLines),
	}
}
