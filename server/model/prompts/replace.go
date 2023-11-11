package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysReplace = `
[YOUR INSTRUCTIONS]

You are an AI replacer. You apply changes from a plan to a given code file using the 'writeReplacements' function.

Call 'writeReplacements' with a list of replacements to apply to the file. Each replacement is an object with two properties: 'old' and 'new'. 'old' is the old text to replace, and 'new' is the new text to replace it with. You can include as many replacements as you want. You must include at least one replacement.
- The 'new' text must include the full text of the replacement without any placeholders or references to the original file.
- The 'old' text *ABSOLUTELY MUST BE AN EXACT SUBSTRING* of the current state of the file.
- The 'old' text must not overlap with any other 'old' text in the list of replacements.
- BE COMPLETELY SURE that replacements are inserted logically at the correct locations in the file and do not break any syntax or logic rules of the programming language.

Replacement examples below. Note: >>> and <<< indicate the start and end of an example response.

1.)
If the current file is:
` + "```" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```" + `

And the previous response was:

>>>
You can change the main.go file to print the current time instead of "Hello, world!".:

- main.go:
` + "```" + `
func main() {
	fmt.Println(time.Now())
}
` + "```" + `

You'll also need to import the time package:

- main.go:
` + "```" + `
import (
	"fmt"
	"time"
)
` + "```" + `
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "import \"fmt\"",
			new: "import (\n\t\"fmt\"\n\t\"time\"\n)"
		},
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(time.Now())"
		}
	}
})

2.)
If the current file is:
` + "```" + `
package helpers

func Add(a, b int) int {
	return a + b
}
` + "```" + `

And the previous response was:

>>>
Add another function to the helpers.go file that subtracts two numbers:

- helpers.go:
` + "```" + `
func Subtract(a, b int) int {
	return a - b
}
` + "```" + `
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "\n}",
			new: "\n}\n\nfunc Subtract(a, b int) int {\n\treturn a - b\n}"
		}
	]
})

3.)
If the current file is:
` + "```" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```" + `

And the previous response was:

>>>
You can change the main.go file to print "I love you!" in addition to "Hello, world!".:

- main.go:
` + "```" + `
func main() {
	fmt.Println("Hello, world!")
	fmt.Println("I love you!")
}
` + "```" + `						
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(\"Hello, world!\")\n\tfmt.Println(\"I love you!\")"
		}
	]
})

[END INSTRUCTIONS]`

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
							Description: "The new text to replace it with. Must be an exact substring of the suggested changes from the plan in the previous response.",
						},
					},
					Required: []string{"old", "new"},
				},
			},
		},
		Required: []string{"replacements"},
	},
}

func GetReplacePrompt(filePath string) string {
	return fmt.Sprintf(`
					Based on your instructions, apply the changes from the plan to %s. Call the 'writeReplacements' function with a JSON array of replacements to apply to the file from your previous response. Each replacement is an object with two properties: 'old' and 'new'. 'old' is the old text to replace, and 'new' is the new text to replace it with. The 'old' text MUST be an exact substring of the current state of the file. The 'new' text MUST be an exact substring of the changes from the plan in the previous response. You MUST call only the 'writeReplacements'--don't call any other function or produce any other output.
					`, filePath)
}

func GetCorrectReplacementPrompt(failedReplacements map[int]*shared.Replacement) string {
	msg := "There were errors with the replacements you suggested."
	for index, failedReplacement := range failedReplacements {
		msg += fmt.Sprintf("\n\nError in replacement at index %d:", index)
		msg += fmt.Sprintf("\n- The string '%s' (which you set for the 'old' key) was not found in the file.", failedReplacement.Old)
	}
	msg += "\n\nPlease review these errors and try again to call the 'writeReplacements' function with corrected replacements. Pay special attention to any special characters in the strings, extra spaces, or anything else that might cause the strings to not match exactly."

	return msg
}
