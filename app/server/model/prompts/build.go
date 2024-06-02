package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildLineNumbersSysPrompt(filePath, preBuildState, changes string) string {
	// hash := sha256.Sum256([]byte(currentState))
	// sha := hex.EncodeToString(hash[:])

	// log.Println("GetBuildLineNumbersSysPrompt currentState sha:", sha)

	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)

	return getListChangesLineNumsPrompt() + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n" + getBuildPromptWithLineNums(changes)
}

func GetBuildFixesLineNumbersSysPrompt(original, changes, updated, reasoning string) string {
	// hash := sha256.Sum256([]byte(updated))
	// sha := hex.EncodeToString(hash[:])
	// log.Println("GetBuildFixesLineNumbersSysPrompt updated sha:", sha)

	updatedWithLineNums := shared.AddLineNums(updated)

	return getFixChangesLineNumsPrompt() + "\n\n" + getBuildPromptForFixesWithLineNums(original, changes, updatedWithLineNums, reasoning)
}

func getBuildPromptWithLineNums(changes string) string {
	s := ""

	s += "Proposed updates:\n```\n" + changes + "\n```"

	s += "\n\n" + "Now call the 'listChangesWithLineNums' function with a valid JSON array of changes according to your instructions. You must always call 'listChangesWithLineNums' with one or more valid changes. Don't call any other function."

	return s
}

func getBuildPromptForFixesWithLineNums(original, changes, updated, reasoning string) string {
	s := ""

	if original != "" {
		s += "**Original file:**\n\n```\n" + original + "\n```"
	}

	s += "**Proposed updates:**\n\n" + changes + "\n\n--\n\n"

	s += fmt.Sprintf("**The incorrectly updated file is:**\n\n```\n%s\n```\n\n**The problems with the file are:**\n\n%s\n\n--", updated, reasoning)

	s += "\n\n" + "Now call the 'listChangesWithLineNums' function with a valid JSON array of changes according to your instructions. You must always call 'listChangesWithLineNums' with one or more valid changes. Don't call any other function."

	return s
}

func getPreBuildStatePrompt(filePath, preBuildState string) string {
	if preBuildState == "" {
		return ""
	}

	return fmt.Sprintf("**The current file is %s. Original state of the file:**\n```\n%s\n```", filePath, preBuildState) + "\n\n"
}

const replacementIntro = `
You are an AI that analyzes a code file and an AI-generated plan to update the code file and produces a list of changes.
`

const changesKeyPrompt = `
'changes': An array of NON-OVERLAPPING changes. Each change is an object with properties: 'summary', 'hasChange', 'old', 'startLineIncludedReasoning', 'startLineIncluded', 'endLineIncludedReasoning', 'endLineIncluded', and 'new'.

Note: all line numbers that are used below are prefixed with 'pdx-', like this 'pdx-5: for i := 0; i < 10; i++ {'. This is to help you identify the line numbers in the file. You *must* include the 'pdx-' prefix in the line numbers in the 'old' property.
`

const summaryChangePrompt = `
The 'summary' property is a brief summary of the change. At the end of the summary, consider if this change will overlap with any ensuing changes. If it will, include those changes in *this* change instead. Continue the summary and includes those ensuing changes that would otherwise overlap. Changes that remove code are especially likely to overlap with ensuing changes. 

'summary' examples: 
	- 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
	- 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
	- 'Add function ExecQuery to execute a query.'
	
'summary' that is larger to avoid overlap:
	- 'Insert function ExecQuery after GetResults function in loop body. Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar. Add function ExecQuery to execute a query.'

The 'hasChange' property is a boolean that indicates whether there is anything to change. If there is nothing to change, set 'hasChange' to false. If there is something to change, set 'hasChange' to true.
`

