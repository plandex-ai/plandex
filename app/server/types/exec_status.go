package types

type PlanExecStatus struct {
	NeedsInput bool `json:"needsInput"`
	Finished   bool `json:"finished"`
}
