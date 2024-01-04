package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/fs"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

var CurrentProjectId string
var CurrentPlanId string
var HomeCurrentProjectDir string
var HomeCurrentPlanPath string

func MustResolveProject() {
	if fs.PlandexDir == "" {
		_, _, err := fs.FindOrCreatePlandex()
		if err != nil {
			panic(fmt.Errorf("error finding or creating plandex: %v", err))
		}
	}

	if fs.PlandexDir == "" || fs.ProjectRoot == "" {
		panic(fmt.Errorf("could not find or create plandex directory"))
	}

	// check if project.json exists in PlandexDir
	path := filepath.Join(fs.PlandexDir, "project.json")
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

	HomeCurrentProjectDir = filepath.Join(fs.HomePlandexDir, CurrentProjectId)
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

func mustInitProject() {
	res, err := api.Client.CreateProject(shared.CreateProjectRequest{Name: filepath.Base(fs.ProjectRoot)})

	if err != nil {
		panic(fmt.Errorf("error creating project: %v", err))
	}

	CurrentProjectId = res.Id

	// write project.json
	path := filepath.Join(fs.PlandexDir, "project.json")
	bytes, err := json.Marshal(types.ProjectSettings{
		Id: CurrentProjectId,
	})

	if err != nil {
		panic(fmt.Errorf("error marshalling project settings: %v", err))
	}

	err = os.WriteFile(path, bytes, os.ModePerm)

	if err != nil {
		panic(fmt.Errorf("error writing project.json: %v", err))
	}

	// write current_plan.json to PlandexHomeDir/[projectId]/current_plan.json
	dir := filepath.Join(fs.HomePlandexDir, CurrentProjectId)
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
