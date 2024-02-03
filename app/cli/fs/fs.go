package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

var Cwd string
var PlandexDir string
var ProjectRoot string
var HomePlandexDir string
var CacheDir string

var HomeAuthPath string
var HomeAccountsPath string

func init() {
	var err error
	Cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic("Couldn't find home dir:" + err.Error())
	}
	HomePlandexDir = filepath.Join(home, ".plandex-home")
	CacheDir = filepath.Join(HomePlandexDir, "cache")
	HomeAuthPath = filepath.Join(HomePlandexDir, "auth.json")
	HomeAccountsPath = filepath.Join(HomePlandexDir, "accounts.json")

	err = os.MkdirAll(filepath.Join(CacheDir, "tiktoken"), os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = os.Setenv("TIKTOKEN_CACHE_DIR", CacheDir)
	if err != nil {
		panic(err)
	}

	PlandexDir, ProjectRoot = findPlandex()
}

func FindOrCreatePlandex() (string, bool, error) {
	PlandexDir, ProjectRoot = findPlandex()

	if PlandexDir != "" && ProjectRoot != "" {
		return PlandexDir, false, nil
	}

	// Determine the directory path
	dir := filepath.Join(Cwd, ".plandex")

	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		return "", false, err
	}
	PlandexDir = dir
	ProjectRoot = Cwd

	return dir, true, nil
}

func ProjectRootIsGitRepo() bool {
	if ProjectRoot == "" {
		return false
	}

	return IsGitRepo(ProjectRoot)
}

func IsGitRepo(dir string) bool {
	isGitRepo := false

	if isCommandAvailable("git") {
		// check whether we're in a git repo
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")

		cmd.Dir = dir

		err := cmd.Run()

		if err == nil {
			isGitRepo = true
		}
	}

	return isGitRepo
}

func GetProjectPaths() (map[string]bool, *ignore.GitIgnore, error) {
	if ProjectRoot == "" {
		return nil, nil, fmt.Errorf("no project root found")
	}

	return GetPaths(ProjectRoot)
}

func GetPaths(dir string) (map[string]bool, *ignore.GitIgnore, error) {
	paths := map[string]bool{}

	if ProjectRootIsGitRepo() {
		// combine `git ls-files` and `git ls-files --others --exclude-standard`
		// to get all files in the repo

		// get all tracked files in the repo
		cmd := exec.Command("git", "ls-files")
		cmd.Dir = dir
		out, err := cmd.Output()

		if err != nil {
			return nil, nil, fmt.Errorf("error getting files in git repo: %s", err)
		}

		files := strings.Split(string(out), "\n")

		for _, file := range files {
			paths[file] = true
		}

		// get all untracked files non-ignored files in the repo
		cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
		cmd.Dir = dir
		out, err = cmd.Output()

		if err != nil {
			return nil, nil, fmt.Errorf("error getting untracked files in git repo: %s", err)
		}

		files = strings.Split(string(out), "\n")

		for _, file := range files {
			paths[file] = true
		}
	} else {
		// get all files in the directory
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				paths[relPath] = true
			}

			return nil
		})

		if err != nil {
			return nil, nil, fmt.Errorf("error getting files in directory: %s", err)
		}
	}

	ignored, err := GetPlandexIgnore()

	if err != nil {
		return nil, nil, err
	}

	if ignored != nil {
		for path := range paths {
			if ignored.MatchesPath(path) {
				delete(paths, path)
			}
		}
	}

	return paths, ignored, nil
}

func GetPlandexIgnore() (*ignore.GitIgnore, error) {
	ignorePath := filepath.Join(ProjectRoot, ".plandexignore")

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

func findPlandex() (string, string) {
	searchPath := Cwd
	for searchPath != "/" {
		dir := filepath.Join(searchPath, ".plandex")
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			return dir, searchPath
		}
		searchPath = filepath.Dir(searchPath)
	}

	return "", ""
}

func isCommandAvailable(name string) bool {
	cmd := exec.Command(name, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
