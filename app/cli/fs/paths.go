package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plandex-cli/types"
	"strings"
	"sync"

	shared "plandex-shared"

	ignore "github.com/sabhiram/go-gitignore"
)

func GetProjectPaths(baseDir string) (*types.ProjectPaths, error) {
	if ProjectRoot == "" {
		return nil, fmt.Errorf("no project root found")
	}

	return GetPaths(baseDir, ProjectRoot)
}

func GetPaths(baseDir, currentDir string) (*types.ProjectPaths, error) {
	ignored, err := GetPlandexIgnore(currentDir)

	if err != nil {
		return nil, err
	}

	allPaths := map[string]bool{}
	activePaths := map[string]bool{}

	allDirs := map[string]bool{}
	activeDirs := map[string]bool{}
	gitIgnoredDirs := map[string]bool{}

	isGitRepo := IsGitRepo(baseDir)

	errCh := make(chan error)
	var mu sync.Mutex
	numRoutines := 0

	deletedFiles := map[string]bool{}

	if isGitRepo {

		// Use git status to find deleted files
		numRoutines++
		go func() {
			cmd := exec.Command("git", "rev-parse", "--show-toplevel")
			output, err := cmd.Output()
			if err != nil {
				errCh <- fmt.Errorf("error getting git root: %s", err)
				return
			}
			repoRoot := strings.TrimSpace(string(output))

			cmd = exec.Command("git", "status", "--porcelain")
			cmd.Dir = baseDir
			out, err := cmd.Output()
			if err != nil {
				errCh <- fmt.Errorf("error getting git status: %s", err)
			}

			lines := strings.Split(string(out), "\n")

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "D ") {
					path := strings.TrimSpace(line[2:])
					absPath := filepath.Join(repoRoot, path)
					relPath, err := filepath.Rel(currentDir, absPath)
					if err != nil {
						errCh <- fmt.Errorf("error getting relative path: %s", err)
						return
					}
					deletedFiles[relPath] = true
				}
			}

			errCh <- nil
		}()

		// combine `git ls-files` and `git ls-files --others --exclude-standard`
		// to get all files in the repo

		numRoutines++
		go func() {
			// get all tracked files in the repo
			cmd := exec.Command("git", "ls-files")
			cmd.Dir = baseDir
			out, err := cmd.Output()

			if err != nil {
				errCh <- fmt.Errorf("error getting files in git repo: %s", err)
				return
			}

			files := strings.Split(string(out), "\n")

			mu.Lock()
			defer mu.Unlock()
			for _, file := range files {
				absFile := filepath.Join(baseDir, file)
				relFile, err := filepath.Rel(currentDir, absFile)

				if err != nil {
					errCh <- fmt.Errorf("error getting relative path: %s", err)
					return
				}

				if ignored != nil && ignored.MatchesPath(relFile) {
					continue
				}

				activePaths[relFile] = true

				parentDir := relFile
				for parentDir != "." && parentDir != "/" && parentDir != "" {
					parentDir = filepath.Dir(parentDir)
					activeDirs[parentDir] = true
				}
			}

			errCh <- nil
		}()

		// get all untracked non-ignored files in the repo
		numRoutines++
		go func() {
			cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
			cmd.Dir = baseDir
			out, err := cmd.Output()

			if err != nil {
				errCh <- fmt.Errorf("error getting untracked files in git repo: %s", err)
				return
			}

			files := strings.Split(string(out), "\n")

			mu.Lock()
			defer mu.Unlock()
			for _, file := range files {
				absFile := filepath.Join(baseDir, file)
				relFile, err := filepath.Rel(currentDir, absFile)

				if err != nil {
					errCh <- fmt.Errorf("error getting relative path: %s", err)
					return
				}

				if ignored != nil && ignored.MatchesPath(relFile) {
					continue
				}

				activePaths[relFile] = true

				parentDir := relFile
				for parentDir != "." && parentDir != "/" && parentDir != "" {
					parentDir = filepath.Dir(parentDir)
					activeDirs[parentDir] = true
				}
			}

			errCh <- nil
		}()

		// get all ignored paths/dirs in the repo
		// in some cases when entire directories are ignored, git will just list the directory and not the files within it
		numRoutines++
		go func() {
			cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
			cmd.Dir = baseDir
			out, err := cmd.Output()

			if err != nil {
				errCh <- fmt.Errorf("error getting untracked files in git repo: %s", err)
				return
			}

			paths := strings.Split(string(out), "\n")

			mu.Lock()
			defer mu.Unlock()

			for _, file := range paths {
				absFile := filepath.Join(baseDir, file)
				relFile, err := filepath.Rel(currentDir, absFile)

				if err != nil {
					errCh <- fmt.Errorf("error getting relative path: %s", err)
					return
				}

				// check if git is ignoring the entire directory, meaning it won't list the files within it
				if strings.HasSuffix(file, "/") {
					allDirs[relFile] = true
					allPaths[relFile] = true
					gitIgnoredDirs[relFile+"/"] = true
				} else {
					allPaths[relFile] = true
				}

			}

			errCh <- nil
		}()

	} else {

		// get all paths in the directory
		numRoutines++
		go func() {
			err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					if info.Name() == ".git" {
						return filepath.SkipDir
					}
					if strings.HasPrefix(info.Name(), ".plandex") {
						return filepath.SkipDir
					}

					relPath, err := filepath.Rel(currentDir, path)
					if err != nil {
						return err
					}

					allDirs[relPath] = true

					if ignored != nil && ignored.MatchesPath(relPath) {
						return filepath.SkipDir
					}
				} else {
					relPath, err := filepath.Rel(currentDir, path)
					if err != nil {
						return err
					}

					allPaths[relPath] = true

					if ignored != nil && ignored.MatchesPath(relPath) {
						return nil
					}

					if !isGitRepo {
						mu.Lock()
						defer mu.Unlock()
						activePaths[relPath] = true

						parentDir := relPath
						for parentDir != "." && parentDir != "/" && parentDir != "" {
							parentDir = filepath.Dir(parentDir)
							activeDirs[parentDir] = true
						}
					}
				}

				return nil
			})

			if err != nil {
				errCh <- fmt.Errorf("error walking directory: %s", err)
				return
			}

			errCh <- nil
		}()
	}

	for i := 0; i < numRoutines; i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	for dir := range allDirs {
		allPaths[dir] = true
	}

	for dir := range activeDirs {
		activePaths[dir] = true
	}

	// remove deleted files from active paths
	for path := range deletedFiles {
		delete(activePaths, path)
	}

	ignoredPaths := map[string]string{}
	for path := range allPaths {
		if _, ok := activePaths[path]; !ok {
			if ignored != nil && ignored.MatchesPath(path) {
				ignoredPaths[path] = "plandex"
			} else {
				ignoredPaths[path] = "git"
			}
		}
	}

	return &types.ProjectPaths{
		ActivePaths:    activePaths,
		AllPaths:       allPaths,
		ActiveDirs:     activeDirs,
		AllDirs:        allDirs,
		PlandexIgnored: ignored,
		IgnoredPaths:   ignoredPaths,
		GitIgnoredDirs: gitIgnoredDirs,
	}, nil
}

