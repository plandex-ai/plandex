package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/types"
	"sync"

	"github.com/plandex/plandex/shared"
)

var Cwd string
var PlandexDir string
var ProjectRoot string
var HomePlandexDir string
var CacheDir string

var CurrentProjectId string
var CurrentOrgId string
var CurrentUserId string
var CurrentPlanId string

var HomeCurrentProjectDir string
var HomeCurrentPlanPath string

func init() {
	var err error
	Cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	HomePlandexDir = filepath.Join(home, ".plandex-home")
	CacheDir = filepath.Join(HomePlandexDir, "cache")
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

func MustResolveProject() {
	if PlandexDir == "" {
		_, _, err := findOrCreatePlandex()
		if err != nil {
			panic(fmt.Errorf("error finding or creating plandex: %v", err))
		}
	}

	if PlandexDir == "" || ProjectRoot == "" {
		panic(fmt.Errorf("could not find or create plandex directory"))
	}

	// check if project.json exists in PlandexDir
	path := filepath.Join(PlandexDir, "project.json")
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		mustInitProject()
	} else if err != nil {
		panic(fmt.Errorf("error checking if project.json exists: %v", err))
	}

	// read project.json
	bytes, err := os.ReadFile(path)

	if err != nil {
		panic(fmt.Errorf("error reading project.json: %v", err))
	}

	var settings types.ProjectSettings
	err = json.Unmarshal(bytes, &settings)

	if err != nil {
		panic(fmt.Errorf("error unmarshalling project.json: %v", err))
	}

	CurrentProjectId = settings.Id
	CurrentOrgId = settings.OrgId

	HomeCurrentProjectDir = filepath.Join(HomePlandexDir, CurrentProjectId)
	HomeCurrentPlanPath = filepath.Join(HomeCurrentProjectDir, "current_plan.json")

	err = os.MkdirAll(HomeCurrentProjectDir, os.ModePerm)

	if err != nil {
		panic(fmt.Errorf("error creating project dir: %v", err))
	}

	MustLoadCurrentPlan()
}

func MustLoadCurrentPlan() {
	if CurrentProjectId == "" {
		panic("No current project")
	}

	// Check if the file exists
	_, err := os.Stat(HomeCurrentPlanPath)

	if os.IsNotExist(err) {
		return
	} else if err != nil {
		panic(fmt.Errorf("error checking if current_plan.json exists: %v", err))
	}

	// Read the contents of the file
	fileBytes, err := os.ReadFile(HomeCurrentPlanPath)
	if err != nil {
		panic(fmt.Errorf("error reading current_plan.json: %v", err))
	}

	// Unmarshal the JSON data into the shared.PlanSettings type
	var planSettings types.PlanSettings
	err = json.Unmarshal(fileBytes, &planSettings)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling current_plan.json: %v", err))
	}

	CurrentPlanId = planSettings.Id
}

func ParseInputPaths(fileOrDirPaths []string, params *types.LoadContextParams) ([]string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	resPaths := []string{}

	for _, path := range fileOrDirPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				if firstErr != nil {
					return firstErr // If an error was encountered, stop walking
				}

				if info.IsDir() {
					if info.Name() == ".git" || info.Name() == ".plandex" {
						return filepath.SkipDir
					}

					if !(params.Recursive || params.NamesOnly) {
						return fmt.Errorf("cannot process directory %s: --recursive or --tree flag not set", path)
					}

					// calculate directory depth from base
					// depth := strings.Count(path[len(p):], string(filepath.Separator))
					// if params.MaxDepth != -1 && depth > params.MaxDepth {
					// 	return filepath.SkipDir
					// }

					if params.NamesOnly {
						// add directory name to results
						resPaths = append(resPaths, path)
					}
				} else {
					// add file path to results
					resPaths = append(resPaths, path)
				}

				return nil
			})

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return resPaths, nil
}

func mustInitProject() {
	res, err := api.Client.CreateProject(shared.CreateProjectRequest{Name: filepath.Base(ProjectRoot)})

	if err != nil {
		panic(fmt.Errorf("error creating project: %v", err))
	}

	CurrentProjectId = res.Id
	CurrentOrgId = res.OrgId

	// write project.json
	path := filepath.Join(PlandexDir, "project.json")
	bytes, err := json.Marshal(types.ProjectSettings{
		Id:    CurrentProjectId,
		OrgId: CurrentOrgId,
	})

	if err != nil {
		panic(fmt.Errorf("error marshalling project settings: %v", err))
	}

	err = os.WriteFile(path, bytes, os.ModePerm)

	if err != nil {
		panic(fmt.Errorf("error writing project.json: %v", err))
	}

	// write current_plan.json to PlandexHomeDir/[projectId]/current_plan.json
	dir := filepath.Join(HomePlandexDir, CurrentProjectId)
	err = os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		panic(fmt.Errorf("error creating project dir: %v", err))
	}

	path = filepath.Join(dir, "current_plan.json")
	bytes, err = json.Marshal(types.PlanSettings{
		Id: "",
	})

	if err != nil {
		panic(fmt.Errorf("error marshalling plan settings: %v", err))
	}

	err = os.WriteFile(path, bytes, os.ModePerm)

	if err != nil {
		panic(fmt.Errorf("error writing current_plan.json: %v", err))
	}
}

func findOrCreatePlandex() (string, bool, error) {
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
