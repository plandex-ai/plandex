package stream

import (
	"log"
	streamtui "plandex-cli/stream_tui"
	"plandex-cli/types"

	shared "plandex-shared"
)

var OnStreamPlan types.OnStreamPlan = func(params types.OnStreamPlanParams) {
	if params.Err != nil {
		log.Println("Error in stream:", params.Err)
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
