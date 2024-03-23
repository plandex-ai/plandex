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
    - 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
    - 'Add function ExecQuery to execute a query.'

  The 'section' property is a description of what section of code from the original file will be replaced.
    
  Refer to sections of code with how the section begins and how the section ends. It must be extremely clear which line(s) of code from the original file will be replaced. Include at LEAST 3-5 lines each for describing where the code begins and ends and more if the exact location of where the section begins or ends is ambiguous with only 3-5 lines. Never describe a section as beginning or ending with only braces, brackets, parentheses, or newlines. Include as many lines as necessary to be sure that the section start and end locations include some code that does something. Even if you have to output a large number of lines to follow this rule, do so.
	
	Sections of code begin and end with *entire* lines. Never begin or end a section in the middle of a line.

	Abbreviate long lines of code with '...'. Include as many characters/words as needed to clearly disambiguate a line and no more.

  'section' examples:
  ---
  Begins: 'for i := 0; i < 10; i++ {...'  
  Ends: '    }\n  }\n}'
  ---

  The 'old' property is an object with two properties: 'startLine' and 'endLine'.

  'startLine' is the line number where the section to be replaced begins in the original file. 'endLine' is the line number where the section to be replaced ends in the original file. For a single line replacement, 'startLine' and 'endLine' will be the same. 
	
	Line numbers in 'startLine' and 'endLine' are 1-indexed. 1 is the minimum line number. The maximum line number is the number of lines in the original file. 'startLine' and 'endLine' must be valid line numbers in the original file, greater than or equal to 1 and less than or equal to the number of lines in the original file. You MUST refer to 1-indexed line numbers exactly as they are labeled in the original file. If the 'startLine' or 'endLine' is the first line of the file, 'startLine' or 'endLine' will be 1. If the 'startLine' or 'endLine' is the last line of the file, 'startLine' or 'endLine' will be the number of lines in the file. 'startLine' must NEVER be 0, -1, or any other number less than 1. 

  The 'new' property is a string that represents the new code that will replace the old code. The new code must be valid and consistent with the intention of the plan. If the the proposed update is to remove code, the 'new' property should be an empty string. 
  
  If the proposed update includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or **any other reference to the original code,** you *MUST* ensure that the comment making the reference is *NOT* included in the 'new' property. Instead, include the **exact code** from the original file that the comment is referencing. Do not be overly strict in identifying references. If there is a comment that seems like it could plausibly be a reference and there is code in the original file that could plausibly be the code being referenced, then treat that as a reference and handle it accordingly by including the code from the original file in the 'new' property instead of the comment. YOU MUST NOT MISS ANY REFERENCES.

  Example function call with all keys:
  ---
  listChanges([{
    summary: "Insert function ExecQuery after GetResults function in loop body.",
    section' "Begins: 'for i := 0; i < 10; i++ {...'\nEnds with '    }\n  }\n}'",
    old: {
      startLine: 5,
      endLine: 10,
    },
    new: "      execQuery()\n    }\n  }\n}",
  }])
  ---

  Apply changes intelligently in order to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

	You ABSOLUTELY MUST NOT ovewrite or delete code from the original file unless the plan *clearly intends* for the code to be overwritten or removed. Do NOT replace a full section of code with only new code unless that is the clear intention of the plan. Instead, merge the original code and the proposed changes together intelligently according to the intention of the plan. 

	Pay *EXTREMELY close attention* to opening and closing brackets, parentheses, and braces. Never leave them unbalanced when the changes are applied.

	The 'listChanges' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChanges' function call parameters JSON object *must be properly escaped* with a backslash.
 
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
