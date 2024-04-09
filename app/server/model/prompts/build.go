package prompts

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildSysPrompt(filePath, currentState, desc, changes string) string {
	currentStateWithLineNums := ""
	var lastLineNum int
	for i, line := range strings.Split(currentState, "\n") {
		currentStateWithLineNums += fmt.Sprintf("%d: %s\n", i+1, line)
		lastLineNum = i + 1
	}

	return getListChangesPrompt(lastLineNum) + "\n\n" + getBuildCurrentStatePrompt(filePath, currentStateWithLineNums) + "\n\n" + getBuildPrompt(desc, changes)
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

func getBuildCurrentStatePrompt(filePath, withLineNums string) string {
	if withLineNums == "" {
		return ""
	}

	return fmt.Sprintf("**The current file is %s. Original state of the file:**\n```\n%s\n```", filePath, withLineNums) + "\n\n"
}

func getListChangesPrompt(lastLineNum int) string {
	return fmt.Sprintf(`	
  You are an AI that analyzes a code file and an AI-generated plan to update the code file and produces a list of changes.
  
  [YOUR INSTRUCTIONS]
  Call the 'listChanges' function with a valid JSON object that includes the 'changes' keys.

  'changes': An array of NON-OVERLAPPING changes. Each change is an object with properties: 'summary', 'section', 'old', and 'new'.
  
  The 'summary' property is a brief summary of the change. At the end of the summary, consider if this change will overlap with any ensuing changes. If it will, include those changes in *this* change instead. Continue the summary and includes those ensuing changes that would otherwise overlap. Changes that remove code are especially likely to overlap with ensuing changes. 
  
  'summary' examples: 
    - 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
    - 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
    - 'Add function ExecQuery to execute a query.'
		
	'summary' that is larger to avoid overlap:
		- 'Insert function ExecQuery after GetResults function in loop body. Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar. Add function ExecQuery to execute a query.'

  The 'section' property is a description of what section of code from the original file will be replaced.
    
  Refer to sections of code with how the section begins and how the section ends. It must be extremely clear which line(s) of code from the original file will be replaced. Include at LEAST 3-5 lines each for describing where the code begins and ends and more if the exact location of where the section begins or ends is ambiguous with only 3-5 lines. Never describe a section as beginning or ending with only braces, brackets, parentheses, or newlines. Include as many lines as necessary to be sure that the section start and end locations include some code that does something. Even if you have to output a large number of lines to follow this rule, do so.
	
	Sections of code begin and end with *entire* lines. Never begin or end a section in the middle of a line.

	Abbreviate long lines of code with '...'. Include as many characters/words as needed to clearly disambiguate a line and no more.

  'section' examples:
  ---
  Begins: 'for i := 0; i < 10; i++ {...'  
  Ends: '    }\n  }\n}'
  ---

  The 'old' property is an object with 5 properties: 'maybeStartLine', 'maybeEndLine', 'err', 'startLine' and 'endLine'.

		'maybeStartLine' is the line number where the section to be replaced begins in the original file. 'maybeEndLine' is the line number where the section to be replaced ends in the original file. For a single line replacement, 'maybeStartLine' and 'maybeEndLine' will be the same.

		'maybeEndLine' MUST ABSOLUTELY ALWAYS be greater than or equal to 'maybeStartLine'.
		
		Line numbers in 'maybeStartLine' and 'maybeEndLine' are 1-indexed. 1 is the minimum line number. The maximum line number is %d, the number of lines in the original file. 'maybeStartLine' and 'maybeEndLine' MUST be valid line numbers in the original file, greater than or equal to 1 and less than or equal to %d. 
		
		You MUST refer to 1-indexed line numbers exactly as they are labeled in the original file. If the 'maybeStartLine' or 'maybeEndLine' is the first line of the file, 'maybeStartLine' or 'maybeEndLine' will be 1. 
		
		If the 'maybeStartLine' or 'maybeEndLine' is the last line of the file, 'maybeStartLine' or 'maybeEndLine' will be %d.
		
		Both 'maybeStartLine' and 'maybeEndLine' must NEVER be 0, -1, or any other number less than 1. They must NEVER be a number higher than %d.

		'err' is a string. If 'maybeEndLine' is greater than or equal to 'maybeStartLine', 'err' must be an empty string. Otherwise, 'err' must be a string that describes how the section from the original file can be better described in order to make the line numbers unambiguous and correct.

		'startLine' is the line number where the section to be replaced begins in the original file. *'startLine' MUST be an integer greater than 0.* If 'err' is not set or is an empty string, 'startLine' must be equal to 'maybeStartLine'. If 'err' is set and is *not* an empty string, 'startLine' should be the *corrected* line number where the section to be replaced begins in the original file.
		
		'endLine' is the line number where the section to be replaced ends in the original file. *'endLine' MUST be an integer greater than 0.* If 'err' is not set or is an empty string, 'endLine' must be equal to 'maybeEndLine'. If 'err' is set and is *not* an empty string, 'endLine' should be the *corrected* line number where the section to be replaced ends in the original file.

  The 'new' property is a string that represents the new code that will replace the old code. The new code must be valid and consistent with the intention of the plan. If the proposed update is to remove code, the 'new' property should be an empty string.
  
  If the proposed update includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or **any other reference to the original code,** you *MUST* ensure that the comment making the reference is *NOT* included in the 'new' property. Instead, include the **exact code** from the original file that the comment is referencing. Do not be overly strict in identifying references. If there is a comment that seems like it could plausibly be a reference and there is code in the original file that could plausibly be the code being referenced, then treat that as a reference and handle it accordingly by including the code from the original file in the 'new' property instead of the comment. YOU MUST NOT MISS ANY REFERENCES.

  Example function call with all keys:
  ---
  listChanges([{
    summary: "Insert function ExecQuery after GetResults function in loop body.",
    section' "Begins: 'for i := 0; i < 10; i++ {...'\nEnds: '    }\n  }\n}'",
    old: {
			maybeStartLine: 5,
			maybeEndLine: 10,
			err: "",
      startLine: 5,
      endLine: 10,
    },
    new: "      execQuery()\n    }\n  }\n}",
  }])
  ---

	Example function calls with errors:
	---
	listChanges([{
		summary: "Insert function ExecQuery after GetResults function in loop body.",
		section' "Begins: 'for i := 0; i < 10; i++ {...'\nEnds: '    }\n  }\n}'",
		old: {
			maybeStartLine: 5,
			maybeEndLine: 4,
			err: "maybeStartLine is greater than maybeEndLine. The 'begins' section is ambiguous. Here's a better 'begins' section: '// Loop up to 10\nfor i := 0; i < 10; i++ {\n	// Loop body'",
			startLine: 5,
			endLine: 10,
		},
		new: "      execQuery()\n    }\n  }\n}",
	}])

	You ABSOLUTELY MUST NOT generate overlapping changes. The start line of each change MUST be **greater than** the end line of the previous change. Group smaller changes together into larger changes where necessary to avoid overlap. Only generate multiple changes when you are ABSOLUTELY CERTAIN that they do not overlap--otherwise group them together into a single change.

  Apply changes intelligently in order to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

	You ABSOLUTELY MUST NOT ovewrite or delete code from the original file unless the plan *clearly intends* for the code to be overwritten or removed. Do NOT replace a full section of code with only new code unless that is the clear intention of the plan. Instead, merge the original code and the proposed changes together intelligently according to the intention of the plan. 

	Pay *EXTREMELY close attention* to opening and closing brackets, parentheses, and braces. Never leave them unbalanced when the changes are applied.

	The 'listChanges' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChanges' function call parameters JSON object *must be properly escaped* with a backslash.
 
  [END YOUR INSTRUCTIONS]
`, lastLineNum, lastLineNum, lastLineNum, lastLineNum)
}

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
								"maybeStartLine": {
									Type: jsonschema.Integer,
								},
								"maybeEndLine": {
									Type: jsonschema.Integer,
								},
								"err": {
									Type: jsonschema.String,
								},
								"startLine": {
									Type: jsonschema.Integer,
								},
								"endLine": {
									Type: jsonschema.Integer,
								},
							},
							Required: []string{"maybeStartLine", "maybeEndLine", "startLine", "endLine"},
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
