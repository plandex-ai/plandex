package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetBuildLineNumbersSysPrompt(filePath, preBuildState, desc, changes string) string {
	// hash := sha256.Sum256([]byte(currentState))
	// sha := hex.EncodeToString(hash[:])

	// log.Println("GetBuildLineNumbersSysPrompt currentState sha:", sha)

	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)

	return getListChangesLineNumsPrompt() + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n" + getBuildPromptWithLineNums(desc, changes)
}

func GetBuildFixesLineNumbersSysPrompt(desc, changes, updated, reasoning string) string {
	// hash := sha256.Sum256([]byte(updated))
	// sha := hex.EncodeToString(hash[:])
	// log.Println("GetBuildFixesLineNumbersSysPrompt updated sha:", sha)

	updatedWithLineNums := shared.AddLineNums(updated)

	return getFixChangesLineNumsPrompt() + "\n\n" + getBuildPromptForFixesWithLineNums(desc, changes, updatedWithLineNums, reasoning)
}

func getBuildPromptWithLineNums(desc, changes string) string {
	s := ""

	if desc != "" {
		s += "Description of the proposed updates from AI-generated plan:\n```\n" + desc + "\n```\n\n"
	}

	s += "Proposed updates:\n```\n" + changes + "\n```"

	s += "\n\n" + "Now call the 'listChangesWithLineNums' function with a valid JSON array of changes according to your instructions. You must always call 'listChangesWithLineNums' with one or more valid changes. Don't call any other function."

	return s
}

