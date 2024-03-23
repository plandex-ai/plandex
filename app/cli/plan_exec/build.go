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

	anyOutdated, didUpdate := params.CheckOutdatedContext(contexts)

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		log.Println("Build canceled")
		return false, nil
	}

	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		return false, fmt.Errorf("error getting project paths: %v", err)
	}

	apiErr = api.Client.BuildPlan(params.CurrentPlanId, params.CurrentBranch, shared.BuildPlanRequest{
		ConnectStream: !buildBg,
		ProjectPaths:  paths.ActivePaths,
		ApiKey:        os.Getenv("OPENAI_API_KEY"),
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
