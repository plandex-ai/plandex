package lib

import (
	"encoding/json"
	"log"
	"plandex/api"
	streamtui "plandex/stream_tui"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

var OnStreamPlan api.OnStreamPlan = func(params api.OnStreamPlanParams) {
	switch params.State.Current() {
	case shared.STATE_REPLYING:
		streamtui.Send(types.StreamTUIUpdate{
			ReplyChunk: params.Content,
		})
	case shared.STATE_DESCRIBING:
		streamtui.Send(types.StreamTUIUpdate{
			Processing: true,
		})

	case shared.STATE_BUILDING:
		var tokenCount shared.PlanTokenCount
		err := json.Unmarshal([]byte(params.Content), &tokenCount)
		if err != nil {
			log.Println("error parsing plan token count update:", err)
			return
		}
		streamtui.Send(types.StreamTUIUpdate{
			PlanTokenCount: &tokenCount,
		})
	}

}
