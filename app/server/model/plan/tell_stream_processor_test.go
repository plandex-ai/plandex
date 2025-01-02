package plan

import (
	"strings"
	"testing"
)

func TestBufferOrStream(t *testing.T) {
	tests := []struct {
		name            string
		initialState    *chunkProcessor
		chunk           string
		maybeFilePath   string
		currentFilePath string
		want            bufferOrStreamResult
		wantState       *chunkProcessor // To verify state transitions
	}{
		{
			name: "streams regular content",
			initialState: &chunkProcessor{
				contentBuffer: &strings.Builder{},
			},
			chunk: "some regular text",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "some regular text",
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           false,
			},
		},
		{
			name: "buffers partial opening tag",
			initialState: &chunkProcessor{
				awaitingOpeningTag: true,
				fileOpen:           false,
				contentBuffer:      &strings.Builder{},
			},
			chunk:           `<Pland`,
			maybeFilePath:   "main.go",
			currentFilePath: "",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: true,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           false,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString(`<Pland`)
					return b
				}(),
			},
		},
		{
			name: "converts opening tag",
			initialState: &chunkProcessor{
				fileOpen: true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString(`<PlandexBlock lang="go">` + "\n")
					return b
				}(),
				awaitingOpeningTag: true,
			},
			chunk:           `package`,
			maybeFilePath:   "",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           true,
			},
		},
		{
			// occurs when replayParser can't identify a 'maybeFilePath' prior a full opening tag being sent ('maybeFilePath' gets skipped and 'currentFilePath' is set immediately)
			name: "converts opening tag without awaitingOpeningTag",
			initialState: &chunkProcessor{
				fileOpen:           true,
				contentBuffer:      &strings.Builder{},
				awaitingOpeningTag: false,
			},
			chunk:           `<PlandexBlock lang="go">` + "\npackage",
			maybeFilePath:   "",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           true,
			},
		},
		{
			name: "buffers partial backticks",
			initialState: &chunkProcessor{
				fileOpen: true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("here's some co")
					return b
				}(),
			},
			chunk:           "de:`",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBackticks: true,
				fileOpen:          true,
			},
		},
		{
			name: "escapes backticks in content",
			initialState: &chunkProcessor{
				fileOpen:          true,
				awaitingBackticks: true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("here's some code:\n`")
					return b
				}(),
			},
			chunk:           "``\npackage",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "here's some code:\n\\`\\`\\`\npackage",
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           true,
			},
		},
		{
			name: "buffers partial closing tag",
			initialState: &chunkProcessor{
				fileOpen:           true,
				awaitingClosingTag: false,
				contentBuffer:      &strings.Builder{},
			},
			currentFilePath: "main.go",
			chunk:           "\n}</Plan",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingClosingTag: true,
				fileOpen:           true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("\n}</Plan")
					return b
				}(),
			},
		},
		{
			name: "buffers full closing tag with file open",
			initialState: &chunkProcessor{
				fileOpen:           true,
				awaitingClosingTag: false,
				contentBuffer:      &strings.Builder{},
			},
			currentFilePath: "main.go",
			chunk:           "\n}</PlandexBlock>",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingClosingTag: true,
				fileOpen:           true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("\n}</PlandexBlock>")
					return b
				}(),
			},
		},
		{
			name: "replaces full closing tag with file closed",
			initialState: &chunkProcessor{
				fileOpen:           false,
				awaitingClosingTag: false,
				contentBuffer:      &strings.Builder{},
			},
			currentFilePath: "",
			chunk:           "\n}</PlandexBlock>",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "\n}```",
			},
			wantState: &chunkProcessor{
				awaitingClosingTag: false,
				fileOpen:           false,
			},
		},
		{
			name: "handles single backticks",
			initialState: &chunkProcessor{
				fileOpen:          true,
				awaitingBackticks: true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("`file.go`")
					return b
				}(),
			},
			chunk:           "\nsomething",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "`file.go`\nsomething",
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  false,
				fileOpen:           true,
			},
		},
		{
			name: "handles close and re-open backticks",
			initialState: &chunkProcessor{
				fileOpen:          true,
				awaitingBackticks: true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("`file.go`")
					return b
				}(),
			},
			chunk:           "\n`file2.go`",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingOpeningTag: false,
				awaitingClosingTag: false,
				awaitingBackticks:  true,
				fileOpen:           true,
				contentBuffer: func() *strings.Builder {
					b := &strings.Builder{}
					b.WriteString("`file.go`\n`file2.go`")
					return b
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := tt.initialState

			// Initialize content buffer if needed
			if processor.contentBuffer == nil {
				processor.contentBuffer = &strings.Builder{}
			}

			got := processor.bufferOrStream(tt.chunk, tt.maybeFilePath, tt.currentFilePath)

			if got.shouldStream != tt.want.shouldStream {
				t.Errorf("shouldStream = %v, want %v", got.shouldStream, tt.want.shouldStream)
			}
			if got.shouldStream && got.content != tt.want.content {
				t.Errorf("content = %q, want %q", got.content, tt.want.content)
			}

			// Check all state transitions
			if processor.fileOpen != tt.wantState.fileOpen {
				t.Errorf("fileOpen = %v, want %v", processor.fileOpen, tt.wantState.fileOpen)
			}
			if processor.awaitingOpeningTag != tt.wantState.awaitingOpeningTag {
				t.Errorf("awaitingOpeningTag = %v, want %v", processor.awaitingOpeningTag, tt.wantState.awaitingOpeningTag)
			}
			if processor.awaitingClosingTag != tt.wantState.awaitingClosingTag {
				t.Errorf("awaitingClosingTag = %v, want %v", processor.awaitingClosingTag, tt.wantState.awaitingClosingTag)
			}
			if processor.awaitingBackticks != tt.wantState.awaitingBackticks {
				t.Errorf("awaitingBackticks = %v, want %v", processor.awaitingBackticks, tt.wantState.awaitingBackticks)
			}

			if tt.wantState.contentBuffer != nil {
				if processor.contentBuffer.String() != tt.wantState.contentBuffer.String() {
					t.Errorf("content buffer = %q, want %q", processor.contentBuffer.String(), tt.wantState.contentBuffer.String())
				}
			}

			// Check buffer is reset when it should be
			if tt.want.shouldStream && processor.contentBuffer.Len() > 0 {
				t.Error("content buffer should be reset after streaming")
			}
		})
	}
}
