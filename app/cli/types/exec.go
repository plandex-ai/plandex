package types

type TellFlags struct {
	TellBg               bool
	TellStop             bool
	TellNoBuild          bool
	IsUserContinue       bool
	IsUserDebug          bool
	IsApplyDebug         bool
	IsChatOnly           bool
	AutoContext          bool
	ContinuedAfterAction bool
	ExecEnabled          bool
	AutoApply            bool
}
type BuildFlags struct {
	BuildBg   bool
	AutoApply bool
}
