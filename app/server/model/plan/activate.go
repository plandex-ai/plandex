package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/host"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func activatePlan(client *openai.Client, plan *db.Plan, branch string, auth *types.ServerAuth, prompt string, buildOnly bool) (*types.ActivePlan, error) {
	active := GetActivePlan(plan.Id, branch)
	if active != nil {
		log.Printf("Tell: Active plan found for plan ID %s on branch %s\n", plan.Id, branch) // Log if an active plan is found
		return nil, fmt.Errorf("plan %s branch %s already has an active stream on this host", plan.Id, branch)
	}

	modelStream, err := db.GetActiveModelStream(plan.Id, branch)
	if err != nil {
		log.Printf("Error getting active model stream: %v\n", err)
		return nil, fmt.Errorf("error getting active model stream: %v", err)
	}

	if modelStream != nil {
		log.Printf("Tell: Active model stream found for plan ID %s on branch %s on host %s\n", plan.Id, branch, modelStream.InternalIp) // Log if an active model stream is found
		return nil, fmt.Errorf("plan %s branch %s already has an active stream on host %s", plan.Id, branch, modelStream.InternalIp)
	}

	active = CreateActivePlan(plan.Id, branch, prompt, buildOnly)

	modelStream = &db.ModelStream{
		OrgId:      auth.OrgId,
		PlanId:     plan.Id,
		InternalIp: host.Ip,
		Branch:     branch,
	}
	err = db.StoreModelStream(modelStream, active.Ctx, active.CancelFn)
	if err != nil {
		log.Printf("Tell: Error storing model stream for plan ID %s on branch %s: %v\n", plan.Id, branch, err) // Log error storing model stream
		log.Printf("Error storing model stream: %v\n", err)
		log.Printf("Tell: Error storing model stream: %v\n", err) // Log error storing model stream

		active.StreamDoneCh <- &shared.ApiError{Msg: fmt.Sprintf("Error storing model stream: %v", err)}

		return nil, fmt.Errorf("error storing model stream: %v", err)
	}

	active.ModelStreamId = modelStream.Id

	log.Printf("Tell: Model stream stored with ID %s for plan ID %s on branch %s\n", modelStream.Id, plan.Id, branch) // Log successful storage of model stream
	log.Println("Model stream id:", modelStream.Id)

	return active, nil
}
