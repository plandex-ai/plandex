package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/fs"
	shared "plandex-shared"
	"sort"
	"sync"
	"time"
)

// GetUndonePlanApplies returns the list of PlanApplies that will be undone by rewinding to targetSHA.
// An apply is considered "undone" if its timestamp is after OR equal to the target SHA's timestamp,
// since we want to revert to the state before the target SHA.
func GetUndonePlanApplies(currentState *shared.CurrentPlanState, timestamp time.Time) []*shared.PlanApply {
	if currentState == nil {
		return nil
	}

	var undoneApplies []*shared.PlanApply
	for _, apply := range currentState.PlanApplies {
		// Include applies after OR equal to target time
		if !apply.CreatedAt.Before(timestamp) {
			undoneApplies = append(undoneApplies, apply)
		}
	}

	// Sort by creation time ascending to ensure proper order
	sort.Slice(undoneApplies, func(i, j int) bool {
		return undoneApplies[i].CreatedAt.Before(undoneApplies[j].CreatedAt)
	})

	return undoneApplies
}

// GetAffectedFilePaths extracts the set of file paths that were modified by the given PlanApplies.
// It looks up each PlanFileResultId in the current state to get the actual file paths.
func GetAffectedFilePaths(currentState *shared.CurrentPlanState, applies []*shared.PlanApply) map[string]bool {
	if currentState == nil || currentState.PlanResult == nil {
		return nil
	}

	// First collect all file result IDs
	fileResultIds := make(map[string]bool)
	for _, apply := range applies {
		if apply == nil {
			continue
		}
		for _, fileId := range apply.PlanFileResultIds {
			if fileId != "" {
				fileResultIds[fileId] = true
			}
		}
	}

	// Then get the actual file paths from the plan result
	affectedPaths := make(map[string]bool)
	for _, result := range currentState.PlanResult.Results {
		if result == nil {
			continue
		}

		// Skip if this result wasn't part of an undone apply
		if !fileResultIds[result.Id] {
			continue
		}

		// Skip if the result was rejected
		if result.RejectedAt != nil {
			continue
		}

		// Skip if the result was never applied
		if result.AppliedAt == nil {
			continue
		}

		// Validate path
		if result.Path == "" {
			continue
		}

		// Check if path is in plan context
		if currentState.ContextsByPath[result.Path] == nil {
			continue
		}

		affectedPaths[result.Path] = true
	}

	return affectedPaths
}

// RewindAnalysis captures all the information about a potential rewind operation
type RewindAnalysis struct {
	// Files that need to be modified when rewinding from current plan state to target plan state
	RequiredChanges map[string]string
	// Files that have been modified on disk relative to current plan state (potential conflicts)
	Conflicts map[string]bool
}

