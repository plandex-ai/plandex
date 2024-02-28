package shared

type PlanStatus string

const (
	PlanStatusDraft       PlanStatus = "draft"
	PlanStatusReplying    PlanStatus = "replying"
	PlanStatusDescribing  PlanStatus = "describing"
	PlanStatusBuilding    PlanStatus = "building"
	PlanStatusMissingFile PlanStatus = "missingFile"
	PlanStatusFinished    PlanStatus = "finished"
	PlanStatusStopped     PlanStatus = "stopped"
	PlanStatusError       PlanStatus = "error"
)