const lineNumsOldPrompt = `
The 'old' property is an object with 3 properties: 'entireFile', 'startLineString' and 'endLineString'.

	'entireFile' is a boolean that indicates whether the **entire file** is being replaced. If 'entireFile' is true, 'startLineString' and 'endLineString' must be empty strings. If 'entireFile' is false, 'startLineString' and 'endLineString' must be valid strings that exactly match lines from the original file. If 'entireFile' is false, 'startLineString' and 'endLineString' MUST NEVER be empty strings.

	'startLineString' is the **entire, exact line** where the section to be replaced begins in the original file, including the line number. Unless it's the first change, 'startLineString' ABSOLUTELY MUST begin with a line number that is HIGHER than both the 'endLineString' of the previous change and the 'startLineString' of the previous change. **The line number and line MUST EXACTLY MATCH a line from the original file.**
	
	If the previous change's 'endLineString' starts with 'pdx-75: ', then the current change's 'startLineString' MUST start with 'pdx-76: ' or higher. It MUST NOT be 'pdx-75: ' or lower. If the previous change's 'startLineString' starts with 'pdx-88: ' and the previous change's 'endLineString' is an empty string, then the current change's 'startLineString' MUST start with 'pdx-89: ' or higher. If the previous change's 'startLineString' starts with 'pdx-100: ' and the previous change's 'endLineString' starts with 'pdx-105: ', then the current change's 'startLineString' MUST start with 'pdx-106: ' or higher.
	
	'endLineString' is the **entire, exact line** where the section to be replaced ends in the original file. Pay careful attention to spaces and indentation. 'startLineString' and 'endLineString' must be *entire lines* and *not partial lines*. Even if a line is very long, you must include the entire line, including the line number and all text on the line. **The line number and line MUST EXACTLY MATCH a line from the original file.**
	
	**For a single line replacement, 'endLineString' MUST be an empty string.**

	'endLineString' MUST ALWAYS come *after* 'startLineString' in the original file. It must start with a line number that is HIGHER than the 'startLineString' line number. If 'startLineString' starts with 'pdx-22: ', then 'endLineString' MUST either be an empty string (for a single line replacement) or start with 'pdx-23: ' or higher (for a multi-line replacement).	

	If 'hasChange' is false, both 'startLineString' and 'endLineString' must be empty strings. If 'hasChange' is true, 'startLineString' and 'endLineString' must be valid strings that exactly match lines from the original file. If 'hasChange' is true, 'startLineString' and 'endLineString' MUST NEVER be empty strings.

	If you are replacing the entire file, 'startLineString' MUST be the first line of the original file and 'endLineString' MUST be the last line of the original file.
`

const changeLineInclusionAndNewPrompt = `
The 'startLineIncludedReasoning' property is a string that very briefly explains whether 'startLineString' should be included in the 'new' property. For example, if the 'startLineString' is the closing bracket of a function and you are adding another function after it, you *MUST* include the 'startLineString' in the 'new' property, or the previous function will lose its closing bracket when the change is applied. Similarly, if the 'startLineString' is a function definition and you are updating the body of the function, you *MUST* also include 'startLineString' so that they function definition is not removed. The only time 'startLineString' should not be included in 'new' is if it is a line that should be removed or replaced. Generalize the above to all types of code blocks, changes, and syntax to ensure the 'new' property will not remove or overwrite code that should not be removed or overwritten. That also includes newlines, line breaks, and indentation.

'startLineIncluded' is a boolean that indicates whether 'startLineString' should be included in the 'new' property. If 'startLineIncluded' is true, 'startLineString' MUST be included in the 'new' property. If 'startLineIncluded' is false, 'startLineString' MUST not be included in the 'new' property.

The 'endLineIncludedReasoning' property is a string that very briefly explains whether 'endLineString' should be included in the 'new' property. For example, if the 'endLineString' is the opening bracket of a function and you are adding another function before it, you *MUST* include the 'endLineString' in the 'new' property, or the subsequent function will lose its opening bracket when the change is applied. Similarly, if the 'endLineString' is the closing bracket of a function and you are updating the body of the function, you *MUST* also include 'endLineString' so that the closing bracket not removed. The only time 'endLineString' should not be included in 'new' is if it is a line that should be removed or replaced. Generalize the above to all types of code blocks, changes, and syntax to ensure the 'new' property will not remove or overwrite code that should not be removed or overwritten. That also includes newlines, line breaks, and indentation.

'endLineIncluded' is a boolean that indicates whether 'endLineString' should be included in the 'new' property. If 'endLineIncluded' is true, 'endLineString' MUST be included in the 'new' property. If 'endLineIncluded' is false, 'endLineString' MUST not be included in the 'new' property.

The 'new' property is a string that represents the new code that will replace the old code. The new code must be valid and consistent with the intention of the plan. If the proposed update is to remove code, the 'new' property should be an empty string. Be precise about newlines, line breaks, and indentation. 'new' must include only full lines of code and *no partial lines*. Do NOT include line numbers in the 'new' property.

If the proposed update includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or **any other reference to the original code,** you *MUST* ensure that the comment making the reference is *NOT* included in the 'new' property. Instead, include the **exact code** from the original file that the comment is referencing. Do not be overly strict in identifying references. If there is a comment that seems like it could plausibly be a reference and there is code in the original file that could plausibly be the code being referenced, then treat that as a reference and handle it accordingly by including the code from the original file in the 'new' property instead of the comment. YOU MUST NOT MISS ANY REFERENCES.

If the 'startLineIncluded' property is true, the 'startLineString' MUST be the first line of 'new'. If the 'startLineIncluded' property is false, the 'startLineString' MUST NOT be included in 'new'. If the 'endLineIncluded' property is true, the 'endLineString' MUST be the last line of 'new'. If the 'endLineIncluded' property is false, the 'endLineString' MUST NOT be included in 'new'.

If the 'hasChange' property is false, the 'new' property must be an empty string. If the 'hasChange' property is true, the 'new' property must be a valid string.

If *any* change has the 'entireFile' key in the 'old' property set to true, the corresponding 'new' key MUST be the entire updated file, and there MUST only be a single change in the 'changes' array.
`