// AnalyzeRewind examines the three states (disk, current plan, target plan) to determine:
// 1. What files need to be changed to reach target state
// 2. Which of those changes would conflict with user modifications
func AnalyzeRewind(targetState, currentState *shared.CurrentPlanState) (*RewindAnalysis, error) {
	if targetState == nil || currentState == nil {
		return nil, fmt.Errorf("both target and current states must be provided")
	}

	// First determine what files need to be changed between current and target plan states
	requiredChanges := make(map[string]string)

	// Track all paths we need to examine for either changes or conflicts
	allPaths := make(map[string]bool)

	// Add paths from both states
	for path, context := range targetState.ContextsByPath {
		if context.ContextType != shared.ContextFileType {
			continue
		}
		allPaths[path] = true
	}
	for path, context := range currentState.ContextsByPath {
		if context.ContextType != shared.ContextFileType {
			continue
		}
		allPaths[path] = true
	}

	// For each path, check if content differs between current and target states
	for path := range allPaths {
		targetContent := ""
		if ctx := targetState.ContextsByPath[path]; ctx != nil {
			targetContent = ctx.Body
		}

		currentContent := ""
		if ctx := currentState.ContextsByPath[path]; ctx != nil {
			currentContent = ctx.Body
		}

		// If content differs between plan states, this is a required change
		if targetContent != currentContent {
			if targetContent == "" {
				// File should be removed
				requiredChanges[path] = ""
			} else {
				// File should be added or modified
				requiredChanges[path] = targetContent
			}
		}
	}

	// Now check for conflicts by comparing disk state with current plan state
	// A conflict exists if a file that needs to be changed has been modified on disk
	conflicts := make(map[string]bool)

	var mu sync.Mutex
	errCh := make(chan error, len(requiredChanges))

	for path := range requiredChanges {
		go func(path string) {
			var outErr error
			defer func() { errCh <- outErr }()

			// Get the content from current plan state
			currentContent := ""
			if ctx := currentState.ContextsByPath[path]; ctx != nil {
				currentContent = ctx.Body
			}

			// Get the actual file content from disk
			dstPath := filepath.Join(fs.ProjectRoot, path)
			diskContent, err := os.ReadFile(dstPath)
			if err != nil {
				if os.IsNotExist(err) {
					// If file doesn't exist on disk and current state has no content,
					// there's no conflict
					if currentContent == "" {
						return
					}
					// Otherwise it's a conflict because file was deleted
					mu.Lock()
					conflicts[path] = true
					mu.Unlock()
					return
				}
				outErr = fmt.Errorf("failed to read %s: %w", path, err)
				return
			}

			// If disk content differs from current plan state, we have a conflict
			if string(diskContent) != currentContent {
				mu.Lock()
				conflicts[path] = true
				mu.Unlock()
			}
		}(path)
	}

	// Collect any errors from goroutines
	for i := 0; i < len(requiredChanges); i++ {
		if err := <-errCh; err != nil {
			return nil, err
		}
	}

	return &RewindAnalysis{
		RequiredChanges: requiredChanges,
		Conflicts:       conflicts,
	}, nil
}

// RemoveEmptyDirs recursively removes empty directories starting from the given path
func RemoveEmptyDirs(path string, baseDir string) error {
	// Check if the path is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	// List directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// If directory has contents, leave it alone
	if len(entries) > 0 {
		return nil
	}

	// Directory is empty, remove it if it's not the base dir
	if path != baseDir {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	// Recursively check parent directory
	parent := filepath.Dir(path)
	if parent != baseDir && parent != path {
		return RemoveEmptyDirs(parent, baseDir)
	}

	return nil
}

// ApplyRewindChanges updates files on disk to match target state
func ApplyRewindChanges(requiredChanges map[string]string) error {
	if len(requiredChanges) == 0 {
		return nil
	}

	// Track directories that might need cleanup
	dirsToCheck := make(map[string]bool)
	var mu sync.Mutex

	errCh := make(chan error, len(requiredChanges))

	for path, content := range requiredChanges {
		go func(path, content string) {
			dstPath := filepath.Join(fs.ProjectRoot, path)

			if content == "" {
				// Remove the file
				err := os.Remove(dstPath)
				if err != nil && !os.IsNotExist(err) {
					errCh <- fmt.Errorf("failed to remove %s: %w", path, err)
					return
				}
				// Mark parent directory for cleanup
				parentDir := filepath.Dir(dstPath)
				mu.Lock()
				dirsToCheck[parentDir] = true
				mu.Unlock()
				errCh <- nil
				return
			}

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				errCh <- fmt.Errorf("failed to create directory for %s: %w", path, err)
				return
			}

			// Write the file
			if err := os.WriteFile(dstPath, []byte(content), 0644); err != nil {
				errCh <- fmt.Errorf("failed to write %s: %w", path, err)
				return
			}

			errCh <- nil
		}(path, content)
	}

	// Collect any errors
	for i := 0; i < len(requiredChanges); i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}

	// Clean up empty directories
	for dir := range dirsToCheck {
		if err := RemoveEmptyDirs(dir, fs.ProjectRoot); err != nil {
			// Log but don't fail the operation for directory cleanup errors
			fmt.Printf("Warning: failed to clean up directory %s: %v\n", dir, err)
		}
	}

	return nil
}
