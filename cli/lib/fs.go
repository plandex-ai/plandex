package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plandex/types"
	"sync"

	"github.com/plandex/plandex/shared"
)

var Cwd string
var PlandexDir string
var HomePlandexDir string
var CacheDir string

var CurrentPlanName string
var CurrentPlanRootDir string
var PlanSubdir string
var ConversationSubdir string
var ContextSubdir string

func init() {
	var err error
	Cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		HomePlandexDir = filepath.Join(home, ".plandex")
		CacheDir = filepath.Join(HomePlandexDir, "cache")
		err = os.MkdirAll(filepath.Join(CacheDir, "tiktoken"), os.ModePerm)
		if err != nil {
			panic(err)
		}
		err = os.Setenv("TIKTOKEN_CACHE_DIR", CacheDir)
		if err != nil {
			panic(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		PlandexDir = FindPlandex()
		err = LoadCurrentPlan()
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}

func LoadCurrentPlan() error {
	// Construct the path to the current_plan.json file
	path := filepath.Join(PlandexDir, "current_plan.json")

	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	// Read the contents of the file
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading current_plan.json: %v", err)
	}

	// Unmarshal the JSON data into the shared.PlanSettings type
	var planSettings shared.PlanSettings
	err = json.Unmarshal(fileBytes, &planSettings)
	if err != nil {
		return fmt.Errorf("error unmarshalling current_plan.json: %v", err)
	}

	CurrentPlanName = planSettings.Name
	CurrentPlanRootDir = filepath.Join(PlandexDir, CurrentPlanName)
	PlanSubdir = filepath.Join(CurrentPlanRootDir, "plan")
	ConversationSubdir = filepath.Join(CurrentPlanRootDir, "conversation")
	ContextSubdir = filepath.Join(CurrentPlanRootDir, "context")

	return nil
}

func FindOrCreatePlandex() (string, bool, error) {
	var err error

	// Determine the directory path
	dir := filepath.Join(Cwd, ".plandex")

	// Check if the directory already exists
	_, err = os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// If not found, create in the current directory
			err = os.Mkdir(dir, os.ModePerm)
			if err != nil {
				return "", false, err
			}
			PlandexDir = dir
			return dir, true, nil
		}
		return "", false, err
	}

	return dir, false, nil
}

func FindPlandex() string {
	searchPath := Cwd
	for searchPath != "/" {
		dir := filepath.Join(searchPath, ".plandex")
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			return dir
		}
		searchPath = filepath.Dir(searchPath)
	}

	return ""
}

func CwdIsPlan() bool {
	// check if parent directory of cwd is '.plandex'
	parentDir := filepath.Dir(Cwd)
	return parentDir == PlandexDir
}

func flattenPaths(fileOrDirPaths []string, params *types.LoadContextParams, depth int16) []string {
	var wg sync.WaitGroup
	resPathsChan := make(chan string, len(fileOrDirPaths))

	for _, path := range fileOrDirPaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Fatalf("Failed to read the file %s: %v", path, err)
		}

		if fileInfo.IsDir() {
			if !params.Recursive {
				log.Fatalf("The path %s is a directory. Please use the --recursive / -r flag to load the directory recursively.", path)
			}
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						log.Fatalf("Failed to read the file %s: %v", path, err)
					}

					if info.IsDir() {
						if depth < params.MaxDepth {
							for _, subPath := range flattenPaths([]string{path}, params, depth+1) {
								resPathsChan <- subPath
							}
						} else if params.NamesOnly {
							resPathsChan <- path
						}
					} else {
						resPathsChan <- path
					}

					return nil
				})
				if err != nil {
					log.Fatalf("Failed to process directory %s: %v", p, err)
				}
			}(path)
		} else {
			resPathsChan <- path
		}
	}

	go func() {
		wg.Wait()
		close(resPathsChan)
	}()

	var resPaths []string
	for p := range resPathsChan {
		resPaths = append(resPaths, p)
	}

	return resPaths
}