const lineNumsRulesPrompt = `
You ABSOLUTELY MUST NOT generate overlapping changes. Group smaller changes together into larger changes where necessary to avoid overlap. Only generate multiple changes when you are ABSOLUTELY CERTAIN that they do not overlap--otherwise group them together into a single change. If changes are close to each other (within several lines), group them together into a single change. You MUST group changes together and make fewer, larger changes rather than many small changes, unless the changes are completely independent of each other and not close to each other in the file. You MUST NEVER generate changes that are adjacent or close to adjacent. Adjacent or closely adjacent changes MUST ALWAYS be grouped into a single larger change.

Furthermore, unless doing so would require a very large change because some changes are far apart in the file, it's ideal to call the 'listChangesWithLineNums' with just a SINGLE change.

Changes must be ordered in the array according to the order they appear in the file. The 'startLineString' of each 'old' property must come after the 'endLineString' of the previous 'old' property. Changes MUST NOT overlap. If a change is dependent on another change or intersects with it, group those changes together into a single change.

You MUST NOT repeat changes to the same block of lines multiple teams. You MUST NOT duplicate changes. It is extremely important that a given change is only applied *once*.
`

const changeRulesPrompt = `
Apply changes intelligently **in order** to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

Changes MUST be ordered based on their position in the original file. ALWAYS go from top to bottom IN ORDER when generating replacements. DO NOT EVER GENERATE AN OVERLAPPING CHANGE. If a change would fall within OR overlap a prior change in the list, SKIP that change and move on to the next one.

You ABSOLUTELY MUST NOT overwrite or delete code from the original file unless the plan *clearly intends* for the code to be overwritten or removed. Do NOT replace a full section of code with only new code unless that is the clear intention of the plan. Instead, merge the original code and the proposed updates together intelligently according to the intention of the plan. 

Pay *EXTREMELY close attention* to opening and closing brackets, parentheses, and braces. Never leave them unbalanced when the changes are applied. Also pay *EXTREMELY close attention* to newlines and indentation. Make sure that the indentation of the new code is consistent with the indentation of the original code, and syntactically correct.
`

const lineNumsJsonPrompt = `
The 'listChangesWithLineNums' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChangesWithLineNums' function call parameters JSON object *must be properly escaped* with a backslash. Pay careful attention to newlines, tabs, and other special characters. The JSON object must be properly formatted and must include all required keys. **You generate perfect JSON -every- time**, no matter how many quotes or special characters are in the input. You must always call 'listChangesWithLineNums' with a valid JSON object. Don't call any other function. 
`

