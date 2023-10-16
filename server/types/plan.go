package types

type Plan struct {
	ProposalId    string
	NumFiles      int
	Files         map[string]string
	FilesFinished map[string]bool
	FileErrs      map[string]error
	ProposalStage
}

func (s *Plan) DidFinish() bool {
	return len(s.FilesFinished) == int(s.NumFiles)
}
