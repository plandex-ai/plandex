package plan

import (
	"log"
	"plandex-server/db"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
)

var (
	activePlans types.SafeMap[*types.ActivePlan] = *types.NewSafeMap[*types.ActivePlan]()
)

func GetActivePlan(planId, branch string) *types.ActivePlan {
	return activePlans.Get(strings.Join([]string{planId, branch}, "|"))
}

func CreateActivePlan(planId, branch, prompt string, buildOnly bool) *types.ActivePlan {
	activePlan := types.NewActivePlan(planId, branch, prompt, buildOnly)
	key := strings.Join([]string{planId, branch}, "|")

	activePlans.Set(key, activePlan)

	go func() {
		for {
			select {
			case <-activePlan.Ctx.Done():
				log.Printf("case <-activePlan.Ctx.Done(): %s\n", planId)

				err := db.SetPlanStatus(planId, branch, shared.PlanStatusStopped, "")
				if err != nil {
					log.Printf("Error setting plan %s status to stopped: %v\n", planId, err)
				}

				DeleteActivePlan(planId, branch)

				return
			case apiErr := <-activePlan.StreamDoneCh:
				log.Printf("case apiErr := <-activePlan.StreamDoneCh: %s\n", planId)
				log.Printf("apiErr: %v\n", apiErr)

				if apiErr == nil {
					log.Printf("Plan %s stream completed successfully", planId)

					err := db.SetPlanStatus(planId, branch, shared.PlanStatusFinished, "")
					if err != nil {
						log.Printf("Error setting plan %s status to ready: %v\n", planId, err)
					}

				} else {
					log.Printf("Error streaming plan %s: %v\n", planId, apiErr)

					err := db.SetPlanStatus(planId, branch, shared.PlanStatusError, apiErr.Msg)
					if err != nil {
						log.Printf("Error setting plan %s status to error: %v\n", planId, err)
					}

					log.Println("Sending error message to client")
					activePlan.Stream(shared.StreamMessage{
						Type:  shared.StreamMessageError,
						Error: apiErr,
					})

					log.Println("Stopping any active summary stream")
					activePlan.SummaryCancelFn()

					time.Sleep(50 * time.Millisecond)
				}

				activePlan.CancelFn()
				DeleteActivePlan(planId, branch)
				return
			}
		}
	}()

	return activePlan
}

func DeleteActivePlan(planId, branch string) {
	activePlans.Delete(strings.Join([]string{planId, branch}, "|"))
}

func UpdateActivePlan(planId, branch string, fn func(*types.ActivePlan)) {
	activePlans.Update(strings.Join([]string{planId, branch}, "|"), fn)
}

func SubscribePlan(planId, branch string) (string, chan string) {
	log.Printf("Subscribing to plan %s\n", planId)
	var id string
	var ch chan string
	UpdateActivePlan(planId, branch, func(activePlan *types.ActivePlan) {
		id, ch = activePlan.Subscribe()
	})
	return id, ch
}

func UnsubscribePlan(planId, branch, subscriptionId string) {
	log.Printf("UnsubscribePlan %s - %s - %s\n", planId, branch, subscriptionId)

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("No active plan found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	UpdateActivePlan(planId, branch, func(activePlan *types.ActivePlan) {
		activePlan.Unsubscribe(subscriptionId)
		log.Printf("Unsubscribed from plan %s - %s - %s\n", planId, branch, subscriptionId)
	})
}

func NumActivePlans() int {
	return activePlans.Len()
}