const commentsPrompt = `
The 'comments' key is an array of objects with two properties: 'txt' and 'reference'. 'txt' is the exact text of a code comment. 'reference' is a boolean that indicates whether the comment is a placeholder of or reference to the original code, like "// rest of the function..." or "# existing init code...", or "// rest of the main function" or "// rest of your function..." or "// Existing methods..." or "// Remaining methods" or "// Existing code..." or "// ... existing setup code ..."" or other comments which reference code from the original file. References DO NOT need to exactly match any of the previous examples. Use your judgement to determine whether each comment is a reference. If 'reference' is true, the comment is a placeholder or reference to the original code. If 'reference' is false, the comment is not a placeholder or reference to the original code.

In 'comments', you must list EVERY comment included in the proposed updates. Only list *code comments* that are valid comments for the programming language being used. Do not list logging statements or any other non-comment text that is not a valid code comment. If there are no code comments in the proposed updates, 'comments' must be an empty array.

If there are multiple identical comments in the proposed updates, you MUST list them *all* in the 'comments' array--list each identical comment as a separate object in the array.
`

const problemsStrategyPrompt = `
In the 'problems' key, you MUST explain how you will strategically generate changes in order to avoid any problems in the updated file. You should explain which changes you will make and how you will *avoid* making any overlapping or invalid changes. Consider whether any changes are close together or whether any change is potentially contained by another. If so, group those changes together into a single change.

You must consider whether you will apply partial changes or replace the entire file. If the original file is long, you MUST NOT replace the entire file with a single change. Instead, you should apply changes to specific sections of the file. If the original file is short and the changes are complex, you may consider replacing the entire file with a single change.

You must consider how you will avoid *incorrectly removing or overwriting code* from the original file. Explain whether any code from the original file needs to be merged with the proposed updates in order to avoid removing or overwriting code that should not be removed. 

You must consider how you will avoid including any references in the updated file if any are present in the proposed updates. 

You must consider how you will *avoid incorrect duplication* in making your changes. For example if a 'main' function is present in the original file and the proposed updates include update code for the 'main' function, you must ensure the changes are applied within the existing 'main' function rather than incorrectly adding a duplicate 'main' function.

If the proposed updates include large sections that are identical to the original file, consider whether the changes can be made more minimal in order to only replace sections of code that are *changing*. If you are making the changes more minimal and specific, explain how you will do this without generating any overlapping changes or introducing any new problems.
`

