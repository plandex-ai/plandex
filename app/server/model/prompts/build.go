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
  
  [YOUR INSTRUCTIONS]
  Call the 'listChanges' function with a valid JSON object that includes the 'changes' keys.

  'changes': An array of changes. Each change is an object with properties: 'summary', 'section', 'old', and 'new'.
  
  The 'summary' property is a brief summary of the change. 
  
  'summary' examples: 
    - 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
    - 'Change the value of someVar to 10.'
    - 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
    - 'Add function ExecQuery to execute a query.'

  The 'section' property is a description of what section of code from the original file will be replaced.
    
  Refer to sections of code using line numbers, how the section begins, and how the section ends. It must be extremely clear which line(s) of code from the original file will be replaced.

  'section' examples:
  ---
  Begins line 10 of the original file with 'for i := 0; i < 10; i++ {...'  
  Ends on line 15 of the original file with '}'
  ---

  The 'old' property is an object with two properties: 'startLine' and 'endLine'.

  'startLine' is the line number where the section to be replaced begins in the original file. 'endLine' is the line number where the section to be replaced ends in the original file. For a single line replacement, 'startLine' and 'endLine' will be the same.

  The 'new' property is a string that represents the new code that will replace the old code. The new code must be valid and consistent with the intention of the plan. If the the proposed update is to remove code, the 'new' property should be an empty string. 
  
  If the proposed update includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or any other reference to the original code, you *MUST* ensure that the comment making the reference is *NOT* included in the 'new' property. If the reference comment is included in the 'new' property the resulting code obviously won't work, so NEVER include the reference comment in the 'new' property. Instead include the actual code that the reference is pointing to so that the change results in the exact code that is intended.

  Example function call with all keys:
  ---
  listChanges([{
    summary: "Insert function ExecQuery after GetResults function.",
    section' "Begins line 10 of the original file with 'for i := 0; i < 10; i++ {...'\nEnds on line 15 of the original file with '}'",
    old: {
      startLine: 5,
      endLine: 5,
    },
    new: "execQuery()\nreturn",
  }])
  ---

  Apply changes intelligently in order to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.
 
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
						"summary": {
							Type: jsonschema.String,
						},
						"section": {
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
							Type: jsonschema.String,
						},
					},
					Required: []string{"summary", "section", "old", "new"},
				},
			},
		},
		Required: []string{"changes"},
	},
}
