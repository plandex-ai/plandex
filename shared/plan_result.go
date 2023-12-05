package shared

type PlanResult struct {
	ProposalId   string         `json:"proposalId"`
	Path         string         `json:"path"`
	ContextSha   string         `json:"contextSha"`
	Content      string         `json:"content"`
	Ts           string         `json:"ts"`
	AnyFailed    bool           `json:"anyFailed"`
	Replacements []*Replacement `json:"replacements"`
	AppliedAt    string         `json:"appliedAt"`
	RejectedAt   string         `json:"rejectedAt"`
}

type PlanResultsByPath map[string][]*PlanResult

func (res *PlanResult) NumPendingReplacements() int {
	numPending := 0
	for _, rep := range res.Replacements {
		if rep.IsPending() {
			numPending++
		}
	}
	return numPending
}

func (res *PlanResult) IsPending() bool {
	return res.AppliedAt == "" && res.RejectedAt == "" && (res.Content != "" || res.NumPendingReplacements() > 0)
}

func (p PlanResultsByPath) SetApplied(ts string) {
	for _, planResults := range p {
		for _, planResult := range planResults {
			if !planResult.IsPending() {
				continue
			}
			planResult.AppliedAt = ts
		}
	}
}

func (p PlanResultsByPath) SetRejected(ts string) int {
	numRejected := 0
	for _, planResults := range p {
		for _, planResult := range planResults {
			if !planResult.IsPending() {
				continue
			}
			planResult.RejectedAt = ts
			numRejected++

			for _, rep := range planResult.Replacements {
				rep.SetRejected(ts)
			}
		}
	}
	return numRejected
}

func (p PlanResultsByPath) NumPending() int {
	numPending := 0
	for _, planResults := range p {
		for _, planResult := range planResults {
			if planResult.IsPending() {
				numPending++
			}
		}
	}
	return numPending
}
