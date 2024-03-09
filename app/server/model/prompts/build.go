package prompts

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildSysPrompt(filePath, currentState, desc, changes string) string {
	return listChangesPrompt + "\n\n" + getBuildCurrentStatePrompt(filePath, currentState) + "\n\n" + getBuildPrompt(desc, changes)
}

func getBuildPrompt(desc, changes string) string {
	s := ""

	if desc != "" {
		s += "Description of the proposed updates from AI-generated plan:\n```\n" + desc + "\n```\n\n"
	}

	withLineNums := ""
	for i, line := range strings.Split(changes, "\n") {
		withLineNums += fmt.Sprintf("%d: %s\n", i+1, line)
	}

	s += "Proposed updates:\n```\n" + withLineNums + "\n```"

	s += "\n\n" + "Now call the 'listChanges' function with a valid JSON array of changes according to your instructions. You must always call 'listChanges' with one or more valid changes. Don't call any other function."

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

var listChangesPrompt = `	
	You are an AI that analyzes a code file and an AI-generated plan to update the code file and produces a list of changes.
	
	There are four types of changes:
	
	1 - Replace: replace a section of code in the original file with a section of code from the proposed updates.

	2 - Append: add a section of code from the proposed update immediately after a section of code in the original file.

	3 - Prepend: add a section of code from the proposed updates immediately before a section of code in the original file.

	[YOUR INSTRUCTIONS]
	Call the 'listChanges' function with a valid JSON array of changes. Each replacement is an object with properties: 'shortSummary', 'changeSections', 'changeType', 'old', and 'new'.
	
	The 'shortSummary' property is a brief summary of the change. 
	
	'shortSummary' examples: 
		- 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
		- 'Change the value of someVar to 10.'
		- 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
		- 'Add function ExecQuery to execute a query.'

	The 'changeSections' property is a description of: 
		- The type of change (replace, append, prepend).
		- If the type of change is 'replace', what section of code from the original file will be replaced.
		- If the type of change is 'append', what section of code from the original file will have new code appended immediately after it.
		- If the type of change is 'prepend', what section of code from the original file will have new code prepended immediately before it.
		- What section of code from the proposed updates will be added.
	
	Refer to sections of code using line numbers, how the section begins, and how the section ends. 
	
	If the change type is 'replace', it must be extremely clear which line(s) of code from the original file will be replaced with which line(s) of code from the proposed updates.

	If the change type is 'append' or 'prepend', it must be extremely clear which line(s) of code from the original file will have which line(s) of code from the proposed updates appended or prepended to them.
	
	'changeSections' examples:
	---
	Type: replace

	The section to be replaced begins on line 10 of the original file with 'for i := 0; i < 10; i++ {...' and ends on line 15 of the original file with '}'

	The new code begins on line 17 of the proposed updates with 'for i := 0; i < 10; i++ {...' and ends on line 25 of the proposed updates with '}'
  ---

	---
	Type: append

	The section that should have code appended to it begins on line 10 of the original file with 'func GetResults() {...' and ends on line 15 of the original file with '}'

	The new code begins on line 10 of the proposed updates with 'func ExecQuery() {...' and ends on line 15 of the proposed updates with '}'
	---

	---
	Type: prepend

	The section that should have code prepended to it begins on line 10 of the original file with 'func GetResults() {...' and ends on line 15 of the original file with '}'

	The new code begins on line 10 of the proposed updates with 'func ExecQuery() {...' and ends on line 15 of the proposed updates with '}'
	---

	If the change type is 'replace' and only a single line needs to be replaced, you can reference a single line from the original file and the proposed updates rather than a range of lines like this:
	---
	Type: replace

	The section to be replaced is on line 10 of the original file with 'someVar = 5'

	The new code is on line 12 of the proposed updates with 'someVar = 10'
	---

	To append to the end of the file, you can reference a single line, the last line of the original file, for the new code to be appended to like this:
	---
	Type: append

	Line 15 of the original file with '}'

	The new code begins on line 10 of the proposed updates with 'func ExecQuery() {...' and ends on line 15 of the proposed updates with '}'
	--

	To prepend to the beginning of the file, you can reference a single line, the first line of the original file, for the new code to be prepended to like this:

	---
	Type: prepend

	Line 1 of the original file with 'package main'

	The new code begins on line 10 of the proposed updates with 'func ExecQuery() {...' and ends on line 15 of the proposed updates with '}'
	---

	The 'changeType' property is an integer that represents the type of change. The value of 'changeType' corresponds to the type of change as follows:
	- 1: replace
	- 2: append
	- 3: prepend
	You must include the 'changeType' property in each change object and it must be one of the above values (1, 2, or 3).

	The 'old' property is an object with two properties: 'startLine' and 'endLine'.

	If the changeType is 'replace': 'startLine' is the line number where the section to be replaced begins in the original file. 'endLine' is the line number where the section to be replaced ends in the original file. For a single line replacement, 'startLine' and 'endLine' will be the same.

	If the changeType is 'append', 'startLine' and 'endLine' are both the same: the line number where the section to have code appended to it ends in the original file.

	If the changeType is 'prepend', 'startLine' and 'endLine' are both the same: the line number where the section to have code prepended to it begins in the original file.

	The 'new' property is also an object with two properties: 'startLine' and 'endLine'.

	'startLine' is the line number where the new section begins in the code block included with the proposed updates. 'endLine' is the line number where the new section ends in the code block included with the proposed updates. For a single line replacement, 'startLine' and 'endLine' will be the same.

	Example function call with all keys:
	---
	listChanges([{
		shortSummary: "Insert function ExecQuery after GetResults function.",
		changeSections: "Type: replace\nThe section to be replaced is on line 5 of the original file with '\\n'\n\nThe new code starts on line 3 of the proposed updates with 'func ExecQuery()...' and ends on line 10 of the proposed updates with '}'",
		changeType: 1,
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

	Very important notes:

	- Apply changes intelligently in order to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

	- If the code block with the proposed updates includes references to the original code file, like this comment "// rest of the function here..." or "# rest of the code here" or "/* rest of the functionality goes here */", **do not include these references** when constructing the changes. In these cases, only replace the code that is actually changing and leave out the references to the original code file. The code should be ready for use and should not contain any artifacts of the planning process.

	[END YOUR INSTRUCTIONS]
`
var ListReplacementsFn = openai.FunctionDefinition{
	Name: "listChanges",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"changes": {
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
						"changeType": {
							Type: jsonschema.Integer,
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
					Required: []string{"shortSummary", "changeSections", "changeType", "old", "new"},
				},
			},
		},
		Required: []string{"changes"},
	},
}
