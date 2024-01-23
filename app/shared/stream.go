package shared

const STREAM_MESSAGE_SEPARATOR = "@@PX@@"

type BuildInfo struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
}

type StreamMessageType string

const (
	StreamMessageReply       StreamMessageType = "reply"
	StreamMessageDescribing  StreamMessageType = "describing"
	StreamMessageDescription StreamMessageType = "description"
	StreamMessageBuildInfo   StreamMessageType = "buildInfo"
	StreamMessageAborted     StreamMessageType = "aborted"
	StreamMessageFinished    StreamMessageType = "finished"
	StreamMessageError       StreamMessageType = "error"
)

type StreamMessage struct {
	Type StreamMessageType `json:"type"`

	ReplyChunk  string                   `json:"replyChunk,omitempty"`
	BuildInfo   *BuildInfo               `json:"planTokenCount,omitempty"`
	Description *ConvoMessageDescription `json:"description,omitempty"`
	Error       string                   `json:"error,omitempty"`
}