func getBuildPromptForFixesWithLineNums(desc, changes, updated, reasoning string) string {
	s := ""

	if desc != "" {
		s += "Description of the proposed updates from AI-generated plan:\n```\n" + desc + "\n```\n\n"
	}

	s += "Proposed updates:\n```\n" + changes + "\n```"

	s += fmt.Sprintf("**The incorrect updated file is:**\n```\n%s\n```\n\n**The problems with the file are:**\n\n%s", updated, reasoning)

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

const lineNumsFunctionCallPrompt = `
Call the 'listChangesWithLineNums' function with a valid JSON object that includes the 'changes' keys.

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
The 'old' property is an object with 2 properties: 'startLineString' and 'endLineString'.

	'startLineString' is the **entire, exact line** where the section to be replaced begins in the original file, including the line number. Unless it's the first change, 'startLineString' ABSOLUTELY MUST begin with a line number that is HIGHER than both the 'endLineString' of the previous change and the 'startLineString' of the previous change.
	
	If the previous change's 'endLineString' starts with 'pdx-75: ', then the current change's 'startLineString' MUST start with 'pdx-76: ' or higher. It MUST NOT be 'pdx-75: ' or lower. If the previous change's 'startLineString' starts with 'pdx-88: ' and the previous change's 'endLineString' is an empty string, then the current change's 'startLineString' MUST start with 'pdx-89: ' or higher. If the previous change's 'startLineString' starts with 'pdx-100: ' and the previous change's 'endLineString' starts with 'pdx-105: ', then the current change's 'startLineString' MUST start with 'pdx-106: ' or higher.
	
	'endLineString' is the **entire, exact line** where the section to be replaced ends in the original file. Pay careful attention to spaces and indentation. 'startLineString' and 'endLineString' must be *entire lines* and *not partial lines*. Even if a line is very long, you must include the entire line, including the line number and all text on the line.
	
	**For a single line replacement, 'endLineString' MUST be an empty string.**

	'endLineString' MUST ALWAYS come *after* 'startLineString' in the original file. It must start with a line number that is HIGHER than the 'startLineString' line number. If 'startLineString' starts with 'pdx-22: ', then 'endLineString' MUST either be an empty string (for a single line replacement) or start with 'pdx-23: ' or higher (for a multi-line replacement).	

	If 'hasChange' is false, both 'startLineString' and 'endLineString' must be empty strings. If 'hasChange' is true, 'startLineString' and 'endLineString' must be valid strings.
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
`

const lineNumsRulesPrompt = `
You ABSOLUTELY MUST NOT generate overlapping changes. Group smaller changes together into larger changes where necessary to avoid overlap. Only generate multiple changes when you are ABSOLUTELY CERTAIN that they do not overlap--otherwise group them together into a single change. If changes are close to each other (within several lines), group them together into a single change. You MUST group changes together and make fewer, larger changes rather than many small changes, unless the changes are completely independent of each other and not close to each other in the file. You MUST NEVER generate changes that are adjacent or close to adjacent. Adjacent or closely adjacent changes MUST ALWAYS be grouped into a single larger change.

Furthermore, unless doing so would require a very large change because some changes are far apart in the file, it's ideal to call the 'listChangesWithLineNums' with just a SINGLE change.

Changes must be ordered in the array according to the order they appear in the file. The 'startLineString' of each 'old' property must come after the 'endLineString' of the previous 'old' property. Changes MUST NOT overlap. If a change is dependent on another change or intersects with it, group those changes together into a single change.
`

const changeRulesPrompt = `
Apply changes intelligently in order to avoid syntax errors, breaking code, or removing code from the original file that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

You ABSOLUTELY MUST NOT overwrite or delete code from the original file unless the plan *clearly intends* for the code to be overwritten or removed. Do NOT replace a full section of code with only new code unless that is the clear intention of the plan. Instead, merge the original code and the proposed changes together intelligently according to the intention of the plan. 

Pay *EXTREMELY close attention* to opening and closing brackets, parentheses, and braces. Never leave them unbalanced when the changes are applied. Also pay *EXTREMELY close attention* to newlines and indentation. Make sure that the indentation of the new code is consistent with the indentation of the original code, and syntactically correct.
`

const lineNumsJsonPrompt = `
The 'listChangesWithLineNums' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChangesWithLineNums' function call parameters JSON object *must be properly escaped* with a backslash.
`

func getListChangesLineNumsPrompt() string {

	return replacementIntro + `

	[YOUR INSTRUCTIONS]
	` + lineNumsFunctionCallPrompt + `

	` + summaryChangePrompt + `

	` + lineNumsOldPrompt + `
  
  ` + changeLineInclusionAndNewPrompt + `

  Example function call with all keys:
  ---
  listChangesWithLineNums([{
    summary: "Fix syntax error in loop body.",
   	old: {
      startLineString: "pdx-5: for i := 0; i < 10; i++ { ",
      endLineString: "pdx-7: }",
    },
    new: "for i := 0; i < 10; i++ {\n  execQuery()\n  }\n  }\n}",
  }])
  ---

	` + lineNumsRulesPrompt + `

	` + changeRulesPrompt + `

	` + lineNumsJsonPrompt + `
 
  [END YOUR INSTRUCTIONS]
`
}

func getFixChangesLineNumsPrompt() string {

	return `
	You are an AI that analyzes an incorrect updated file, the changes that should have been applied to the file, a description of the problems with the file, and a plan to fix them, and then produces a list of changes to apply to the *incorrect updated file* that will fix *ALL* the problems.

	Problems you MUST fix include:
	- Syntax errors
	- Incorrectly applied changes
	- Incorrectly removed code
	- Incorrectly overwritten code
	- Incorrectly duplicated code
	- Incorrectly applied comments that reference the original code

	If the updated includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods..." **any other reference to the original code, the file is incorrect. References like these must be handled by including the exact code from the original file that the comment is referencing.

	[YOUR INSTRUCTIONS]
	` + lineNumsFunctionCallPrompt + `

	` + summaryChangePrompt + `

  ` + lineNumsOldPrompt + `
	
	` + changeLineInclusionAndNewPrompt + `

	You MUST ensure the line numbers for the 'old' property correctly remove *ALL* code that has problems and that the 'new' property correctly fixes *ALL* the problems present in the updated file. You MUST NOT miss any problems, fail to fix any problems, or introduce any new problems.

  Example function call with all keys:
  ---
  listChangesWithLineNums([{
    summary: "Fix syntax error in loop body.",
    old: {
      startLineString: "pdx-5: for i := 0; i < 10; i++ { ",
      endLineString: "pdx-7: }",
    },
    new: "for i := 0; i < 10; i++ {\n  execQuery()\n  }\n  }\n}",
  }])
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
		Required: []string{"changes"},
	},
}

func GetVerifyPrompt(preBuildState, updated, desc, changes string) string {
	s := `
Based on an original file (if one exists), an AI-generated plan, and an updated file, determine whether the proposed changes were applied correctly to the updated file. Is the syntax in the updated file correct? Were the changes applied correctly or was some code from the original file removed or overwritten that should not have been? Does the updated file as a whole make sense and was it updated consistently with the intention of the plan? Were *all* comments in the proposed changes that referenced the original code correctly handled in the updated file by including the exact code from the original file that the comment was referencing? Did any code get duplicated that should not have been?

If the updated includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods...", "// Existing code..." **any other reference to the original code**, the file is incorrect. References like these must be handled by including the exact code from the original file that the comment is referencing.

If there is no original file, it means that a new file was created from scratch based on the AI-generated plan. In this case, the syntax in the new file must be valid and consistent with the intention of the plan. You must ensure there are no syntax errors or other clear mistakes in the new file.

Call the 'verifyOutput' function with a valid JSON object that include the 'reasoning' and 'isCorrect' keys.

'reasoning': Succinctly explain whether the proposed changes were or were not applied correctly and whether the syntax is valid. If the changes were not applied correctly or the syntax isn't valid, list *EVERY* problem and what needs to be done to fix *ALL* the errors. If the syntax isn't valid only because the syntax wasn't valid in the original file, explain that the syntax, though incorrect, is consistent with the original file and that the changes were applied correctly.

'isCorrect': A boolean that indicates whether the proposed changes were applied correctly. If the proposed changes were applied correctly, set 'isCorrect' to true. If the proposed changes were not applied correctly, set 'isCorrect' to false.
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
## **Description of the proposed updates from AI-generated plan:**

` + desc + `

--

## **Proposed updates:**

` + changes + `

--
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

	s += updated + `

Now call the 'verifyOutput' function with a valid JSON object that includes the 'reasoning', and 'isCorrect' keys. You must always call 'verifyOutput' with a valid JSON object. Don't call any other function.`

	return s
}

var VerifyOutputFn = openai.FunctionDefinition{
	Name: "verifyOutput",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"reasoning": {
				Type: jsonschema.String,
			},
			"isCorrect": {
				Type: jsonschema.Boolean,
			},
		},
	},
}

