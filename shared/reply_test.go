package shared

import (
	"fmt"
	"os"
	"testing"
)

type TestExample struct {
	FilePath string
	NumPaths int
}

var examples = []TestExample{
	{
		FilePath: "reply_test_examples/1.md",
		NumPaths: 2,
	},
	{
		FilePath: "reply_test_examples/2.md",
		NumPaths: 2,
	},
}

func TestReplyTokenCounter(t *testing.T) {
	for _, example := range examples {
		bytes, err := os.ReadFile(example.FilePath)
		if err != nil {
			t.Error(err)
		}

		content := string(bytes)

		chunkSize := 10

		counter := NewReplyInfo(true)

		for i := 0; i < len(content); {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			chunk := content[i:end]
			counter.AddChunk(chunk)
			i = end
		}

		files, tokensByFilePath := counter.FinishAndRead()

		if len(files) != example.NumPaths {
			t.Error(fmt.Sprintf("Expected %d file paths", example.NumPaths))
		}

		if len(tokensByFilePath) != example.NumPaths {
			t.Error(fmt.Sprintf("Expected %d file paths", example.NumPaths))
		}
	}
}
