package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/term"
	"plandex/types"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

var CurrentProjectId string
var CurrentPlanId string
var CurrentBranch string
var HomeCurrentProjectDir string
var HomeCurrentPlanPath string

func MustResolveOrCreateProject() {
	resolveProject(true, true)
}

func MustResolveProject() {
	resolveProject(true, false)
}

func MaybeResolveProject() {
	resolveProject(false, false)
}

func resolveProject(mustResolve, shouldCreate bool) {
	if fs.PlandexDir == "" {
		var err error
		if shouldCreate {
			_, _, err = fs.FindOrCreatePlandex()
		} else {
			fs.FindPlandexDir()
		}

		if err != nil && mustResolve {
			term.OutputErrorAndExit("error finding or creating plandex: %v", err)
		}
	}

	if (fs.PlandexDir == "" || fs.ProjectRoot == "") && mustResolve {
		fmt.Printf(
			"🤷‍♂️ No plans in current directory\nTry %s to create a plan or %s to see plans in nearby directories\n",
			color.New(color.Bold, term.ColorHiCyan).Sprint("plandex new"),
			color.New(color.Bold, term.ColorHiCyan).Sprint("plandex plans"))
		os.Exit(0)
	}

	if fs.PlandexDir == "" {
		return
	}

	MigrateLegacyProjectFile(auth.Current.UserId)

	// check if projects-v2.json exists in PlandexDir
	path := filepath.Join(fs.PlandexDir, "projects-v2.json")
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		log.Println("projects-v2.json does not exist")
		log.Println("Initializing project")
		mustInitProject(nil)
	} else if err != nil {
		term.OutputErrorAndExit("error checking if projects-v2.json exists: %v", err)
	}

	var settings *types.CurrentProjectSettings
	var loadProjectSettings func()
	loadProjectSettings = func() {
		// read projects-v2.json
		bytes, err := os.ReadFile(path)

		if err != nil {
			term.OutputErrorAndExit("error reading projects-v2.json: %v", err)
		}

		var settingsByAccount types.CurrentProjectSettingsByAccount
		err = json.Unmarshal(bytes, &settingsByAccount)

		if err != nil {
			term.OutputErrorAndExit("error unmarshalling projects-v2.json: %v", err)
		}

		settings = settingsByAccount[auth.Current.UserId]
		if settings == nil {
			mustInitProject(&settingsByAccount)
			loadProjectSettings()
		}
	}

	loadProjectSettings()

	CurrentProjectId = settings.Id
	MigrateLegacyCurrentPlanFile(auth.Current.UserId)

	HomeCurrentProjectDir = filepath.Join(fs.HomePlandexDir, CurrentProjectId)
	HomeCurrentPlanPath = filepath.Join(HomeCurrentProjectDir, "current-plans-v2.json")

	err = os.MkdirAll(HomeCurrentProjectDir, os.ModePerm)

	if err != nil {
		term.OutputErrorAndExit("error creating project dir: %v", err)
	}

	MustLoadCurrentPlan()
	MigrateLegacyPlanSettingsFile(auth.Current.UserId)
}

func MustLoadCurrentPlan() {
	if CurrentProjectId == "" {
		term.OutputErrorAndExit("No current project")
	}

	// Check if the file exists
	_, err := os.Stat(HomeCurrentPlanPath)

	if os.IsNotExist(err) {
		return
	} else if err != nil {
		term.OutputErrorAndExit("error checking if current-plans-v2.json exists: %v", err)
	}

	// Read the contents of the file
	fileBytes, err := os.ReadFile(HomeCurrentPlanPath)
	if err != nil {
		term.OutputErrorAndExit("error reading current-plans-v2.json: %v", err)
	}

	var currentPlansByAccount types.CurrentPlanSettingsByAccount
	err = json.Unmarshal(fileBytes, &currentPlansByAccount)
	if err != nil {
		term.OutputErrorAndExit("error unmarshalling current-plans-v2.json: %v", err)
	}

	currentPlan := currentPlansByAccount[auth.Current.UserId]

	if currentPlan != nil {
		CurrentPlanId = currentPlan.Id
	}

	if CurrentPlanId != "" {
		err = loadCurrentBranch()

		if err != nil {
			term.OutputErrorAndExit("error loading current branch: %v", err)
		}

		if CurrentBranch == "" {
			err = WriteCurrentBranch("main")

			if err != nil {
				term.OutputErrorAndExit("error setting current branch: %v", err)
			}
		}
	}
}

func loadCurrentBranch() error {
	// Load plan-specific settings
	if CurrentPlanId == "" {
		return fmt.Errorf("no current plan")
	}

	path := filepath.Join(HomeCurrentProjectDir, CurrentPlanId, "settings-v2.json")

	// Check if the file exists
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking if settings-v2.json exists: %v", err)
	}

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		term.OutputErrorAndExit("error reading settings-v2.json: %v", err)
	}

	var settingsByAccount types.PlanSettingsByAccount
	err = json.Unmarshal(fileBytes, &settingsByAccount)
	if err != nil {
		term.OutputErrorAndExit("error unmarshalling settings-v2.json: %v", err)
	}
	settings := settingsByAccount[auth.Current.UserId]

	if settings == nil {
		return nil
	}

	CurrentBranch = settings.Branch

	return nil
}

func mustInitProject(existingSettings *types.CurrentProjectSettingsByAccount) {
	res, apiErr := api.Client.CreateProject(shared.CreateProjectRequest{Name: filepath.Base(fs.ProjectRoot)})

	if apiErr != nil {
		term.OutputErrorAndExit("error creating project: %v", apiErr.Msg)
	}

	log.Println("Project created:", res.Id)

	CurrentProjectId = res.Id

	var settingsByAccount types.CurrentProjectSettingsByAccount
	if existingSettings != nil {
		settingsByAccount = *existingSettings
	} else {
		settingsByAccount = types.CurrentProjectSettingsByAccount{}
	}

	settingsByAccount[auth.Current.UserId] = &types.CurrentProjectSettings{
		Id: CurrentProjectId,
	}

	// write projects-v2.json
	path := filepath.Join(fs.PlandexDir, "projects-v2.json")
	bytes, err := json.Marshal(settingsByAccount)

	if err != nil {
		term.OutputErrorAndExit("error marshalling project settings: %v", err)
	}

	err = os.WriteFile(path, bytes, os.ModePerm)

	if err != nil {
		term.OutputErrorAndExit("error writing projects-v2.json: %v", err)
	}

	log.Println("Wrote projects-v2.json")

	// write current-plans-v2.json to PlandexHomeDir/[projectId]/current-plans-v2.json
	dir := filepath.Join(fs.HomePlandexDir, CurrentProjectId)
	err = os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		term.OutputErrorAndExit("error creating project dir: %v", err)
	}

	path = filepath.Join(dir, "current-plans-v2.json")
	bytes, err = json.Marshal(types.CurrentPlanSettingsByAccount{
		auth.Current.UserId: &types.CurrentPlanSettings{
			Id: "",
		},
	})

	if err != nil {
		term.OutputErrorAndExit("error marshalling plan settings: %v", err)
	}

	err = os.WriteFile(path, bytes, os.ModePerm)

	if err != nil {
		term.OutputErrorAndExit("error writing current-plans-v2.json: %v", err)
	}

	log.Println("Wrote current-plans-v2.json")
}
