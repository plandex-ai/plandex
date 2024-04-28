package plan

import (
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MaxBuildStreamErrorRetries = 3 // uses naive exponential backoff so be careful about setting this too high

type activeBuildStreamState struct {
	clients       map[string]*openai.Client
	auth          *types.ServerAuth
	currentOrgId  string
	currentUserId string
	plan          *db.Plan
	branch        string
	settings      *shared.PlanSettings
	modelContext  []*db.Context
}

type activeBuildStreamFileState struct {
	*activeBuildStreamState
	filePath           string
	convoMessageId     string
	build              *db.PlanBuild
	currentPlanState   *shared.CurrentPlanState
	activeBuild        *types.ActiveBuild
	currentState       string
	lineNumsNumRetry   int
	verifyFileNumRetry int
	fixFileNumRetry    int
	// fullChangesRetry            int
	streamedChangesWithLineNums []*shared.StreamedChangeWithLineNums
	updated                     string
	initialPlanFileResult       *db.PlanFileResult
	incorrectlyUpdatedReasoning string
}
