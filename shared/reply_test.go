package shared

import (
	"fmt"
	"os"
	"testing"
)

type TestExample struct {
	N        int
	NumPaths int
}

var examples = []TestExample{
	{
		N:        1,
		NumPaths: 2,
	},
	{
		N:        2,
		NumPaths: 2,
	},
	{
		N:        3,
		NumPaths: 2,
	},
	{
		N:        4,
		NumPaths: 1,
	},
	{
		N:        5,
		NumPaths: 2,
	},
	{
		N:        6,
		NumPaths: 1,
	},
}

func TestReplyTokenCounter(t *testing.T) {

	for _, example := range examples {
		filePath := fmt.Sprintf("reply_test_examples/%d.md", example.N)
		fmt.Println(filePath)

		bytes, err := os.ReadFile(filePath)
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

		fmt.Printf("%d files: %v\n", len(files), files)

		if len(files) != example.NumPaths {
			t.Error(fmt.Sprintf("Expected %d file paths", example.NumPaths))
		}

		if len(tokensByFilePath) != example.NumPaths {
			t.Error(fmt.Sprintf("Expected %d file paths", example.NumPaths))
		}
	}
}