func GetPlandexIgnore(dir string) (*ignore.GitIgnore, error) {
	ignorePath := filepath.Join(dir, ".plandexignore")

	if _, err := os.Stat(ignorePath); err == nil {
		ignored, err := ignore.CompileIgnoreFile(ignorePath)

		if err != nil {
			return nil, fmt.Errorf("error reading .plandexignore file: %s", err)
		}

		return ignored, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("error checking for .plandexignore file: %s", err)
	}

	return nil, nil
}

func GetBaseDirForContexts(contexts []*shared.Context) string {
	var paths []string

	for _, context := range contexts {
		if context.FilePath != "" {
			paths = append(paths, context.FilePath)
		}
	}

	return GetBaseDirForFilePaths(paths)
}

func GetBaseDirForFilePaths(paths []string) string {
	baseDir := ProjectRoot
	dirsUp := 0

	for _, path := range paths {
		currentDir := ProjectRoot

		pathSplit := strings.Split(path, string(os.PathSeparator))

		n := 0
		for _, p := range pathSplit {
			if p == ".." {
				n++
				currentDir = filepath.Dir(currentDir)
			} else {
				break
			}
		}

		if n > dirsUp {
			dirsUp = n
			baseDir = currentDir
		}
	}

	return baseDir
}

// isSubpathOf checks if 'child' is within 'parent' (same path or deeper).
// Both 'parent' and 'child' can be absolute or relative to 'baseDir';
// we’ll convert them to absolute paths based on 'baseDir' and then compare.
func IsSubpathOf(parent, child, baseDir string) (bool, error) {
	// Convert 'parent' -> absolute
	absParent := parent
	if !filepath.IsAbs(parent) {
		absParent = filepath.Join(baseDir, parent)
	}
	absParent = filepath.Clean(absParent)

	// Convert 'child' -> absolute
	absChild := child
	if !filepath.IsAbs(child) {
		absChild = filepath.Join(baseDir, child)
	}
	absChild = filepath.Clean(absChild)

	// filepath.Rel(absParent, absChild) will be something like:
	//   - ".",  "foo",  "foo/bar", or ".." references
	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		// If there's some I/O error or invalid path, just fail safe
		return false, fmt.Errorf("error getting relative path: %s", err)
	}

	// If rel starts with "..", then absChild is outside of absParent
	// or at a higher level (e.g. absParent/../sibling).
	if strings.HasPrefix(rel, "..") {
		return false, nil
	}

	// If we want "absChild == absParent" to count as inside,
	// then !HasPrefix(rel, "..") is enough.
	// This means child == parent or child is deeper within parent.
	return true, nil
}

func IsIgnored(paths *types.ProjectPaths, path, baseDir string) (bool, string, error) {
	if !paths.AllPaths[path] {
		// if the path isn't in AllPaths, it either:
		// 1. doesn't exist (in which case we shouldn't be calling this function)
		// 2. is a subpath of a git ignored dir

		for dir := range paths.GitIgnoredDirs {
			subpath, err := IsSubpathOf(dir, path, baseDir)
			if err != nil {
				return false, "", fmt.Errorf("error checking if %s is a subpath of %s: %s", path, dir, err)
			}
			if subpath {
				return true, "git", nil
			}
		}
		return false, "", fmt.Errorf("path %s is not in the project", path)
	}

	if paths.ActivePaths[path] {
		return false, "", nil
	}

	if paths.PlandexIgnored != nil && paths.PlandexIgnored.MatchesPath(path) {
		return true, "plandex", nil
	}

	return true, "git", nil
}
