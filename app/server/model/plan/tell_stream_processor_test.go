package plan

import (
	"plandex-server/types"
	shared "plandex-shared"
	"testing"
)

func TestBufferOrStream(t *testing.T) {
	tests := []struct {
		only            bool
		name            string
		initialState    *chunkProcessor
		chunk           string
		maybeFilePath   string
		currentFilePath string
		isInMoveBlock   bool
		isInRemoveBlock bool
		isInResetBlock  bool
		want            bufferOrStreamResult
		wantState       *chunkProcessor // To verify state transitions
	}{
		{
			name: "streams regular content",
			initialState: &chunkProcessor{
				contentBuffer: "",
			},
			chunk: "some regular text",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "some regular text",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                false,
			},
		},
		{
			name: "buffers partial opening tag",
			initialState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
				fileOpen:                false,
				contentBuffer:           "",
			},
			chunk:           `<Pland`,
			maybeFilePath:   "main.go",
			currentFilePath: "",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                false,
				contentBuffer:           "<Pland",
			},
		},
		{
			name: "converts opening tag",
			initialState: &chunkProcessor{
				fileOpen:                true,
				contentBuffer:           `<PlandexBlock lang="go" path="main.go">` + "\n",
				awaitingBlockOpeningTag: true,
			},
			chunk:           `package`,
			maybeFilePath:   "",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                true,
			},
		},
		{
			// occurs when replayParser can't identify a 'maybeFilePath' prior a full opening tag being sent ('maybeFilePath' gets skipped and 'currentFilePath' is set immediately)
			name: "converts opening tag without awaitingOpeningTag",
			initialState: &chunkProcessor{
				fileOpen:                true,
				contentBuffer:           "",
				awaitingBlockOpeningTag: false,
			},
			chunk:           `<PlandexBlock lang="go" path="main.go">` + "\npackage",
			maybeFilePath:   "",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                true,
			},
		},
		{
			name: "buffers partial backticks",
			initialState: &chunkProcessor{
				fileOpen:      true,
				contentBuffer: "here's some co",
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
				contentBuffer:     "here's some code:\n`",
			},
			chunk:           "``\npackage",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "here's some code:\n\\`\\`\\`\npackage",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                true,
			},
		},
		{
			name: "buffers partial closing tag",
			initialState: &chunkProcessor{
				fileOpen:                true,
				awaitingBlockClosingTag: false,
				contentBuffer:           "",
			},
			currentFilePath: "main.go",
			chunk:           "\n}</Plan",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockClosingTag: true,
				fileOpen:                true,
				contentBuffer:           "\n}</Plan",
			},
		},
		{
			name: "buffers full closing tag with file open",
			initialState: &chunkProcessor{
				fileOpen:                true,
				awaitingBlockClosingTag: false,
				contentBuffer:           "",
			},
			currentFilePath: "main.go",
			chunk:           "\n}</PlandexBlock>",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockClosingTag: true,
				fileOpen:                true,
				contentBuffer:           "\n}</PlandexBlock>",
			},
		},
		{
			name: "replaces full closing tag with file closed",
			initialState: &chunkProcessor{
				fileOpen:                false,
				awaitingBlockClosingTag: false,
				contentBuffer:           "",
			},
			currentFilePath: "",
			chunk:           "\n}</PlandexBlock>",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "\n}```",
			},
			wantState: &chunkProcessor{
				awaitingBlockClosingTag: false,
				fileOpen:                false,
			},
		},
		{
			name: "replaces full closing tag with file closed and awaiting backticks",
			initialState: &chunkProcessor{
				fileOpen:                false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       true,
				contentBuffer:           "",
			},
			currentFilePath: "",
			chunk:           " ONLY this one-line title and nothing else.`\n</PlandexBlock>\n\nNow let",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      " ONLY this one-line title and nothing else.`\n```\n\nNow let",
			},
			wantState: &chunkProcessor{
				awaitingBlockClosingTag: false,
				fileOpen:                false,
			},
		},
		{
			name: "handles single backticks",
			initialState: &chunkProcessor{
				fileOpen:          true,
				awaitingBackticks: true,
				contentBuffer:     "`file.go`",
			},
			chunk:           "\nsomething",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "`file.go`\nsomething",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       false,
				fileOpen:                true,
			},
		},
		{
			name: "handles close and re-open backticks",
			initialState: &chunkProcessor{
				fileOpen:          true,
				awaitingBackticks: true,
				contentBuffer:     "`file.go`",
			},
			chunk:           "\n`file2.go`",
			currentFilePath: "main.go",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				awaitingBlockClosingTag: false,
				awaitingBackticks:       true,
				fileOpen:                true,
				contentBuffer:           "`file.go`\n`file2.go`",
			},
		},
		{
			name:          "buffers for end of file operations",
			initialState:  &chunkProcessor{},
			isInMoveBlock: true,
			chunk:         "\n<EndPlandexFileOps/>\nmore",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingOpClosingTag: true,
				contentBuffer:        "\n<EndPlandexFileOps/>\nmore",
			},
		},
		{
			name: "replaces full end of file operations tag",
			initialState: &chunkProcessor{
				awaitingOpClosingTag: true,
				contentBuffer:        "\n<EndPlandexFileOps/>\nmore",
			},
			chunk: " stuff",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "\nmore stuff",
			},
			wantState: &chunkProcessor{
				awaitingOpClosingTag: false,
			},
		},
		{
			name: "buffers for end of file operations with partial tag",
			initialState: &chunkProcessor{
				awaitingOpClosingTag: true,
			},
			chunk: "\n<EndPlandex",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingOpClosingTag: true,
				contentBuffer:        "\n<EndPlandex",
			},
		},
		{
			name: "replaces end of file operation closing partial tag",
			initialState: &chunkProcessor{
				awaitingOpClosingTag: true,
				contentBuffer:        "\n<EndPlandex",
			},
			chunk: "FileOps/>\nmore",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "\nmore",
			},
			wantState: &chunkProcessor{
				awaitingOpClosingTag: false,
			},
		},
		{
			name:         "buffers for partial opening tag with no file path label",
			initialState: &chunkProcessor{},
			chunk:        "something<Pland",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
			},
		},
		{
			name: "continues buffering partial opening tag with no file path label",
			initialState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
				contentBuffer:           "something<Pland",
			},
			chunk: "exBlock lang=\"go\" path=\"main",
			want: bufferOrStreamResult{
				shouldStream: false,
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
				contentBuffer:           "something<PlandexBlock lang=\"go\" path=\"main",
			},
		},
		{
			name: "replaces opening tag with no file path label when it completes",
			initialState: &chunkProcessor{
				awaitingBlockOpeningTag: true,
				contentBuffer:           "something\n<Pland",
				fileOpen:                true,
			},
			currentFilePath: "main.go",
			chunk:           "exBlock lang=\"go\" path=\"main.go\">\npackage",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "something\n```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				fileOpen:                true,
			},
		},
		{
			name: "replaces full opening tag without file path label",
			initialState: &chunkProcessor{
				fileOpen: true,
			},
			currentFilePath: "main.go",
			chunk:           "something\n<PlandexBlock lang=\"go\" path=\"main.go\">\npackage",
			want: bufferOrStreamResult{
				shouldStream: true,
				content:      "something\n```go\npackage",
			},
			wantState: &chunkProcessor{
				awaitingBlockOpeningTag: false,
				fileOpen:                true,
			},
		},
	}

	only := map[int]bool{}
	for i, tt := range tests {
		if tt.only {
			only[i] = true
		}
	}

	for i, tt := range tests {
		if len(only) > 0 && !only[i] {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			processor := tt.initialState

			got := processor.bufferOrStream(tt.chunk, &types.ReplyParserRes{
				MaybeFilePath:   tt.maybeFilePath,
				CurrentFilePath: tt.currentFilePath,
				IsInMoveBlock:   tt.isInMoveBlock,
				IsInRemoveBlock: tt.isInRemoveBlock,
				IsInResetBlock:  tt.isInResetBlock,
			}, shared.CurrentStage{
				TellStage: shared.TellStageImplementation,
			})

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
			if processor.awaitingBlockOpeningTag != tt.wantState.awaitingBlockOpeningTag {
				t.Errorf("awaitingOpeningTag = %v, want %v", processor.awaitingBlockOpeningTag, tt.wantState.awaitingBlockOpeningTag)
			}
			if processor.awaitingBlockClosingTag != tt.wantState.awaitingBlockClosingTag {
				t.Errorf("awaitingClosingTag = %v, want %v", processor.awaitingBlockClosingTag, tt.wantState.awaitingBlockClosingTag)
			}
			if processor.awaitingBackticks != tt.wantState.awaitingBackticks {
				t.Errorf("awaitingBackticks = %v, want %v", processor.awaitingBackticks, tt.wantState.awaitingBackticks)
			}

			if tt.wantState.contentBuffer != "" {
				if processor.contentBuffer != tt.wantState.contentBuffer {
					t.Errorf("content buffer = %q, want %q", processor.contentBuffer, tt.wantState.contentBuffer)
				}
			}

			// Check buffer is reset when it should be
			if tt.want.shouldStream && processor.contentBuffer != "" {
				t.Error("content buffer should be reset after streaming")
			}
		})
	}
}
