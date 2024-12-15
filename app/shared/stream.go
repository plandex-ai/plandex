package shared

const STREAM_MESSAGE_SEPARATOR = "@@PX@@"

type BuildInfo struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
	Removed   bool   `json:"removed,omitempty"`
}

type StreamMessageType string

const (
	StreamMessageStart             StreamMessageType = "start"
	StreamMessageConnectActive     StreamMessageType = "connectActive"
	StreamMessageReply             StreamMessageType = "reply"
	StreamMessageDescribing        StreamMessageType = "describing"
	StreamMessageRepliesFinished   StreamMessageType = "repliesFinished"
	StreamMessageBuildInfo         StreamMessageType = "buildInfo"
	StreamMessagePromptMissingFile StreamMessageType = "promptMissingFile"
	StreamMessageLoadContext       StreamMessageType = "loadContext"
	StreamMessageAborted           StreamMessageType = "aborted"
	StreamMessageFinished          StreamMessageType = "finished"
	StreamMessageError             StreamMessageType = "error"

	StreamMessageMulti StreamMessageType = "multi"
)

type StreamMessage struct {
	Type StreamMessageType `json:"type"`

	ReplyChunk string `json:"replyChunk,omitempty"`

	BuildInfo              *BuildInfo               `json:"buildInfo,omitempty"`
	Description            *ConvoMessageDescription `json:"description,omitempty"`
	Error                  *ApiError                `json:"error,omitempty"`
	MissingFilePath        string                   `json:"missingFilePath,omitempty"`
	MissingFileAutoContext bool                     `json:"missingFileAutoContext,omitempty"`
	ModelStreamId          string                   `json:"modelStreamId,omitempty"`
	LoadContextFiles       []string                 `json:"loadContextFiles,omitempty"`
	InitPrompt             string                   `json:"initPrompt,omitempty"`
	InitReplies            []string                 `json:"initReplies,omitempty"`
	InitBuildOnly          bool                     `json:"initBuildOnly,omitempty"`

	StreamMessages []StreamMessage `json:"streamMessages,omitempty"`
}
