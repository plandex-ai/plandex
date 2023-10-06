package types

type Plan struct {
	ProposalId    string
	NumFiles      int
	Files         map[string]string
	HasExec       bool
	Exec          string
	ExecErr       error
	ExecFinished  bool
	FilesFinished map[string]bool
	FileErrs      map[string]error
	ProposalStage
}

func (s *Plan) DidFinish() bool {
	if s.HasExec && !s.ExecFinished {
		return false
	}

	return len(s.FilesFinished) == int(s.NumFiles)
}
