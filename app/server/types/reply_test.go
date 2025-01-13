package types

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/plandex/plandex/shared"
)

type TestExample struct {
	Only       bool
	Operations []shared.Operation
}

// These aren't the real number of tokens
// We're just splitting the file into chunks of 5 characters to simulate tokens
var examples = []TestExample{
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/apply.go",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/checkout.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/context_rm.go",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/context_update.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/context_rm.go",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "cmd/context_update.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "server/types/section.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "shared/types.go",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "cli/lib/conversation.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "server/model/proposal/create.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "file_map/map.go",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "Makefile",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "_apply.sh",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type:        shared.OperationTypeMove,
				Path:        "src/game.c",
				Destination: "src/game/game.c",
			},
			{
				Type:        shared.OperationTypeMove,
				Path:        "src/game.h",
				Destination: "src/game/game.h",
			},
			{
				Type: shared.OperationTypeRemove,
				Path: "src/README.md",
			},
			{
				Type: shared.OperationTypeReset,
				Path: "Makefile",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "Makefile",
			},
		},
	},
	{
		Operations: []shared.Operation{
			{
				Type: shared.OperationTypeFile,
				Path: "server/model/prompts/describe.go",
			},
			{
				Type: shared.OperationTypeFile,
				Path: "server/model/plan/commit_msg.go",
				Description: `Now let's update the commit message handling in commit_msg.go:

**Updating ` + "`server/model/plan/commit_msg.go`" + `:** I'll update the genPlanDescription method to handle XML output instead of JSON.`,
			},
		},
	},
}

func TestReplyParser(t *testing.T) {
	only := map[int]bool{}
	for i, example := range examples {
		if example.Only {
			only[i] = true
		}
	}

	for i, example := range examples {

		if len(only) > 0 && !only[i] {
			continue
		}

		t.Run(fmt.Sprintf("Example_%d", i+1), func(t *testing.T) {
			filePath := fmt.Sprintf("reply_test_examples/%d.md", i+1)
			fmt.Println(filePath)

			bytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Error(err)
			}

			content := string(bytes)
			tokenSize := 5

			parser := NewReplyParser()

			for i := 0; i < len(content); {
				end := i + tokenSize
				if end > len(content) {
					end = len(content)
				}
				chunk := content[i:end]
				parser.AddChunk(chunk, true)
				i = end
			}

			res := parser.FinishAndRead()

			operations := res.Operations

			if len(operations) != len(example.Operations) {
				t.Errorf("Example %d: Expected %d operations, got %d",
					i+1, len(example.Operations), len(operations))
			}

			for j, operation := range operations {
				if operation.Name() != example.Operations[j].Name() {
					t.Errorf("Example %d: Expected operation %s, got %s",
						i+1, example.Operations[j].Name(), operation.Name())
				}

				if example.Operations[j].Description != "" {
					if operation.Description != example.Operations[j].Description {
						t.Errorf("Example %d: Expected description %s, got %s",
							i+1, strconv.Quote(example.Operations[j].Description), strconv.Quote(operation.Description))
					}
				}
			}

		})
	}
}
