package types

type TellFlags struct {
	TellBg                 bool
	TellStop               bool
	TellNoBuild            bool
	IsUserContinue         bool
	IsUserDebug            bool
	IsApplyDebug           bool
	IsChatOnly             bool
	AutoContext            bool
	SmartContext           bool
	ContinuedAfterAction   bool
	ExecEnabled            bool
	AutoApply              bool
	IsImplementationOfChat bool
	SkipChangesMenu        bool
}
type BuildFlags struct {
	BuildBg   bool
	AutoApply bool
}
