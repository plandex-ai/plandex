package types

import "github.com/plandex/plandex/shared"

type Plan struct {
	ProposalId string
	NumFiles   int
	Buffers    map[string]string
	Files      map[string]*shared.PlanFile
	FileErrs   map[string]error
	ProposalStage
}

func (s *Plan) DidFinish() bool {
	return len(s.Files) == int(s.NumFiles)
}