func getListChangesLineNumsPrompt() string {

	return replacementIntro + `

	[YOUR INSTRUCTIONS]

	Call the 'listChangesWithLineNums' function with a valid JSON object that includes the 'comments', 'problems', and 'changes' keys.
	
	` + lineNumsRulesPrompt + `

	` + commentsPrompt + `
	
	` + problemsStrategyPrompt + `

	` + changesKeyPrompt + `

	` + summaryChangePrompt + `

	` + lineNumsOldPrompt + `
  
  ` + changeLineInclusionAndNewPrompt + `

  Example change object:
  ---
  {
    summary: "Fix syntax error in loop body.",
   	old: {
      startLineString: "pdx-5: for i := 0; i < 10; i++ { ",
      endLineString: "pdx-7: }",
    },
    new: "for i := 0; i < 10; i++ {\n  execQuery()\n  }\n  }\n}",
  }
  ---

	` + `

	` + changeRulesPrompt + `

	` + lineNumsJsonPrompt + `
 
  [END YOUR INSTRUCTIONS]
`
}
func getFixChangesLineNumsPrompt() string {

	return `
	You are an AI that analyzes an original file (if present), an incorrectly updated file, the changes that were proposed, and a description of the problems with the file, and then produces a list of changes to apply to the *incorrectly updated file* that will fix *ALL* the problems.

	Problems you MUST fix include:
	- Syntax errors, including unbalanced brackets, parentheses, braces, quotes, indentation, and other code structure errors
	- Missing or incorrectly scoped declarations
	- Any other errors that make the code invalid and would prevent it from being run as-is for the programming language being used
	- Incorrectly applied changes
	- Incorrectly removed code
	- Incorrectly overwritten code
	- Incorrectly duplicated code
	- Incorrectly applied comments that reference the original code

	If the updated file includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods..." **any other reference to the original code, the file is incorrect. References like these must be handled by including the exact code from the original file that the comment is referencing.

	[YOUR INSTRUCTIONS]
	Call the 'listChangesWithLineNums' function with a valid JSON object that includes the 'comments','problems' and 'changes' keys.
	
	'comments': Since this is a fix, comments must be an empty array.

	'problems': A string that describes all problems present within the updated file. Explain the cause of each problem and how it should be fixed. Do not just restate that there is a syntax error on a specific line. Explain what the syntax error is and how to fix it. Be exhaustive and include *every* problem that is present in the file.

	Since you are fixing an incorrectly updated file, you *MUST* include the 'problems' key and you *MUST* describe *all* problems present in the file. If there are multiple problems, list each one individually. If there are multiple identical problems, list each one individually.

	You should also explain your strategy for generating changes in the 'problems' key according to these instructions:
	
	` + problemsStrategyPrompt + ` 
	
	` + changesKeyPrompt + `

	` + summaryChangePrompt + `

  ` + lineNumsOldPrompt + `
	
	` + changeLineInclusionAndNewPrompt + `

	You MUST ensure the line numbers for the 'old' property correctly remove *ALL* code that has problems and that the 'new' property correctly fixes *ALL* the problems present in the updated file. You MUST NOT miss any problems, fail to fix any problems, or introduce any new problems.

  Example change object:
  ---
  {
    summary: "Fix syntax error in loop body.",
    old: {
      startLineString: "pdx-5: for i := 0; i < 10; i++ { ",
      endLineString: "pdx-7: }",
    },
    new: "for i := 0; i < 10; i++ {\n  execQuery()\n  }\n  }\n}",
  }
  ---

	` + lineNumsRulesPrompt + `

	` + changeRulesPrompt + `

	` + lineNumsJsonPrompt + `
 
  [END YOUR INSTRUCTIONS]
`
}

var ListReplacementsFn = openai.FunctionDefinition{
	Name: "listChangesWithLineNums",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"comments": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"txt": {
							Type: jsonschema.String,
						},
						"reference": {
							Type: jsonschema.Boolean,
						},
					},
					Required: []string{"txt", "reference"},
				},
			},
			"problems": {
				Type: jsonschema.String,
			},
			"changes": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"summary": {
							Type: jsonschema.String,
						},
						"hasChange": {
							Type: jsonschema.Boolean,
						},
						"old": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"entireFile": {
									Type: jsonschema.Boolean,
								},
								"startLineString": {
									Type: jsonschema.String,
								},
								"endLineString": {
									Type: jsonschema.String,
								},
							},
							Required: []string{"startLineString", "endLineString"},
						},
						"startLineIncludedReasoning": {
							Type: jsonschema.String,
						},
						"startLineIncluded": {
							Type: jsonschema.Boolean,
						},
						"endLineIncludedReasoning": {
							Type: jsonschema.String,
						},
						"endLineIncluded": {
							Type: jsonschema.Boolean,
						},
						"new": {
							Type: jsonschema.String,
						},
					},
					Required: []string{
						"summary",
						"hasChange",
						"old",
						"startLineIncludedReasoning",
						"startLineIncluded",
						"endLineIncludedReasoning",
						"endLineIncluded",
						"new",
					},
				},
			},
		},
		Required: []string{"comments", "problems", "changes"},
	},
}

