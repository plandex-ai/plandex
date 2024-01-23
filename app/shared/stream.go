package shared

import "github.com/looplab/fsm"

const STREAM_MESSAGE_SEPARATOR = "@@PX@@"
const STREAM_FINISHED = "@@PXEND@@"
const STREAM_DESCRIPTION_PHASE = "@@PXDESC@@"
const STREAM_RESUME = "@@PXRESUME@@"
const STREAM_ABORTED = "@@PXABORT@@"

const EVENT_DESCRIBE = "describe"
const EVENT_FINISH = "finish"
const EVENT_RESUME = "resume"
const EVENT_ABORT = "abort"
const EVENT_CANCEL = "cancel"
const EVENT_ERROR = "error"

const STATE_REPLYING = "replying"
const STATE_DESCRIBING = "describing"
const STATE_FINISHED = "finished"
const STATE_ABORTED = "aborted"
const STATE_CANCELED = "canceled"
const STATE_ERROR = "error"

func NewPlanStreamState() *fsm.FSM {
	sm := fsm.NewFSM(
		STATE_REPLYING,
		fsm.Events{
			// Define state transitions
			{Name: EVENT_DESCRIBE, Src: []string{STATE_REPLYING}, Dst: STATE_DESCRIBING},

			{Name: EVENT_RESUME, Src: []string{STATE_DESCRIBING}, Dst: STATE_REPLYING},

			{Name: EVENT_FINISH, Src: []string{STATE_DESCRIBING}, Dst: STATE_FINISHED},

			{Name: EVENT_ABORT, Src: []string{STATE_REPLYING, STATE_DESCRIBING}, Dst: STATE_ABORTED},

			{Name: EVENT_CANCEL, Src: []string{STATE_ABORTED}, Dst: STATE_CANCELED},

			{Name: EVENT_ERROR, Src: []string{STATE_REPLYING, STATE_DESCRIBING}},
		},
		fsm.Callbacks{},
	)

	return sm
}

type PlanTokenCount struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
}
