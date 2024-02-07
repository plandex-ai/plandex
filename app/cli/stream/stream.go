package stream

import (
	"log"
	streamtui "plandex/stream_tui"
	"plandex/types"
)

var OnStreamPlan types.OnStreamPlan = func(params types.OnStreamPlanParams) {
	if params.Err != nil {
		log.Println("Error in stream:", params.Err)
		return
	}

	// log.Println("Stream message:")
	// log.Println(spew.Sdump(*params.Msg))

	streamtui.Send(*params.Msg)
}
