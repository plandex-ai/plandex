package plan_exec

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/fs"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func Build(params ExecParams, buildBg bool) (bool, error) {
	params.CheckOutdatedContext()

	term.StartSpinner("🏗️  Starting build...")

	contexts, apiErr := api.Client.ListContext(params.CurrentPlanId, params.CurrentBranch)

	if apiErr != nil {
		color.New(color.FgRed).Fprintln(os.Stderr, "Error getting context:", apiErr)
		os.Exit(1)
	}

	projectPaths, _, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting project paths:", err)
		return false, fmt.Errorf("error getting project paths: %v", err)
	}

	apiErr = api.Client.BuildPlan(params.CurrentPlanId, params.CurrentBranch, shared.BuildPlanRequest{
		ConnectStream: !buildBg,
		ProjectPaths:  projectPaths,
		ApiKey:        os.Getenv("OPENAI_API_KEY"),
	}, stream.OnStreamPlan)

	term.StopSpinner()

	if apiErr != nil {
		if apiErr.Msg == shared.NoBuildsErr {
			fmt.Println("🤷‍♂️ This plan has no pending changes to build")
			return false, nil
		}

		fmt.Fprintln(os.Stderr, "Error building plan:", apiErr.Msg)
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
