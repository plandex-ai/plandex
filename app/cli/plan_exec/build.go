package plan_exec

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/fs"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/plandex/plandex/shared"
)

func Build(params ExecParams, buildBg bool) (bool, error) {
	term.StartSpinner("")

	contexts, apiErr := api.Client.ListContext(params.CurrentPlanId, params.CurrentBranch)

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting context: %v", apiErr)
	}

	anyOutdated, didUpdate, err := params.CheckOutdatedContext(contexts)

	if err != nil {
		term.OutputErrorAndExit("error checking outdated context: %v", err)
	}

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		log.Println("Build canceled")
		return false, nil
	}

	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		return false, fmt.Errorf("error getting project paths: %v", err)
	}

	var legacyApiKey, openAIBase, openAIOrgId string

	if params.ApiKeys["OPENAI_API_KEY"] != "" {
		legacyApiKey = params.ApiKeys["OPENAI_API_KEY"]
		openAIBase = os.Getenv("OPENAI_API_BASE")
		if openAIBase == "" {
			openAIBase = os.Getenv("OPENAI_ENDPOINT")
		}
		openAIOrgId = os.Getenv("OPENAI_ORG_ID")
	}

	// log.Println("Building plan...")
	// log.Println("API keys:", params.ApiKeys)
	// log.Println("Legacy API key:", legacyApiKey)

	apiErr = api.Client.BuildPlan(params.CurrentPlanId, params.CurrentBranch, shared.BuildPlanRequest{
		ConnectStream: !buildBg,
		ProjectPaths:  paths.ActivePaths,
		ApiKey:        legacyApiKey, // deprecated
		Endpoint:      openAIBase,   // deprecated
		ApiKeys:       params.ApiKeys,
		OpenAIBase:    openAIBase,
		OpenAIOrgId:   openAIOrgId,
	}, stream.OnStreamPlan)

	term.StopSpinner()

	if apiErr != nil {
		if apiErr.Msg == shared.NoBuildsErr {
			fmt.Println("🤷‍♂️ This plan has no pending changes to build")
			return false, nil
		}

		return false, fmt.Errorf("error building plan: %v", apiErr.Msg)
	}

	if !buildBg {
		ch := make(chan error)

		go func() {
			err := streamtui.StartStreamUI("", true)

			if err != nil {
				ch <- fmt.Errorf("error starting stream UI: %v", err)
				return
			}

			ch <- nil
		}()

		// Wait for the stream to finish
		err := <-ch

		if err != nil {
			return false, err
		}
	}

	return true, nil
}
