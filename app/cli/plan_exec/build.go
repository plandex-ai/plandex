package plan_exec

import (
	"fmt"
	"log"
	"plandex-cli/api"
	"plandex-cli/fs"
	"plandex-cli/stream"
	streamtui "plandex-cli/stream_tui"
	"plandex-cli/term"
	"plandex-cli/types"

	shared "plandex-shared"
)

func Build(params ExecParams, flags types.BuildFlags) (bool, error) {
	buildBg := flags.BuildBg

	term.StartSpinner("")

	contexts, apiErr := api.Client.ListContext(params.CurrentPlanId, params.CurrentBranch)

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting context: %v", apiErr)
	}

	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		return false, fmt.Errorf("error getting project paths: %v", err)
	}

	anyOutdated, didUpdate, err := params.CheckOutdatedContext(contexts, paths)

	if err != nil {
		term.OutputErrorAndExit("error checking outdated context: %v", err)
	}

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		log.Println("Build canceled")
		return false, nil
	}

	apiErr = api.Client.BuildPlan(params.CurrentPlanId, params.CurrentBranch, shared.BuildPlanRequest{
		ConnectStream: !buildBg,
		ProjectPaths:  paths.ActivePaths,
		AuthVars:      params.AuthVars,
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
			err := streamtui.StartStreamUI("", true, !flags.AutoApply)

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
