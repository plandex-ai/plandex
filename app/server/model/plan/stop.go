package plan

import (
	"fmt"
	"plandex-server/db"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Stop(planId, branch, currentUserId, currentOrgId string) error {
	active := GetActivePlan(planId, branch)

	if active == nil {
		return fmt.Errorf("no active plan with id %s", planId)
	}

	active.Stream(shared.StreamMessage{
		Type: shared.StreamMessageAborted,
	})

	// give some time for stream message to be processed before canceling
	time.Sleep(100 * time.Millisecond)

	active.CancelFn()

	if !active.BuildOnly {
		num := active.MessageNum + 1

		userMsg := db.ConvoMessage{
			OrgId:   currentOrgId,
			PlanId:  planId,
			UserId:  currentUserId,
			Role:    openai.ChatMessageRoleAssistant,
			Tokens:  active.NumTokens,
			Num:     num,
			Stopped: true,
			Message: active.CurrentReplyContent,
		}

		_, err := db.StoreConvoMessage(&userMsg, currentUserId, branch, true)

		if err != nil {
			return fmt.Errorf("error storing convo message: %v", err)
		}
	}

	return nil
}
