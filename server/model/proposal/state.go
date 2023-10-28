package proposal

import (
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

var (
	proposals         types.SafeMap[*types.Proposal]             = *types.NewSafeMap[*types.Proposal]()
	plans             types.SafeMap[*types.Plan]                 = *types.NewSafeMap[*types.Plan]()
	convoSummaryProcs types.SafeMap[*types.ConvoSummaryProc]     = *types.NewSafeMap[*types.ConvoSummaryProc]()
	convoSummaries    types.SafeMap[*shared.ConversationSummary] = *types.NewSafeMap[*shared.ConversationSummary]()
)

func GetConvoSummary(rootId string) *shared.ConversationSummary {
	return convoSummaries.Get(rootId)
}
