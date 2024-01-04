package fs

import (
	"os"
	"path/filepath"
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
