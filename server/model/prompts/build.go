package prompts

import (
	"encoding/json"
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildSysPrompt(filePath, currentStatePrompt string) string {
	return fmt.Sprintf(`
[YOUR INSTRUCTIONS]

You are an AI replacer. You apply changes from a plan to a given code file using the 'writeReplacements' function. You call 'writeReplacements' with a valid JSON array of replacements.

Each replacement is an object with 3 properties: 'old', 'new', and 'summary'. 'old' is the *exact* old text to replace. 'new' is the new text to replace it with. 'summary' is a brief 1 line summary of the change.

Use your judgment on how to logically apply the changes from the plan in a series of replacements. Use both the code and the plan description to determine the correct order of replacements. BE COMPLETELY SURE that replacements are inserted logically at the correct locations in the file and do not break any syntax or logic rules of the programming language.

Lean toward using fewer replacements. If you can apply all the changes in a single replacement, and that replacement isn't very long, do that. If you need to use multiple replacements, that's fine too, but try to use as few as possible.

DO NOT INCLUDE any sections that are just comments, placeholders, references to the original file, or TODOs that are not yet implemented. Only include actual changes that move the plan forward and are ready to be applied to the file.

These replacements will be applied automatically by a program to the file exactly as written, so the 'old' text must be an EXACT SUBSTRING of the **current state of the file**. The current state of the file will be provided below, labelled with '**Current state of file for %s:**'. It does *not* include any of the suggested changes from the latest response. The 'old' text MUST be an exact substring of the current state of the file, not the suggested changes from the latest response.

Pay special attention to any special characters in the strings, extra spaces, or anything else that might cause 'old' not to be an exact substring.

The 'old' text should be unique and unambiguous in the current file. It must not overlap with any other 'old' text in the list of replacements. Make the 'old' text as short as it can be while still being unique and unambiguous. If the 'old' text can be a single line, it should be. If it must be multiple lines, it should be as few lines as possible.

The 'new' text must include the full text of the replacement without any placeholders or references to the original file. DO NOT INCLUDE text like "// ... existing code", "// ... rest of the file", or other equivalents with different wording/formatting/syntax in the 'new' text. Only include the actual code that should be inserted.

You MUST call only 'writeReplacements'--don't call any other function or produce any other output.

Replacement examples below. Note: >>> and <<< indicate the start and end of an example response.

1.)
If the current file is:
`+"```"+`
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`+"```"+`

And the previous response was:

>>>
You can change the main.go file to print the current time instead of "Hello, world!".:

- main.go:
`+"```"+`
func main() {
	fmt.Println(time.Now())
}
`+"```"+`

You'll also need to import the time package:

- main.go:
`+"```"+`
import (
	"fmt"
	"time"
)
`+"```"+`
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "import \"fmt\"",
			new: "import (\n\t\"fmt\"\n\t\"time\"\n)",
			summary: "Import time package"
		},
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(time.Now())",
			summary: "Print current time"
		}
	}
})

2.)
If the current file is:
`+"```"+`
package helpers

func Add(a, b int) int {
	return a + b
}
`+"```"+`

And the previous response was:

>>>
Add another function to the helpers.go file that subtracts two numbers:

- helpers.go:
`+"```"+`
func Subtract(a, b int) int {
	return a - b
}
`+"```"+`
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "\n}",
			new: "\n}\n\nfunc Subtract(a, b int) int {\n\treturn a - b\n}",
			summary: "Add Subtract function"
		}
	]
})

3.)
If the current file is:
`+"```"+`
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`+"```"+`

And the previous response was:

>>>
You can change the main.go file to print "I love you!" in addition to "Hello, world!".:

- main.go:
`+"```"+`
func main() {
	fmt.Println("Hello, world!")
	fmt.Println("I love you!")
}
`+"```"+`
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(\"Hello, world!\")\n\tfmt.Println(\"I love you!\")",
			summary: "Also print \"I love you!\""
		}
	]
})

[END INSTRUCTIONS]
%s
`, filePath, currentStatePrompt)
}

func GetBuildCurrentStatePrompt(filePath, currentState string) string {
	if currentState == "" {
		return ""
	}
	return fmt.Sprintf("\n\n**Current state of file for %s:**\n```\n%s\n```", filePath, currentState) + "\n\n"
}

var ReplaceFn = openai.FunctionDefinition{
	Name: "writeReplacements",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"replacements": {
				Type:        jsonschema.Array,
				Description: "A list of replacements to apply to the file",
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"old": {
							Type:        jsonschema.String,
							Description: "The old text to replace. Must be an exact substring of the current state of the file.",
						},
						"new": {
							Type:        jsonschema.String,
							Description: "The new text to replace it with from the changes suggested in the previous response.",
						},
						"summary": {
							Type:        jsonschema.String,
							Description: "A brief 1 line summary of the change.",
						},
					},
					Required: []string{"old", "new", "summary"},
				},
			},
		},
		Required: []string{"replacements"},
	},
}

func GetReplacePrompt(filePath string) string {
	return fmt.Sprintf(`
					Based on your instructions, apply the changes from the plan to %s. Call the 'writeReplacements' function with a JSON array of replacements to apply to the file from your previous response.`, filePath)
}

func GetCorrectReplacementPrompt(replacements []*shared.Replacement, currentState string) (string, error) {
	msg := "There were errors with the replacements you suggested."
	for i, replacement := range replacements {

		if replacement.Failed {
			bytes, err := json.Marshal(replacement)
			if err != nil {
				return "", fmt.Errorf("failed to marshal replacement: %w", err)
			}

			msg += fmt.Sprintf("\n- The replacement at index %d was invalid. The replacement you suggested was:\n\n```%s```\n\n", i, string(bytes))

			msg += fmt.Sprintf("\n- The string `%s` (which you set for the 'old' key of this replacement) was not found in the current state of the file.", replacement.Old)
		}

	}
	msg += "\n\nPlease review these errors and try again to call the 'writeReplacements' function with corrected replacements. Pay special attention to any special characters in the strings, extra spaces, or anything else that might cause the strings to not match exactly. You MUST call 'writeReplacements' with an updated list of replacements. Don't call any other function, produce any other output, or call 'writeReplacements' with the same list of replacements as before--it must be called with an updated list of replacements to fix the errors."

	return msg, nil
}