// Saving the prompts below to maybe take a stab at full replacements in the future -- not robust enough for now

// func GetBuildFullChangesSysPrompt(filePath, currentState, desc, changes string) string {

// 	return fullChangesPrompt + "\n\n" + getBuildCurrentStatePrompt(filePath, currentState) + "\n\n" + getBuildPromptWithFullChanges(desc, changes)
// }

// func getBuildPromptWithFullChanges(desc, changes string) string {
// 	s := ""

// 	if desc != "" {
// 		s += "Description of the proposed updates from AI-generated plan:\n```\n" + desc + "\n```\n\n"
// 	}

// 	s += "Proposed updates:\n```\n" + changes + "\n```"

// 	s += "\n\n" + "Now call the 'listChangesFull' function with a valid JSON array of changes according to your instructions. You must always call 'listChangesFull' with one or more valid changes. Don't call any other function."

// 	return s
// }

// var ListReplacementsFullFn = openai.FunctionDefinition{
// 	Name: "listChangesFull",
// 	Parameters: &jsonschema.Definition{
// 		Type: jsonschema.Object,
// 		Properties: map[string]jsonschema.Definition{
// 			"changes": {
// 				Type: jsonschema.Array,
// 				Items: &jsonschema.Definition{
// 					Type: jsonschema.Object,
// 					Properties: map[string]jsonschema.Definition{
// 						"summary": {
// 							Type: jsonschema.String,
// 						},
// 						"old": {
// 							Type: jsonschema.String,
// 						},
// 						"new": {
// 							Type: jsonschema.String,
// 						},
// 					},
// 					Required: []string{"summary", "old", "new"},
// 				},
// 			},
// 		},
// 		Required: []string{"changes"},
// 	},
// }

// const fullChangesPrompt = replacementIntro + `

// 	[YOUR INSTRUCTIONS]
// 	Call the 'listChangesFull' function with a valid JSON object that includes the 'changes' keys.

// 	'changes': An array of NON-OVERLAPPING changes. Each change is an object with properties: 'summary', 'old', and 'new'.

// 	` + summaryPrompt + `

//   The 'old' property is the *exact* old text to replace. 'old' must be an EXACT SUBSTRING of the original file. 'old' *must only include entire lines and *no partial lines*. Pay *extremely* close attention to quotes, indentation, newlines, spaces and special characters. All double quotes inside 'old' MUST ALWAYS be escaped with a backslash. Apart from escaped double quotes, the 'old' property must be an exact match of the original code that will be replaced. 'old' must be large enough to be *unique and unambiguous* within the file. If the 'old' property occurs in multiple places in the file, you must include enough context to ensure that the 'old' property is unambiguous. Expand 'old' as necessary. 'old' must be an *exact* match of the original code that will be replaced.

//   ` + changeNewPrompt + `

//   Example function call with all keys:
//   ---
//   listChangesFull([{
//     summary: "Insert function ExecQuery after GetResults function in loop body.",
//     old: "      GetResults()\n    }\n   }\n }",
//     new: "      GetResults()\n     execQuery()\n    }\n  }\n}",
//   }])
//   ---

// 	` + changeRulesPrompt + `

// 	The 'listChangesFull' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChangesFull' function call parameters JSON object *must be properly escaped* with a backslash.

//   [END YOUR INSTRUCTIONS]
// `
