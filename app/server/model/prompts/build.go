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

	return getListChangesLineNumsPrompt(false) + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n" + getBuildPromptWithLineNums(changes)
}

func GetBuildLineNumbersSysPromptUpdated(filePath, preBuildState, changesFile, changesDesc, nonExpandedRefs string) string {
	// hash := sha256.Sum256([]byte(currentState))
	// sha := hex.EncodeToString(hash[:])

	// log.Println("GetBuildLineNumbersSysPrompt currentState sha:", sha)

	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	res := getListChangesLineNumsPrompt(nonExpandedRefs != "") + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n" + "**Proposed updates:**\n" + changesDesc

	if nonExpandedRefs == "" {
		res += "\n```\n" + changesWithLineNums + "\n```"
	} else {
		res += "\n\nWithout expanded references:\n\n```\n" + nonExpandedRefs + "\n```" + "\n\n" + "With expanded references:\n\n```\n" + changesWithLineNums + "\n```"
	}

	return res
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

const entireFilePrompt = `
In the 'entireFileReasoning' key, evaluate whether the *proposed updates* contain the final state of the ENTIRE updated file *with all changes applied* and with nothing missing. Evaluate whether the *proposed updates* contain the file updated state of the entire file. Think carefully through the sections of the file and evaluate whether every section that should be included in the final result is present in the *proposed updates*. Evaluate whether the *original file* contains any code that is *NOT* inlcluded in the *proposed updates* AND *SHOULD* be included in the final result.

Note that here you are *not* evaluating which sections have changed or not changed, but only whether the *proposed updates* contains the entire updated file.

You are also NOT evaluating whether the *proposed updates* contains any new code that is not part of the *original file*. You are only evaluating whether ALL the code from both the *original file* and *proposed updates* that should be present in the final result when the two are combined is present in the *proposed updates*.

The 'entireFile' key should be set to true if the *proposed updates* contain the final state of the ENTIRE updated file *with all changes applied* and with nothing missing. If the *proposed updates* do not contain the final state of the ENTIRE updated file *with all changes applied* and with nothing missing, set 'entireFile' to false.

If 'entireFile' is set to true, 'changes' must be an empty array and 'problems' must be an empty string.
`

const changesKeyPrompt = `
'changes': An array of changes. Each change is an object with properties: 'section', 'summary', 'newReasoning', 'reasoning', 'structureReasoning', 'closingSyntaxReasoning', 'orderReasoning', 'hasChange', 'insertBefore', 'insertAfter', 'old', 'new'

If 'entireFile' is set to true, 'changes' must be an empty array.

Note: all line numbers that are used below are prefixed with 'pdx-' when referencing lines in the *original file*, like this 'pdx-5: for i := 0; i < 10; i++ {'. This is to help you identify the line numbers in the *original file*. You *must* include the 'pdx-' prefix in the line numbers in the 'old' property. 

'insertBefore' is an object with properties: 'shouldInsertBefore' (boolean), 'line' (string).

	If 'shouldInsertBefore' is true:
		- 'line' must be *entire line* that *exactly match* a line from the *original file* including the line number with 'pdx-' prefix.
		- The 'old' property of the change should be filled with empty strings.
		- The 'new' key of the change should contain the content to be inserted before the specified line. 

'insertAfter' is an object with properties: 'shouldInsertAfter' (boolean), 'line' (string).

	If 'shouldInsertAfter' is true:
		- 'line' must be *entire line* that *exactly match* a line from the *original file* including the line number with 'pdx-' prefix.
		- The 'old' property of the change should be filled with empty strings.
		- The 'new' key of the change should contain the content to be inserted after the specified line. 

	If 'shouldInsertAfter' is false, then 'line' should be an empty string. 'old' and 'new' should be filled according to your other instructions.

If 'insertBefore', or 'insertAfter' is set to true, the 'old' property of the change ABSOLUTELY MUST be filled by empty strings. Do NOT fill the 'old' property with any other values if 'insertBefore', or 'insertAfter' is set to true. The 'new' property should be filled according to your other instructions to the exact, correct section of the *proposed updates* that should be prepended, appended, inserted between, inserted before, or inserted after the specified lines.

Expand the lines the 'new' property refers to in the *proposed updates* where needed to include line breaks and preserve formatting.

Note: all line numbers that are used below are prefixed with 'pdx-' when referencing lines in the *original file*, like this 'pdx-5: for i := 0; i < 10; i++ {'. This is to help you identify the line numbers in the *original file*. You *must* include the 'pdx-' prefix in the line numbers in the 'old' property. 

Line numbers that are used below are prefixed with 'pdx-new-' when referencing lines in the *proposed updates*, like this 'pdx-new-5: for i := 0; i < 10; i++ {'. This is to help you identify the line numbers in the *proposed updates*. You *must* include the 'pdx-new-' prefix in the line numbers in the 'new' property.
`

const summaryChangePrompt = `
The 'section' property is the name of the section from the 'originalSections' array that the change applies to. Each change must apply to a single section.

The 'summary' property is a brief summary of the change.

'summary' examples: 
	- 'Update loop that aggregates the results to iterate 10 times instead of 5 and log the value of someVar.'
	- 'Update the Org model to include StripeCustomerId and StripeSubscriptionId fields.'
	- 'Add function ExecQuery to execute a query.'

The 'newReasoning' property evaluates what needs to be included in the 'new' section to ensure that *ALL* the code from the *proposed changes* are correctly applied to the resulting file. Carefully consider whether the code from the *proposed updates* contain code structures that need to be properly closed in the resulting file. For example, "the <Route> tag needs to be closed properly in the resulting file, so the 'new' section should end on the closing tag at pdx-new-10: </Route>".

The 'structureReasoning' property is an object with 2 properties: 'old' and 'new'.

	The 'old' property is an object with 3 properties: 'structure', 'structureOpens', and 'structureCloses'.
		
		'structure' is the *code structure* (e.g. 'function', 'class', 'loop', 'conditional', etc.) that this change is contained by. If the change is not contained within a code structure and is instead at the top level of the file, output 'top level'. This must be the MOST specific, deeply nested code structure that contains the change. You must output only a single structure or 'top level'. Identify the structure unambiguously. If a structure is being *replaced*, the 'structure' property must identify the exact structure that is being replaced. If new code is being added to a structure, the 'structure' property must identify the structure that the new code is being added to.

		'structureOpens' is the entire line from the *original file* that contains the opening symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-'. Empty if the code structure is 'top level'.

		'structureCloses' is the entire line from the *original file* that contains the closing symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-'. Empty if the code structure is 'top level'.

		When determining 'structureOpens' and 'structureCloses' properties for 'old', evaluate both the *original file* AND the *proposed updates* to help you identify the correct closing symbol.

	The 'new' property is an object with 3 properties: 'structure', 'structureOpens', and 'structureCloses'.

		'structure' is the *code structure* (e.g. 'function', 'class', 'loop', 'conditional', etc.) in the *proposed updates* that should be inserted. If the code that should be inserted is 'flat' and not contained within a code structure, either because the *proposed updates* themselves are 'flat' or because surrounding code structures in the *proposed updates* are already contained by the *original file*, output 'flat'. If you output a structure and not 'flat', it must be the MOST specific, deeply nested code structure that contains the new code to be inserted. You must output only a single structure or 'flat'. Identify the structure unambiguously. 

		'structureOpens' is the entire line from the *proposed updates* that contains the opening symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-new-'. Must be an EXACT MATCH to a line from the *proposed updates*. Empty if the code structure is 'flat'.

		'structureCloses' is the entire line from the *proposed updates* that contains the closing symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-new-'. Must be an EXACT MATCH to a line from the *proposed updates*. Empty if the code structure is 'flat'.

If you are adding new code to the *end* of an existing code structure (and not replacing any existing code), you must use the 'insertBefore' property to add the new code before the closing symbol of the existing code structure.

If you are adding new code to the *beginning* of an existing code structure (and not replacing any existing code), you must use the 'insertAfter' property to add the new code after the opening symbol of the existing code structure.

If you are replacing existing code, you must always use the 'old' property to specify the code that should be replaced.

The 'closingSyntaxReasoning' property, for each code structure related to the change, list what is the BOTTOM-MOST, RIGHT-MOST, *FINAL* closing symbol after which *ALL* code structures in the section are closed.

For example, if there is a section like:

---
pdx-new-32: <Route
pdx-new-33:   path="/instructor-grid"
pdx-new-33:   element={loggedIn ? <InstructorGridPage /> : <Navigate to=\"/\" />} />
pdx-new-34: />
--

The 'closingSyntaxReasoning' property should explain that the final closing symbol is /> on line pdx-new-34. Even though the 'InstructorGridPage' and 'Navigate' code structures are closed properly on pdx-new-33, it is NOT the final closing symbol because the 'Route' code structure is not closed. The 'closingSyntaxReasoning' property in this example would explain that the final closing symbol that closes all code structures in the section is /> on line pdx-new-34.

The 'closingSyntaxReasoning' section ABSOLUTELY MUST include a mention of each section that is specified in the 'structureReasoning' property. If the 'insertAfter' property is used, the 'closingSyntaxReasoning' section MUST include a mention of the 'insertAfter' section. And the 'closingSyntaxReasoning' section MUST ALWAYS include a mention of the 'new' section.

The same applies to the 'line' property of the 'insertAfter' section. For example, if in the *original file* there is a section like:

---
pdx-20: <Route
pdx-21:   path="/instructor-grid"
pdx-22:   element={loggedIn ? <InstructorGridPage /> : <Navigate to=\"/\" />} />
pdx-23: />
--

For example, if the plan is to insert code from the *proposed updates* after the 'Route' code structure using the 'insertAfter' property, then 'closingSyntaxReasoning' property should explain that the final closing symbol is /> on line pdx-23. Even though the 'InstructorGridPage' and 'Navigate' code structures are closed properly on pdx-22, it is NOT the final closing symbol because the 'Route' code structure is not closed. The 'closingSyntaxReasoning' property in this example would explain that the final closing symbol that closes all code structures in the section is /> on line pdx-23.

'closingSyntaxReasoning' must use this template: 'Old: [syntax reasoning for old section] | New: [syntax reasoning for new section]'.

The 'orderReasoning' property evaluates how the change should be applied in order to preserve the order of code from the *proposed updates*. State at what position the new code would need to be inserted (or to overwrite) in order to preserve the order of code from the *proposed updates*. For example, "To ensure the order is preserved, it's important that the new import of 'fmt' ('pdx-new-3: import "fmt"') should be added after the import of 'os' ('pdx-1: import "os"') and before the import of 'log' ('pdx-2: import "log"')." Do not state in this key what should be the start and end lines of the change, either in the *original file* or the *proposed updates*. Instead, make note of which invariants need to be preserved in terms of the order of the code in order for the *proposed updates* to be applied correctly.

The 'hasChange' property is a boolean that indicates whether there is anything to change. If there is nothing to change, set 'hasChange' to false. If there is something to change, set 'hasChange' to true.

The 'reasoning' property is an explanation of how the change relates to the strategy outlined in the 'problems' key and in the 'orginalSections' array from above. Which 'originalSections' does this change apply to? 

If this is a partial change , where should the change begin and end in the *original file* in order to avoid removing or overwriting code that should not be removed? Include specific line numbers prefixed with 'pdx-'. This explanation MUST be include in the 'reasoning' key for partial changes, and the explanation must begin with "To avoid removing". Example: "To avoid removing any existing imports, the change should be begin on line pdx-5 to avoid removing the existing imports."

You must then consider what the first and last line of the 'new' section should be. Should the 'new' section start on a function signature and end on a closing bracket? Start on a variable declaration and end on a newline? Start on an opening tag and end on a closing tag? What needs to be included in the 'new' section to ensure the code will be syntactically correct and the plan will be applied as intended? Answer these questions.

You MUST include closing symbols in the 'new' section such that EVERY code structure in the *proposed updates* is closed properly. ALWAYS expand the 'new' section to include closing syntax that is related to the code structures present in the 'new' section.

Next, state your reasoning in the format "Because [reason(s)] the last line of the new section should be the [line of code explained in words] on [line number in the *proposed updates* prefixed with 'pdx-new-']. And because [reason(s)], the first line of the new section should be the [line of code explained in words] on [line number in the *proposed updates* prefixed with 'pdx-new-'].". The 'reasoning' key ABSOLUTELY MUST include your stated reasoning in the afore-stated format. You must never leave it out.

If the next non-newline character after the change in the *original file* is a *closing symbol* for a code structure like a closing bracket, parenthesis, quote, brace, tag, etc., explicitly list each closing symbol and state whether it closes a code structure that is present in the 'old' section. If it does, state that the 'old' section must now be expaned to include this closing symbol, and state what the new last line of the 'old' section should be, including the line number prefixed with 'pdx-'.

If the next non-newline character after the change in the *proposed updates* is an *closing symbol* for a code structure like a closing bracket, parenthesis, quote, brace, tag, etc., explicitly list each closing symbol and state whether it closes a code structure that is present in the 'new' section. If it does, state that the 'new' section must now be expanded to include this closing symbol, and state what the new last line of the 'new' section should be, including the line number prefixed with 'pdx-new-'.

You must not try to make the reasoning in any reasoning key too succinct. It needs to properly address these issues in order for the change to be correctly applied in complex scenarios. Make sure you have fully evaluated the considerations and consequences of how this change will be applied. Be thorough, meticulous, and perfectionist. Use the space needed to fully explain the reasoning and spell out in detail all the factors that need to be considered, even if the reasoning seems obvious.
`

const lineNumsOldPrompt = `
The 'old' property is an object with 2 properties: 'startLineString' and 'endLineString'.

	'startLineString' is the **entire, exact line** where the section to be replaced begins in the *original file*, including the line number. Unless it's the first change, 'startLineString' ABSOLUTELY MUST begin with a line number that is HIGHER than both the 'endLineString' of the previous change and the 'startLineString' of the previous change. **The line number and line MUST EXACTLY MATCH a line from the *original file*.** It must be only a *SINGLE LINE* and not contain any line breaks.
	
	If the previous change's 'old' property 'endLineString' starts with 'pdx-75: ', then the current change's 'startLineString' MUST start with 'pdx-76: ' or higher. It MUST NOT be 'pdx-75: ' or lower. If the previous change's 'old' property 'startLineString' starts with 'pdx-88: ' and the previous change's 'old' property 'endLineString' is an empty string, then the current change's 'startLineString' MUST start with 'pdx-89: ' or higher. If the previous change's 'old' property 'startLineString' starts with 'pdx-100: ' and the previous change's 'old' property 'endLineString' starts with 'pdx-105: ', then the current change's 'startLineString' MUST start with 'pdx-106: ' or higher.
	
	'endLineString' is the **entire, exact line** where the section to be replaced ends in the *original file*. Pay careful attention to spaces and indentation. 'startLineString' and 'endLineString' must be *entire lines* and *not partial lines*. Even if a line is very long, you must include the entire line, including the line number and all text on the line. **The line number and line MUST EXACTLY MATCH a line from the *original file*.** It must be only a *SINGLE LINE* and not contain any line breaks.
	
	**For a single line replacement, 'endLineString' MUST be an empty string.**

	If the 'reasoning' key above includes guidance on where to start and/or end the change, use that guidance to determine the 'startLineString' and 'endLineString'. If the reasoning key states that the change should begin at a specific line number to avoid removing or overwriting code that should not be removed, ensure that 'startLineString' starts with that line number. If the reasoning key states that the change should end at a specific line number to avoid removing or overwriting code that should not be removed, ensure that 'endLineString' ends with that line number. You MUST follow the guidance in the 'reasoning' key.

	'endLineString' MUST ALWAYS come *after* 'startLineString' in the *original file*. It must start with a line number that is HIGHER than the 'startLineString' line number. If 'startLineString' starts with 'pdx-22: ', then 'endLineString' MUST either be an empty string (for a single line replacement) or start with 'pdx-23: ' or higher (for a multi-line replacement).	

	If 'hasChange' is false both 'startLineString' and 'endLineString' must be empty strings. If 'hasChange' is true, 'startLineString' and 'endLineString' must be valid strings that exactly match lines from the *original file*. If 'hasChange' is true, 'startLineString' and 'endLineString' MUST NEVER be empty strings.
`

const changeLineInclusionAndNewPrompt = `
The 'new' property is an object with 2 properties: 'startLineString' and 'endLineString'.

	'startLineString' is the **entire, exact line** where the updated code that the 'old' section will be replaced with begins *in the *proposed updates**, including the line number. Unless it's the first change, 'startLineString' ABSOLUTELY MUST begin with a line number that is HIGHER than both the 'new' property 'endLineString' of the previous change and the 'startLineString' of the previous change. **The line number and line MUST EXACTLY MATCH a line from the *proposed updates*.** It must be only a *SINGLE LINE* and not contain any line breaks. IT ABSOLUTELY MUST match a line from the **proposed updates**.
	
	If the previous change's 'new' property 'endLineString' starts with 'pdx-new-75: ', then the current change's 'startLineString' MUST start with 'pdx-new-76: ' or higher. It MUST NOT be 'pdx-new-75: ' or lower. If the previous change's 'new' property 'startLineString' starts with 'pdx-new-88: ' and the previous change's 'new' property 'endLineString' is an empty string, then the current change's 'startLineString' MUST start with 'pdx-new-89: ' or higher. If the previous change's 'new' property 'startLineString' starts with 'pdx-new-100: ' and the previous change's 'new' property 'endLineString' starts with 'pdx-new-105: ', then the current change's 'startLineString' MUST start with 'pdx-new-106: ' or higher.
	
	'endLineString' is the **entire, exact line** where the section to be replaced ends in the *proposed updates*. Pay careful attention to spaces and indentation. 'startLineString' and 'endLineString' must be *entire lines* and *not partial lines*. Even if a line is very long, you must include the entire line, including the line number and all text on the line. **The line number and line MUST EXACTLY MATCH a line from the *proposed updates*.** It must be only a *SINGLE LINE* and not contain any line breaks. IT ABSOLUTELY MUST match a line from the **proposed updates**. For multi-line changes, the 'endLineString' of the 'new' section MUST NOT be empty.
	
	**For a single line replacement, 'endLineString' MUST be an empty string.**

	'endLineString' MUST ALWAYS come *after* 'startLineString' in the *proposed updates*. It must start with a line number that is HIGHER than the 'startLineString' line number. If 'startLineString' starts with 'pdx-new-22: ', then 'endLineString' MUST either be an empty string (for a single line replacement) or start with 'pdx-new-23: ' or higher (for a multi-line replacement).	

	If 'hasChange' is false, both 'startLineString' and 'endLineString' must be empty strings. If 'hasChange' is true, 'startLineString' and 'endLineString' must be valid strings that exactly match lines from the *proposed updates*. If 'hasChange' is true, 'startLineString' and 'endLineString' MUST NEVER be empty strings.

If the 'hasChange' property is false, the 'new' property must be an empty string. If the 'hasChange' property is true, the 'new' property must be a valid string.

Expand the lines the 'new' property refers to in the *proposed updates* where needed to include line breaks and preserve formatting.

'includeOldReasoning' is a string. Evaluate whether the section of code from the *original file* that the old property refers to should be included in the final result. 

If the code from the 'new' section should fully replace the code from the 'old' property, say "Fully replace" and stop.

If the 'old' section spans multiple lines of the *original file*, say "Multiple lines, don't include." and stop.

Otherwise, if the code from the 'old' property is a *single line* and it's not in the *proposed updates*, state this. In that case, proceed to evaluate whether the code from the old property makes more sense at the beginning or end of the change from the new property. State which makes more sense and why. For example, "The closing </Routes> tag should be included in the final result. It should be added *after* the 'new' code from the *proposed updates*.

'appendOld' must be true if 'includeOldReasoning' determined that the code from the old property should be included in the final result and that it makes the most sense to include it at the end of the change from the new property.

'prependOld' must be true if 'includeOldReasoning' determined that the code from the old property should be included in the final result and that it makes the most sense to include it at the beginning of the change from the new property.
`

const changeRulesPrompt = `
Apply changes intelligently **in order** to avoid syntax errors, breaking code, or removing code from the *original file* that should not be removed. Consider the reason behind the update and make sure the result is consistent with the intention of the plan.

Changes MUST be ordered based on their position in the *original file*. ALWAYS go from top to bottom IN ORDER when generating changes.

When generating changes, you MUST consider the list of sections from the 'originalSections' array. If a section is marked as 'shouldChange', there MUST be at least one corresponding change in the 'changes' array that addresses that section.

You ABSOLUTELY MUST NOT overwrite or delete code from the *original file* unless the plan *clearly intends* for the code to be overwritten or removed. Do NOT replace a full section of code with only new code unless that is the clear intention of the plan. Instead, merge the original code and the *proposed updates* together intelligently according to the intention of the plan. 

Pay *EXTREMELY close attention* to opening and closing brackets, parentheses, and braces. Never leave them unbalanced when the changes are applied. Also pay *EXTREMELY close attention* to newlines and indentation. Make sure that the indentation of the new code is consistent with the indentation of the original code, and syntactically correct. Do NOT remove extraneous newlines from the *original file* unless it is explicitly mentioned in the *proposed updates*.

You ABSOLUTELY MUST NOT generate overlapping changes.

Changes must be ordered in the array according to the order they appear in the *original file*. The 'startLineString' of each 'old' property ABSOLUTELY MUST come after the 'endLineString' of the previous 'old' property. If the 'problems' key includes guidance on the order that changes should be applied, you MUST follow that guidance and generate the changes in the order specified.

You MUST NOT repeat changes to the same block of lines multiple teams. You MUST NOT duplicate changes. It is extremely important that a given change is only applied *once*.

ALL THE CODE in the *proposed updates* must be accounted for in the 'changes' array. You MUST NOT leave out any new code, changes, or deletions that are introduced in the *proposed updates*.

Break up changes when they apply to different sections of the file, as listed in the 'originalSections' array.
`

const lineNumsJsonPrompt = `
The 'listChangesWithLineNums' function MUST be called *valid JSON*. Double quotes within json properties of the 'listChangesWithLineNums' function call parameters JSON object *must be properly escaped* with a backslash. Pay careful attention to newlines, tabs, and other special characters. The JSON object must be properly formatted and must include all required keys. **You generate perfect JSON -every- time**, no matter how many quotes or special characters are in the input. You must always call 'listChangesWithLineNums' with a valid JSON object. Don't call any other function. 
`

const commentsPrompt = `
The 'comments' key is an array of objects with two properties: 'txt' and 'reference'. 'txt' is the exact text of a code comment. 'reference' is a boolean that indicates whether the comment is a placeholder of or reference to the original code, like "// rest of the function..." or "# existing init code...", or "// rest of the main function" or "// rest of your function..." or "// Existing methods..." or "// Remaining methods" or "// Existing code..." or "// ... existing setup code ..." or "// Existing code..." or "// ... existing code ..." or "// ..." or other comments which reference code from the *original file*. References DO NOT need to exactly match any of the previous examples. Use your judgement to determine whether each comment is a reference. If 'reference' is true, the comment is a placeholder or reference to the original code. If 'reference' is false, the comment is not a placeholder or reference to the original code.

In 'comments', you must list EVERY comment included in the *proposed updates*. Only list *code comments* that are valid comments for the programming language being used. Do not list logging statements or any other non-comment text that is not a valid code comment. If there are no code comments in the *proposed updates*, 'comments' must be an empty array.

If there are multiple identical comments in the *proposed updates*, you MUST list them *all* in the 'comments' array--list each identical comment as a separate object in the array.
`

const problemsStrategyPrompt = `
In the 'problems' key, you MUST explain how you will strategically generate changes in order to avoid any problems in the updated file.

If 'entireFile' is set to true, 'problems' must be an empty string.

You must consider whether you will apply partial changes, prepend or append to the file, or insert before or after specific lines. Choose whichever is the simplest, most effective, and least error-prone method of making your changes. Before choosing, evaluate the consequences of each option and choose the one that will cause the fewest errors and the easiest changes.

You must consider how you will avoid *incorrectly removing or overwriting code* from the *original file*. Explain whether any code from the *original file* needs to be merged with the *proposed updates* in order to avoid removing or overwriting code that should not be removed. 

It is ABSOLUTELY CRITICAL that no pre-existing code or functionality is removed or overwritten unless the plan explicitly intends for it to be removed or overwritten. New code and functionality introduced in the *proposed updates* MUST be *merged* with existing code and functionality in the *original file*. Explain how you will achieve this. 

You must consider how you will *avoid incorrect duplication* in making your changes. For example if a 'main' function is present in the *original file* and the *proposed updates* include update code for the 'main' function, you must ensure the changes are applied within the existing 'main' function rather than incorrectly adding a duplicate 'main' function.

If the *proposed updates* include large sections that are identical to the *original file*, consider whether the changes can be made more minimal in order to only replace sections of code that are *changing*. If you are making the changes more minimal and specific, explain how you will do this.

You MUST also explicitly state the *order* that all the changes you've listed will be applied. Changes MUST be applied in the order they appear in the *original file*. ALWAYS go from top to bottom in order when making your changes.
`

const originalSectionsPrompt = `
The 'originalSections' key is an array of objects with four properties: 'description', 'structure', 'reasoning', 'sectionStartLine', 'sectionEndLine', 'shouldChange', and 'shouldRemove'. In the 'originalSections' key you must logically divide the *original file* into sections based on functionality, logic, code structure, and general organization. 

You must list every section that exists in the *original file*. When large sections of the *original file* are not changing, combine them into a single section. Only include sections from the *original file*. Do NOT include sections from the *proposed updates*.

Don't make the sections overly small and granular unless there is a clear semantic reason to do so. For example, if there are many small functions in a file, don't create a section for each section. Sections should be larger than that and instead reflect the general structure of a file, rather than being a long list of every top-level code structure.

'description' is a brief summary of the section and its purpose.

'structure' is an object with three properties: 'structure', 'structureOpens', and 'structureCloses'.
	'structure' is the OUTERMOST code structure (namespace, class, function, etc.) of the section. State whether the full structure is contained within the section or whether the section only contains part of the structure. If the seciton is not contained within a structure, output 'top level'.
	
	'structureOpens' is the entire line from the *original file* that contains the opening symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-'. Empty if the code structure is 'top level'. Empty if the the section does not contain the opening symbol of the identified structure. If the section clearly contains an entire structure, like "SomeClass definition", then the 'structureOpens' property must be set and must be the line that opens the structure.

	'structureCloses' is the entire line from the *original file* that contains the closing symbol of the code structure identified in the 'structure' property, including the line number prefixed with 'pdx-'. Empty if the code structure is 'top level'. Empty if the the section does not contain the closing symbol of the identified structure. If the secton clearly contains an entire structure, like "SomeClass definition", then the 'structureCloses' property must be set and must be the line that closes the structure.

'reasoning' is a brief evaluation of how this section relates to the *proposed updates*, and whether all or any part of it will be changed, removed, or preserved as is. If only part of the section should be changed, explain which part(s) will change and which parts will remain the same, and further explain how you will avoid incorrectly removing or overwriting code that should not be removed.

Anytime there is a partial change to a section, it is CRITICAL that you explain BOTH which parts of the section should be *changed* and which parts will be *preserved* and how you will ensure this when generating changes below. You MUST include this explanation in the 'reasoning' key when there's a partial change. It must begin with "To avoid removing". If new code is being added, note *where* in the section it will be added, and include line numbers prefixed with 'pdx-'. For example, "To avoid removing any existing imports, the new imports will be added after the existing imports, which end on pdx-5." or "To avoid removing any existing init code, the new code for checking the GOENV environment variable will be added before the existing init code, which starts on pdx-10."

'sectionStartLine' is the line number, prefixed with 'pdx-', where the section begins in the *original file*. Include only the line number, not the line itself. If 'structure.structureOpens' is not empty, 'sectionStartLine' MUST be the same as 'structure.structureOpens'.

'sectionEndLine' is the line number, prefixed with 'pdx-', where the section ends in the *original file*. Include only the line number, not the line itself. If 'structure.structureCloses' is not empty, 'sectionEndLine' MUST be the same as 'structure.structureCloses'.

'shouldChange' is a boolean that indicates whether the section should be changed.

'shouldRemove' is a boolean that indicates whether the section should be removed.

A section should only be changed or removed if the *proposed updates* clearly intend for it to be changed or removed.
`

func getListChangesLineNumsPrompt(expandedRefs bool) string {
	res := replacementIntro + `

	[YOUR INSTRUCTIONS]

	Call the 'listChangesWithLineNums' function with a valid JSON object that includes the 'originalSections', 'entireFileReasoning', 'entireFile', 'problems', and 'changes' keys.	
	
	` + originalSectionsPrompt + `

	` + entireFilePrompt + `

	` + problemsStrategyPrompt + `
	
	` + changesKeyPrompt + `

	` + summaryChangePrompt + `

	` + lineNumsOldPrompt + `
  
  ` + changeLineInclusionAndNewPrompt

	if expandedRefs {
		res += `
			While the code changes *without* expanded references have been included to help you understand the scope of the change, the 'new' key of each change object *ABSOLUTELY MUST ALWAYS* refer to the *proposed updates* *with* expanded references and pdx-new- prefixed line numbers.	When the *proposed updates* are mentioned above, this refers to the *proposed updates* *with* expanded references and pdx-new- prefixed line numbers.
			
			Use the changes *without* expanded references to understand specifically which code is changing and what the *order* of the code structures in the final updated file should be. It's extremely important that the code is ordered exactly as it is intended in the *proposed updates*.

			If the changes *without* expanded references look like this:

			---
			class MyClass {
				// ... existing code ...
			
				someNewMethod() {
				  doStuff();
				}
			}
			---

			Then the newly added 'someNewMethod' method MUST go at the *end* of the 'MyClass' class.
		`
	}

	res += `

  Example change object:
  ---
  {
    summary: "Fix syntax error in loop body.",
   	old: {
      startLineString: "pdx-5: for i := 0; i < 10; i++ { ",
      endLineString: "pdx-7: }",
    },
    new: {
			startLineString: "pdx-new-5: for i := 0; i < 10; i++ { ",
			endLineString: "pdx-new-7: }",
		},
  }
  ---

	` + `

	` + changeRulesPrompt + `

	` + lineNumsJsonPrompt + `
 
  [END YOUR INSTRUCTIONS]
`

	return res
}
func getFixChangesLineNumsPrompt() string {

	return `
	You are an AI that analyzes an *original file* (if present), an incorrectly updated file, the changes that were proposed, and a description of the problems with the file, and then produces a list of changes to apply to the *incorrectly updated file* that will fix *ALL* the problems.

	Problems you MUST fix include:
	- Syntax errors, including unbalanced brackets, parentheses, braces, quotes, indentation, and other code structure errors
	- Missing or incorrectly scoped declarations
	- Any other errors that make the code invalid and would prevent it from being run as-is for the programming language being used
	- Incorrectly applied changes
	- Incorrectly removed code
	- Incorrectly overwritten code
	- Incorrectly duplicated code
	- Incorrectly applied comments that reference the original code

	If the updated file includes references to the original code in comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods..." **any other reference to the original code, the file is incorrect. References like these must be handled by including the exact code from the *original file* that the comment is referencing.

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
Based on an *original file* (if one exists), an AI-generated plan, an updated file, and a diff between the original and updated file, determine whether the updated file's syntax is correct and whether the *proposed updates* were applied correctly to the updated file.

You must consider whether any of the following problems are present in the updated file:
- Syntax errors, including unbalanced brackets, parentheses, braces, quotes, indentation, and other code structure errors
- Missing or incorrectly scoped declarations
- Any other errors that make the code invalid and would prevent it from being run as-is for the programming language being used
- Code from the *original file* was incorrectly removed or overwritten.
- Code was incorrectly duplicated. For example, if a file should have a single main function, but instead of updating the existing main function, the updated file includes multiple main functions, then the file is incorrect. The same applies to any other functions or elements that should not be duplicated.
- Incorrectly included comments that reference the original code.. If the updated file includes comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods...", "// Existing code..." **any other reference to the original code**, the file is incorrect. References like these must be handled by including the exact code from the *original file* that the comment is referencing.

If there is no *original file*, it means that a new file was created from scratch based on the AI-generated plan. In this case, the syntax in the new file must be valid and consistent with the intention of the plan. You must ensure there are no syntax errors or other clear mistakes in the new file.

Call the 'verifyOutput' function with a valid JSON object that include the following keys:

'syntaxErrorsReasoning': A string that succinctly explains whether there are any syntax or scoping errors in the updated file. Explain all syntax errors, scoping errors, or other code structure errors that are present in the updated file. 

'hasSyntaxErrors': A boolean that indicates whether there are any syntax errors in the updated file, based on the reasoning provided in 'syntaxErrorsReasoning'.

'removed': an array of objects with three properties: 'code', 'reasoning', and 'correct'. 
   - 'code' is a string. It shows the section of code that was removed or overwritten in the updated file. This can be abbreviated by detailing how the section starts and end and describing the purpose of the code. If the section is longer than a few lines, rather than reproducding a long section of code verbatim, provide a summary of the code that was removed or overwritten, as well as the exact code which starts and ends the section. The summary and start/end code should be details enough to disambiguate the section of code that was removed or overwritten from any other similar sections of code in the file.
   - 'reasoning' is a string that explains whether, based on the *proposed updates*, this section was deliberately removed consistent with the intention of the plan, or whether the plan did NOT specify that this section should be removed, and the code was therefore removed incorrectly. Also consider whether this removal breaks the syntax or functionality of the code in the updated file based on the programming language being used--if it does, explain this and state that the removal was incorrect. If the removal either wasn't intended by the *proposed updates* *or* breaks the syntax or functionality of the code, the removal is incorrect. If the removal was consistent with the intention of the plan *and* did not break the syntax or functionality of the code, the removal is correct.
   - 'correct' is a boolean that indicates whether the removal or overwriting was correct based on the 'reasoning' provided. If the removal was correct, set 'correct' to true. If the removal was incorrect, set 'correct' to false.

Based on supplied diffs, you must list EVERY code section that was removed or overwritten in the updated file in 'removed'. If there are no code sections that were removed or overwritten, 'removed' must be an empty array. If multiple code sections were removed or overwritten, list each one as a separate object in the 'removed' array. If multiple identical code sections were removed or overwritten, you MUST list them *all* in the 'removed' array--list each identical code section as a separate object in the array. Do NOT include removals that only modify whitespace. Do NOT include sections that were moved or refactored, only sections that were fully removed or overwritten.

'removedCodeErrorsReasoning': A string that succinctly explains whether any code was incorrectly removed or overwritten in the updated file based on the 'removed' array. If code was incorrectly removed or overwritten, succinctly explain why it was incorrect, and how the file can be corrected. If code was correctly removed or overwritten, consistent with the intention of the plan, state this.

'hasRemovedCodeErrors': A boolean that indicates whether any code was *incorrectly* removed or overwritten in the updated file, based on the reasoning provided in 'removedCodeErrorsReasoning'.

'duplicationErrorsReasoning': A string that succinctly explains whether any code was *incorrectly* duplicated in the updated file. First explain whether any code, functions, or other elements are duplicated in the updated file, then explain whether the duplication is deliberate and consistent with the plan, and whether the duplication is correct and valid in the programming language being used.

'hasDuplicationErrors': A boolean that indicates whether any code was *incorrectly* duplicated in the updated file, based on the reasoning provided in 'duplicationErrorsReasoning'. If code was *incorrectly* duplicated, set 'hasDuplicationErrors' to true. If code was *correctly* duplicated, consistent with the intention of the plan, set 'hasDuplicationErrors' to false.

'comments':  an array of objects with two properties: 'txt' and 'reference'. 'txt' is the exact text of a code comment. 'reference' is a boolean that indicates whether the comment is a placeholder of or reference to the original code, like  "// rest of the function..." or "# existing init code...", or "// rest of the main function" or "// rest of your function..." or "// Existing methods..." or "// Remaining methods" or "// Existing code..." or "// ... existing setup code ..." or "// Existing code..." "// ... existing code ..." or "// ..." or other comments which reference code from the *original file*. References DO NOT need to exactly match any of the previous examples. Use your judgement to determine whether each comment is a reference. If 'reference' is true, the comment is a placeholder or reference to the original code. If 'reference' is false, the comment is not a placeholder or reference to the original code.

In 'comments', you must list EVERY comment included in the *updated file*. Only list *code comments* that are valid comments for the programming language being used. Do not list logging statements or any other non-comment text that is not a valid code comment. If there are no code comments in the *updated file*, 'comments' must be an empty array.

If there are multiple identical comments in the *updated file*, you MUST list them *all* in the 'comments' array--list each identical comment as a separate object in the array.

'referenceErrorsReasoning': A string that succinctly explains whether any comments in the updated file are placeholders/references that should have been replaced with code from the *original file*. These are comments like "// rest of the function..." or "# existing init code...", or "// rest of the main function..." or "// rest of your function..." or "// Existing methods...", "// Existing code..." or other  comments which reference code from the *original file*. Only include comments that *are not* present in the *original file* and *are* present in the *proposed updates*. If there are no such comments, explain that there are no reference errors.

'hasReferenceErrors': A boolean that indicates whether any comments in the updated file are placeholders/references that should be replaced with code from the *original file*, based on the reasoning provided in 'referenceErrorsReasoning'.

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
