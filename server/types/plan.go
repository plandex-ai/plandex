package types

import "github.com/plandex/plandex/shared"

type Plan struct {
	ProposalId string
	NumFiles   int
	Buffers    map[string]string
	Results    map[string]*shared.PlanResult
	Errs       map[string]error
	ProposalStage
}

func (s *Plan) DidFinish() bool {
	return len(s.Results) == int(s.NumFiles)
}
