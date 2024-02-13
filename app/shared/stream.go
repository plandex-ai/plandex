package shared

const STREAM_MESSAGE_SEPARATOR = "@@PX@@"

type BuildInfo struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
}

type StreamMessageType string

const (
	StreamMessageReply             StreamMessageType = "reply"
	StreamMessageDescribing        StreamMessageType = "describing"
	StreamMessageRepliesFinished   StreamMessageType = "repliesFinished"
	StreamMessageBuildInfo         StreamMessageType = "buildInfo"
	StreamMessagePromptMissingFile StreamMessageType = "promptMissingFile"
	StreamMessageAborted           StreamMessageType = "aborted"
	StreamMessageFinished          StreamMessageType = "finished"
	StreamMessageError             StreamMessageType = "error"
)

type StreamMessage struct {
	Type StreamMessageType `json:"type"`

	ReplyChunk string `json:"replyChunk,omitempty"`

	BuildInfo       *BuildInfo               `json:"planTokenCount,omitempty"`
	Description     *ConvoMessageDescription `json:"description,omitempty"`
	Error           *ApiError                `json:"error,omitempty"`
	MissingFilePath string                   `json:"missingFilePath,omitempty"`
	ModelStreamId   string                   `json:"modelStreamId,omitempty"`
}
