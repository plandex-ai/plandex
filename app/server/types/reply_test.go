package types

import (
	"fmt"
	"os"
	"testing"
)

type TestExample struct {
	N                int
	TokensByFilePath map[string]int
}

// These aren't the real number of tokens
// We're just splitting the file into chunks of 5 characters to simulate tokens
var examples = []TestExample{
	{
		N: 1,
		TokensByFilePath: map[string]int{
			"cmd/checkout.go": 54,
			"cmd/apply.go":    180,
		},
	},
	{
		N: 2,
		TokensByFilePath: map[string]int{
			"cmd/context_rm.go":     210,
			"cmd/context_update.go": 188,
		},
	},
	{
		N: 3,
		TokensByFilePath: map[string]int{
			"cmd/context_rm.go":     210,
			"cmd/context_update.go": 188,
		},
	},
	{
		N: 4,
		TokensByFilePath: map[string]int{
			"server/types/section.go": 32,
		},
	},
	{
		N: 5,
		TokensByFilePath: map[string]int{
			"shared/types.go":         20,
			"cli/lib/conversation.go": 58,
		},
	},
	{
		N: 6,
		TokensByFilePath: map[string]int{
			"server/model/proposal/create.go": 239,
		},
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

		tokenSize := 5

		counter := NewReplyParser()

		totalTokens := 0
		for i := 0; i < len(content); {
			end := i + tokenSize
			if end > len(content) {
				end = len(content)
			}
			chunk := content[i:end]
			counter.AddChunk(chunk, true)
			totalTokens++
			i = end
		}

		res := counter.FinishAndRead()

		totalCounted := res.TotalTokens
		files := res.Files
		fileContents := res.FileContents
		tokensByFilePath := res.NumTokensByFile

		fmt.Printf("Total tokens counted: %d\n", totalCounted)
		fmt.Printf("%d files: %v\n", len(files), files)
		fmt.Printf("%d file content: %v\n", len(fileContents), fileContents)
		fmt.Println("Tokens by file path:")
		for filePath, tokens := range tokensByFilePath {
			fmt.Printf("%s: %d\n", filePath, tokens)
		}

		if totalCounted != totalTokens {
			t.Errorf("Expected %d tokens, got %d", totalTokens, totalCounted)
		}

		for filePath, tokens := range example.TokensByFilePath {
			if tokensByFilePath[filePath] != tokens {
				t.Errorf("Expected %d tokens for %s, got %d", tokens, filePath, tokensByFilePath[filePath])
			}
		}
	}
}
