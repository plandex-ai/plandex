package prompts

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildSysPrompt(filePath, currentState, desc, changes string) string {
	return listReplacementsPrompt + "\n\n" + getBuildCurrentStatePrompt(filePath, currentState) + "\n\n" + getBuildPrompt(desc, changes)
}

func getBuildPrompt(desc, changes string) string {
	s := ""

	if desc != "" {
		s += "Description of the proposed changes from AI-generated plan:\n```\n" + desc + "\n```\n\n"
	}

	withLineNums := ""
	for i, line := range strings.Split(changes, "\n") {
		withLineNums += fmt.Sprintf("%d: %s\n", i+1, line)
	}

	s += "Proposed changes:\n```\n" + withLineNums + "\n```"

	s += "\n\n" + "Now call the 'listReplacements' function with a valid JSON array of replacements according to your instructions. You must always call 'listReplacements' with one or more valid replacements. Don't call any other function."

	return s
}

func getBuildCurrentStatePrompt(filePath, currentState string) string {
	if currentState == "" {
		return ""
	}

	withLineNums := ""
	for i, line := range strings.Split(currentState, "\n") {
		withLineNums += fmt.Sprintf("%d: %s\n", i+1, line)
	}

	return fmt.Sprintf("**The current file is %s. Original state of the file:**\n```\n%s\n```", filePath, withLineNums) + "\n\n"
}

var listReplacementsPrompt = `	
	You are an AI that analyzes a code file and an AI-generated plan to update the code file and produces a list of replacements. 
	
	[YOUR INSTRUCTIONS]
	Call the 'listReplacements' function with a valid JSON array of replacements. Each replacement is an object with properties: 'shortSummary', 'changeSections', 'old', and 'new'.
	
	The 'shortSummary' property is a brief summary of the change. 
	
	'shortSummary' examples: 
		- 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
		- 'Change the value of someVar to 10.'
		- 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'

	The 'changeSections' property is a description of what section of the original file will be replaced, and what section of code from the proposed changes will replace it. Refer to sections of code using line numbers, how the section begins, and how the section ends. It must be extremely clear which line(s) of code from the original file will be replaced with which line(s) of code from the proposed changes.
	
	'changeSections' example:
	---
	The section to be replaced begins on line 10 of the original file with 'for i := 0; i < 10; i++ {...' and ends on line 15 of the original file with '}'

	The new code begins on line 17 of the proposed changes with 'for i := 0; i < 10; i++ {...' and ends on line 25 of the proposed changes with '}'
  ---

	If only a single line needs to be replaced, you can reference a single line from the original file and the proposed changes rather than a range of lines like this:
	---
	The section to be replaced is on line 10 of the original file with 'someVar = 5'

	The new code is on line 12 of the proposed changes with 'someVar = 10'
	---

	The 'old' property is an object with two properties: 'startLine' and 'endLine'.

	'startLine' is the line number where the section to be replaced begins in the original file. 'endLine' is the line number where the section to be replaced ends in the original file. For a single line replacement, 'startLine' and 'endLine' will be the same.

	The 'new' property is also an object with two properties: 'startLine' and 'endLine'.

	'startLine' is the line number where the new section begins in the code block included with the proposed changes. 'endLine' is the line number where the new section ends in the code block included with the proposed changes. For a single line replacement, 'startLine' and 'endLine' will be the same.

	Example function call with all keys:
	---
	listReplacements([{
		shortSummary: "Insert function ExecQuery after GetResults function.",
		changeSections: "The section to be replaced is on line 5 of the original file with '\\n'\n\nThe new code starts on line 3 of the proposed changes with 'func ExecQuery()...' and ends on line 10 of the proposed changes with '}'",
		old: {
			startLine: 5,
			endLine: 5,
		},
		new: {
			startLine: 3,
			endLine: 10,
		}
	}])
	---
	[END YOUR INSTRUCTIONS]
`
var ListReplacementsFn = openai.FunctionDefinition{
	Name: "listReplacements",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"replacements": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"shortSummary": {
							Type: jsonschema.String,
						},
						"changeSections": {
							Type: jsonschema.String,
						},
						"old": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"startLine": {
									Type: jsonschema.Integer,
								},
								"endLine": {
									Type: jsonschema.Integer,
								},
							},
							Required: []string{"startLine", "endLine"},
						},
						"new": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"startLine": {
									Type: jsonschema.Integer,
								},
								"endLine": {
									Type: jsonschema.Integer,
								},
							},
							Required: []string{"startLine", "endLine"},
						},
					},
					Required: []string{"shortSummary", "changeSections", "old", "new"},
				},
			},
		},
		Required: []string{"replacements"},
	},
}
