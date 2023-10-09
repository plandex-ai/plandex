package lib

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
		FilePath: "conversation_test_examples/1.md",
		NumPaths: 2,
	},
	{
		FilePath: "conversation_test_examples/2.md",
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

		counter := NewReplyTokenCounter()

		for i := 0; i < len(content); {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			chunk := content[i:end]
			counter.AddChunk(chunk)
			i = end
		}

		tokensByFilePath := counter.FinishAndRead()

		if len(tokensByFilePath) != example.NumPaths {
			t.Error(fmt.Sprintf("Expected %d file paths", example.NumPaths))
		}

	}
}

// func TestGetTokensPerFilePath(t *testing.T) {

// 	bytes, err := os.ReadFile("conversation_test.md")

// 	if err != nil {
// 		t.Error(err)
// 	}

// 	// time the function call to ensure it is fast enough
// 	start := time.Now()

// 	tokensByFilePath := getTokensPerFilePath(string(bytes))

// 	elapsed := time.Since(start)

// 	if len(tokensByFilePath) != 2 {
// 		t.Error("Expected 2 file paths")
// 	}

// 	fmt.Println("getTokensPerFilePath took ", elapsed)

// 	spew.Dump(tokensByFilePath)

// }
