package shared

import "github.com/looplab/fsm"

const STREAM_MESSAGE_SEPARATOR = "@@PX@@"
const STREAM_FINISHED = "@@PXEND@@"
const STREAM_DESCRIPTION_PHASE = "@@PXDESC@@"
const STREAM_BUILD_PHASE = "@@PXBUILD@@"
const STREAM_ABORTED = "@@PXABORT@@"

const EVENT_DESCRIBE = "describe"
const EVENT_BUILD = "build"
const EVENT_FINISH = "finish"
const EVENT_ABORT = "abort"
const EVENT_REVISE = "revise"
const EVENT_CANCEL = "cancel"
const EVENT_ERROR = "error"

const STATE_REPLYING = "replying"
const STATE_DESCRIBING = "describing"
const STATE_BUILDING = "building"
const STATE_FINISHED = "finished"
const STATE_ABORTED = "aborted"
const STATE_REVISING = "revising"
const STATE_CANCELED = "canceled"
const STATE_ERROR = "error"

func NewPlanStreamState() *fsm.FSM {
	sm := fsm.NewFSM(
		STATE_REPLYING,
		fsm.Events{
			// Define state transitions
			{Name: EVENT_DESCRIBE, Src: []string{STATE_REPLYING, STATE_REPLYING}, Dst: STATE_DESCRIBING},
			{Name: EVENT_BUILD, Src: []string{STATE_DESCRIBING}, Dst: STATE_BUILDING},
			{Name: EVENT_FINISH, Src: []string{STATE_DESCRIBING, STATE_BUILDING}, Dst: STATE_FINISHED},
			{Name: EVENT_ABORT, Src: []string{STATE_REPLYING, STATE_DESCRIBING, STATE_BUILDING, STATE_REVISING},
				Dst: STATE_ABORTED},
			{Name: EVENT_REVISE, Src: []string{STATE_ABORTED}, Dst: STATE_REPLYING},
			{Name: EVENT_CANCEL, Src: []string{STATE_ABORTED}, Dst: STATE_CANCELED},
			{Name: EVENT_ERROR, Src: []string{STATE_REPLYING, STATE_DESCRIBING, STATE_BUILDING, STATE_REVISING}},
		},
		fsm.Callbacks{},
	)

	return sm
}

type StreamedFile struct {
	Content string `json:"content"`
}
