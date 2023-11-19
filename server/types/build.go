package types

import "github.com/plandex/plandex/shared"

type BuildParams struct {
	ProposalId  string                   `json:"proposalId"`
	BuildPaths  []string                 `json:"buildPaths"`
	CurrentPlan *shared.CurrentPlanFiles `json:"currentPlan"`
}

type Build struct {
	BuildId    string
	ProposalId string
	NumFiles   int
	Buffers    map[string]string
	Results    map[string]*shared.PlanResult
	Errs       map[string]error
	ProposalStage
}

func (s *Build) DidFinish() bool {
	return len(s.Results) == int(s.NumFiles)
}
