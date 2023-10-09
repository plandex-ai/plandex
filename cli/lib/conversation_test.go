package lib

import (
	"os"
	"testing"
)

func TestReplyTokenCounter(t *testing.T) {
	bytes, err := os.ReadFile("conversation_test.md")
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

	if len(tokensByFilePath) != 2 {
		t.Error("Expected 2 file paths")
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
