package plan

import (
	"fmt"
	"plandex-server/db"

	"github.com/sashabaranov/go-openai"
)

func Stop(planId, currentUserId, currentOrgId string) error {

	active := Active.Get(planId)

	if active == nil {
		return fmt.Errorf("no active plan with id %s", planId)
	}

	content := active.Content

	active.CancelFn()

	userMsg := db.ConvoMessage{
		OrgId:   currentOrgId,
		PlanId:  planId,
		UserId:  currentUserId,
		Role:    openai.ChatMessageRoleAssistant,
		Tokens:  active.NumTokens,
		Num:     active.PromptMessageNum + 1,
		Stopped: true,
		Message: content,
	}

	err := db.StoreConvoMessage(&userMsg)

	if err != nil {
		return fmt.Errorf("error storing convo message: %v", err)
	}

	return nil
}