func GetVerifyPrompt(preBuildState, updated, changes, diff string) string {
	s := `
Based on an original file (if one exists), an AI-generated plan, an updated file, and a diff between the original and updated file, determine whether the updated file's syntax is correct and whether the proposed updates were applied correctly to the updated file.

You must consider whether any of the following problems are present in the updated file:
- Syntax errors, including unbalanced brackets, parentheses, braces, quotes, indentation, and other code structure errors
- Missing or incorrectly scoped declarations
- Any other errors that make the code invalid and would prevent it from being run as-is for the programming language being used
- Code from the original file was incorrectly removed or overwritten.
- Code was incorrectly duplicated. For example, if a file should have a single main function, but instead of updating the existing main function, the updated file includes multiple main functions, then the file is incorrect. The same applies to any other functions or elements that should not be duplicated.
- Incorrectly included comments that reference the original code.. If the updated file includes comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods...", "// Existing code..." **any other reference to the original code**, the file is incorrect. References like these must be handled by including the exact code from the original file that the comment is referencing.

If there is no original file, it means that a new file was created from scratch based on the AI-generated plan. In this case, the syntax in the new file must be valid and consistent with the intention of the plan. You must ensure there are no syntax errors or other clear mistakes in the new file.

Call the 'verifyOutput' function with a valid JSON object that include the following keys:

'syntaxErrorsReasoning': A string that succinctly explains whether there are any syntax or scoping errors in the updated file. Explain all syntax errors, scoping errors, or other code structure errors that are present in the updated file. 

'hasSyntaxErrors': A boolean that indicates whether there are any syntax errors in the updated file, based on the reasoning provided in 'syntaxErrorsReasoning'.

'removed': an array of objects with three properties: 'code', 'reasoning', and 'correct'. 
   - 'code' is a string. It shows the section of code that was removed or overwritten in the updated file. This can be abbreviated by detailing how the section starts and end and describing the purpose of the code. If the section is longer than a few lines, rather than reproducding a long section of code verbatim, provide a summary of the code that was removed or overwritten, as well as the exact code which starts and ends the section. The summary and start/end code should be details enough to disambiguate the section of code that was removed or overwritten from any other similar sections of code in the file.
   - 'reasoning' is a string that explains whether, based on the proposed changes, this section was deliberately removed consistent with the intention of the plan, or whether the plan did NOT specify that this section should be removed, and the code was therefore removed incorrectly. Also consider whether this removal breaks the syntax or functionality of the code in the updated file based on the programming language being used--if it does, explain this and state that the removal was incorrect. If the removal either wasn't intended by the proposed updates *or* breaks the syntax or functionality of the code, the removal is incorrect. If the removal was consistent with the intention of the plan *and* did not break the syntax or functionality of the code, the removal is correct.
   - 'correct' is a boolean that indicates whether the removal or overwriting was correct based on the 'reasoning' provided. If the removal was correct, set 'correct' to true. If the removal was incorrect, set 'correct' to false.

Based on supplied diffs, you must list EVERY code section that was removed or overwritten in the updated file in 'removed'. If there are no code sections that were removed or overwritten, 'removed' must be an empty array. If multiple code sections were removed or overwritten, list each one as a separate object in the 'removed' array. If multiple identical code sections were removed or overwritten, you MUST list them *all* in the 'removed' array--list each identical code section as a separate object in the array. Do NOT include removals that only modify whitespace. Do NOT include sections that were moved or refactored, only sections that were fully removed or overwritten.

'removedCodeErrorsReasoning': A string that succinctly explains whether any code was incorrectly removed or overwritten in the updated file based on the 'removed' array. If code was incorrectly removed or overwritten, succinctly explain why it was incorrect, and how the file can be corrected. If code was correctly removed or overwritten, consistent with the intention of the plan, state this.

'hasRemovedCodeErrors': A boolean that indicates whether any code was *incorrectly* removed or overwritten in the updated file, based on the reasoning provided in 'removedCodeErrorsReasoning'.

'duplicationErrorsReasoning': A string that succinctly explains whether any code was *incorrectly* duplicated in the updated file. First explain whether any code, functions, or other elements are duplicated in the updated file, then explain whether the duplication is deliberate and consistent with the plan, and whether the duplication is correct and valid in the programming language being used.

'hasDuplicationErrors': A boolean that indicates whether any code was *incorrectly* duplicated in the updated file, based on the reasoning provided in 'duplicationErrorsReasoning'. If code was *incorrectly* duplicated, set 'hasDuplicationErrors' to true. If code was *correctly* duplicated, consistent with the intention of the plan, set 'hasDuplicationErrors' to false.

'comments':  an array of objects with two properties: 'txt' and 'reference'. 'txt' is the exact text of a code comment. 'reference' is a boolean that indicates whether the comment is a placeholder of or reference to the original code, like "// rest of the function..." or "# existing init code...", or "// rest of the main function" or "// rest of your function..." or "// Existing methods..." or "// Remaining methods" or "// Existing code..." or "// ... existing setup code ..."" or other comments which reference code from the original file. References DO NOT need to exactly match any of the previous examples. Use your judgement to determine whether each comment is a reference. If 'reference' is true, the comment is a placeholder or reference to the original code. If 'reference' is false, the comment is not a placeholder or reference to the original code.

In 'comments', you must list EVERY comment included in the *updated file*. Only list *code comments* that are valid comments for the programming language being used. Do not list logging statements or any other non-comment text that is not a valid code comment. If there are no code comments in the *updated file*, 'comments' must be an empty array.

If there are multiple identical comments in the *updated file*, you MUST list them *all* in the 'comments' array--list each identical comment as a separate object in the array.

'referenceErrorsReasoning': A string that succinctly explains whether any comments in the updated file are placeholders/references that should have been replaced with code from the original file. These are comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods...", "// Existing code..." or other  comments which reference code from the original file. Only include comments that *are not* present in the original file and *are* present in the proposed updates. If there are no such comments, explain that there are no reference errors.

'hasReferenceErrors': A boolean that indicates whether any comments in the updated file are placeholders/references that should be replaced with code from the original file, based on the reasoning provided in 'referenceErrorsReasoning'.

In each of the reasoning keys above, be exhaustive and include *every* problem that is present in the file. But if there are no problems in a reasoning key, do NOT invent problems--explain according to your instructions for each key that there are no problems in that category.
`

	if preBuildState != "" {
		s += `
--

## **Original file:**

` + preBuildState + `

--
`
	}

	s += `
## **Proposed updates:**
[START PROPOSED UPDATES]

` + changes + `

[END PROPOSED UPDATES]
`

	if preBuildState != "" {
		s += `
	
## **Updated file:**

`
	} else {
		s += `
	## **New file:**

	`
	}

	s += updated + "\n\n"

	if diff != "" {
		s += "**Diff:**\n\n" + diff + "\n\n"
	}

	s += `

Now call the 'verifyOutput' function with a valid JSON object. Don't call any other function.

You absolutely MUST generate PERFECTLY VALID JSON. Pay extremely close attention to the JSON syntax and structure. Double quotes within JSON properties *MUST* be properly escaped with a backslash.`

	return s
}

