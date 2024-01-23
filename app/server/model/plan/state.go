package plan

import (
	"log"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

var (
	Active types.SafeMap[*types.ActivePlan] = *types.NewSafeMap[*types.ActivePlan]()
)

func CreateActivePlan(planId, prompt string) *types.ActivePlan {
	activePlan := types.NewActivePlan(planId, prompt)
	Active.Set(planId, activePlan)

	go func() {
		for {
			select {
			case <-activePlan.Ctx.Done():
				return
			case err := <-activePlan.StreamDoneCh:
				if err == nil {
					log.Printf("Plan %s stream completed successfully", planId)

					err := db.SetPlanStatus(planId, shared.PlanStatusFinished, "")
					if err != nil {
						log.Printf("Error setting plan %s status to ready: %v\n", planId, err)
					}

				} else {
					log.Printf("Error streaming plan %s: %v\n", planId, err)

					err := db.SetPlanStatus(planId, shared.PlanStatusError, err.Msg)
					if err != nil {
						log.Printf("Error setting plan %s status to error: %v\n", planId, err)
					}
				}

				activePlan.CancelFn()

				Active.Delete(planId)
				return
			}
		}
	}()

	return activePlan
}

func SubscribePlan(planId string) (string, chan string) {
	var id string
	var ch chan string
	Active.Update(planId, func(activePlan *types.ActivePlan) {
		id, ch = activePlan.Subscribe()
	})
	return id, ch
}

func UnsubscribePlan(planId, subscriptionId string) {
	active := Active.Get(planId)

	if active == nil {
		return
	}

	Active.Update(planId, func(activePlan *types.ActivePlan) {
		activePlan.Unsubscribe(subscriptionId)
	})
}
