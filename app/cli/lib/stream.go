package lib

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

	streamtui.Send(*params.Msg)
}