var VerifyOutputFn = openai.FunctionDefinition{
	Name: "verifyOutput",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"syntaxErrorsReasoning": {
				Type: jsonschema.String,
			},
			"hasSyntaxErrors": {
				Type: jsonschema.Boolean,
			},
			"removed": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"code": {
							Type: jsonschema.String,
						},
						"reasoning": {
							Type: jsonschema.String,
						},
						"correct": {
							Type: jsonschema.Boolean,
						},
					},
					Required: []string{"code", "reasoning", "correct"},
				},
			},
			"removedCodeErrorsReasoning": {
				Type: jsonschema.String,
			},
			"hasRemovedCodeErrors": {
				Type: jsonschema.Boolean,
			},
			"duplicationErrorsReasoning": {
				Type: jsonschema.String,
			},
			"hasDuplicationErrors": {
				Type: jsonschema.Boolean,
			},
			"comments": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"txt": {
							Type: jsonschema.String,
						},
						"reference": {
							Type: jsonschema.Boolean,
						},
					},
					Required: []string{"txt", "reference"},
				},
			},
			"referenceErrorsReasoning": {
				Type: jsonschema.String,
			},
			"hasReferenceErrors": {
				Type: jsonschema.Boolean,
			},
		},
		Required: []string{"syntaxErrorsReasoning", "hasSyntaxErrors", "removed",
			"removedCodeErrorsReasoning", "hasRemovedCodeErrors", "duplicationErrorsReasoning", "hasDuplicationErrors", "comments", "referenceErrorsReasoning", "hasReferenceErrors"},
	},
}
