package proposal

import (
	"plandex-server/types"
)

var (
	proposals types.SafeMap[types.Proposal] = *types.NewSafeMap[types.Proposal]()
	plans     types.SafeMap[types.Plan]     = *types.NewSafeMap[types.Plan]()
)
