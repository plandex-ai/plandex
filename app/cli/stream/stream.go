package stream

import (
	"log"
	"plandex-cli/api"
	"plandex-cli/lib"
	streamtui "plandex-cli/stream_tui"
	"plandex-cli/term"
	"plandex-cli/types"
	"strings"

	shared "plandex-shared"
)

var OnStreamPlan types.OnStreamPlan

func init() {
	OnStreamPlan = func(params types.OnStreamPlanParams) {
		if params.Err != nil {
			if strings.Contains(params.Err.Error(), "missing heartbeats") || strings.Contains(strings.ToLower(params.Err.Error()), "eof") {
				log.Println("Error in stream:", params.Err)
				streamtui.Send(shared.StreamMessage{
					Type: shared.StreamMessageError,
					Error: &shared.ApiError{
						Msg: "Stream error: " + params.Err.Error(),
					},
				})

				// try to reconnect
				term.StartSpinner("Reconnecting...")
				apiErr := api.Client.ConnectPlan(lib.CurrentPlanId, lib.CurrentBranch, OnStreamPlan)
				term.StopSpinner()

				if apiErr != nil {
					log.Println("Error reconnecting to stream:", apiErr)
				}
			}

			return
		}

		if params.Msg.Type == shared.StreamMessageStart {
			log.Println("Stream started")
			return
		}

		// log.Println("Stream message:")
		// log.Println(spew.Sdump(*params.Msg))

		streamtui.Send(*params.Msg)
	}
}
