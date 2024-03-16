package plan

import (
	"fmt"
	"plandex-server/db"

	"github.com/sashabaranov/go-openai"
)

func Stop(planId, branch, currentUserId, currentOrgId string) error {
	active := GetActivePlan(planId, branch)

	if active == nil {
		return fmt.Errorf("no active plan with id %s", planId)
	}

	active.SummaryCancelFn()
	active.CancelFn()

	// rollback repo in case there are uncommitted builds
	err := db.GitClearUncommittedChanges(currentOrgId, planId)

	if err != nil {
		return fmt.Errorf("error clearing uncommitted changes: %v", err)
	}

	if !active.BuildOnly && !active.RepliesFinished {
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
