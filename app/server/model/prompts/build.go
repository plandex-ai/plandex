package prompts

import (
	"fmt"
	"plandex-server/syntax"
	"strings"

	"github.com/plandex/plandex/shared"
)

type ValidateEditsPromptParams struct {
	Path         string
	Original     string
	Desc         string
	Proposed     string
	Diff         string
	Reasons      []syntax.NeedsVerifyReason
	SyntaxErrors []string

	FullCorrection bool
}

func GetValidateEditsPrompt(params ValidateEditsPromptParams) string {
	path := params.Path
	original := params.Original
	desc := params.Desc
	proposed := params.Proposed
	diff := params.Diff
	reasons := params.Reasons
	fullCorrection := params.FullCorrection
	syntaxErrors := params.SyntaxErrors

	var s string

	// 	if fullCorrection {
	// 		s += fmt.Sprintf(`
	// [Explanation Rules]

	// %s

	// [End Explanation Rules]
	// `, ChangeExplanationPrompt)
	// 	}

	s += fmt.Sprintf("Path: %s\n\nOriginal file:\n%s\n\nProposed changes explanation:\n%s\n\nProposed changes:\n%s", path, original, desc, proposed)
	s += fmt.Sprintf("\n\nApplied changes:\n%s", diff)
	s += `You will examine the original file, proposed changes explanation, proposed changes, and applied changes diff.`

	var editsIncorrect bool

	for _, reason := range reasons {
		if reason == syntax.NeedsVerifyReasonAmbiguousLocation {
			editsIncorrect = true
			s += `The proposed changes were applied to an ambiguous location. This may mean that anchors were written incorrectly (with incorrect spacing or indentation, for example), or else that the ordering of anchors was incorrect. It may also mean that necessary surrounding context and anchors were not included.`
		} else if reason == syntax.NeedsVerifyReasonCodeRemoved {
			s += `The proposed changes removed or replaced code. Was this intentional? If this was the clear intention of the plan, then removing or replacing code can be correct. But it could also be a sign that the proposed changes were not correctly applied.`
		} else if reason == syntax.NeedsVerifyReasonCodeDuplicated {
			s += `The proposed may have duplicated code. Was this intentional? If this was the clear intention of the plan, or the duplication exists in the original file for a valid reason, then duplicating code can be correct. But it could also be a sign that the proposed changes were not correctly applied.`
		}
	}

	if len(syntaxErrors) > 0 {
		editsIncorrect = true
		s += `The proposed changes, when applied, resulted in a file with syntax errors, meaning the proposed changes were either written incorrectly or were not correctly applied. The resulting file has the following syntax errors:\n\n` + strings.Join(syntaxErrors, "\n") + `\n\n`
	}

	if fullCorrection {
		if !editsIncorrect {
			s += EditsValidateFullPrompt
		}

		// s += EditsExplanationPrompt
		s += ReferencesPrompt
		s += EditsFindReplacePrompt
	}

	if fullCorrection {
		s += getFullCorrectionExamples(!editsIncorrect)
	} else {
		s += EditsValidateOnlyPrompt
		s += ValidateOnlyExample
	}

	return s
}

const EditsValidateBasePrompt = `
Your task is to examine the applied changes diff and output EXACTLY ONE of these patterns:

Your evaluation should assess:
a. Whether any code was removed that should have been kept
b. Whether any code was duplicated that should not have been
c. Whether any code was incorrectly inserted or replaced
d. Whether any code was inserted at the wrong location

Be succinct—do not add additional text or summarize the changes beyond what is necessary.
`

const EditsValidateCorrectPattern = `
Pattern 1 - If changes were applied correctly:
## Evaluate Diff
[Your evaluation]
<PlandexCorrect/>
<PlandexFinish/>
`

const EditsValidateFullIncorrectPattern = `
Pattern 2 - If changes were NOT applied correctly:
## Evaluate Diff
[Your evaluation]
<PlandexIncorrect/>
[Continue to next section for full correction]
`

const EditsValidateOnlyIncorrectPattern = `
Pattern 2 - If changes were NOT applied correctly:
## Evaluate Diff
[Your evaluation]
<PlandexIncorrect/>
<PlandexFinish/>
`

const EditsValidateFullPrompt = EditsValidateBasePrompt + EditsValidateCorrectPattern + EditsValidateFullIncorrectPattern

const EditsValidateOnlyPrompt = EditsValidateBasePrompt + EditsValidateCorrectPattern + EditsValidateOnlyIncorrectPattern

// const EditsExplanationPrompt = `
// - Next, in an '## Evaluate Explanation' section, examine the proposed changes explanation and determine whether it is correct. Did it follow all the rules in the [Explanation Rules] section?

// 	a. If the changes are inserting, removing, or replacing code between two specific code structures, are these the correct code structures to use for the update? Are these structures *immediately* adjacent to each other or is there other code between them?

// 	b. If the changes are inserting, removing, or replacing code between a specific code structure and the start or end of the file, are these the correct code structures or locations to use for the update? If the change is between the start of the file and a specific code structure, is there any other code between the start of the file and that code structure? (There must not be). If the change is between a specific code structure and the end of the file, is there any other code between that code structure and the end of the file? (There must not be).

// - If the proposed changes explanation is *not* correct, you must rewrite the proposed changes explanation to correctly follow *all* the rules in the [Explanation Rules] section in a <PlandexProposedUpdatesExplanation> element. Include *only* the rewritten proposed changes explanation inside the <PlandexProposedUpdatesExplanation> element, and no other text. If the proposed changes explanation is already correct, skip this step.

//   - When rewriting the proposed changes explanation, DO NOT change the *type* of the change. You must ONLY update the portion of the explanation that explains the *location* of the change, not the *type* of the change. For example, if the proposed changes explanation is "I'll add 'someFunction' at the end of the file, immediately after 'existingFunction1'", you must rewrite it to correctly explain the *location* of the change, like "I'll add 'someFunction' at the end of the file, immediately after 'existingFunction2'". You MUST NOT change the type of the change, such as from 'add' to 'remove'.

// 	- If the explanation does not match the changes that were made in the proposed updates (for example, the explanation says to add code, but the proposed updates removed code), this means that the proposed updates are *incorrect*—you MUST NOT alter the explanation to match the incorrect proposed updates (for example, you MUST NOT change the explanation to remove code instead of adding it just because the proposed updates incorrectly removed code). Instead, focus on ensuring that the explanation correctly shows the *location* of the change so that a *correct* find/replace operation can be generated.

// 	- You MUST NOT make *functional* changes when rewriting the explanation. Your job is *NOT* to change functionality or add additional code in *any way*. It is *only* to ensure that the explanation follows all the rules correctly. Stick as close to the original explanation as possible when rewriting. Fix the problems, but keep everything else *exactly the same*.
// `

const EditsFindReplacePrompt = `
- Next, convert the proposed updates into a find/replace operation. You MUST output EXACTLY ONE <PlandexReplacement> element. Multiple replacements are NOT allowed under any circumstances.

- The SINGLE <PlandexReplacement> element MUST contain two child elements:
	- A <PlandexOld> element that contains the *EXACT* text from the *original file* that will be replaced.
	- A <PlandexNew> element that contains the *EXACT* text that will replace it.

CRITICAL RULES FOR REPLACEMENTS:
1. You MUST output EXACTLY ONE <PlandexReplacement> element
2. If multiple changes are needed, you MUST combine them into a single replacement operation
3. The <PlandexOld> element MUST match a UNIQUE block of text in the original file
4. You MUST NEVER output multiple <PlandexReplacement> elements

For multiple changes, include ALL changes in the SAME <PlandexReplacement> element:

Example of CORRECT way to handle multiple changes:

<PlandexReplacement>
	<PlandexOld>
		function existingFunction1() {
			someFunction();
			someOtherFunction();
		}

		function existingFunction2() {
			anotherFunction();
		}
	</PlandexOld>
	<PlandexNew>
		function existingFunction1() {
			someFunction(nil);
			someOtherFunction(nil);
			yetAnotherFunction(10);
		}

		function existingFunction2() {
			anotherFunction(true);
			newFunction();
		}
	</PlandexNew>
</PlandexReplacement>

Example of INCORRECT way (DO NOT DO THIS):

❌ WRONG - Multiple replacement tags are NOT allowed:

<PlandexReplacement>
	<PlandexOld>function1...</PlandexOld>
	<PlandexNew>function1...</PlandexNew>
</PlandexReplacement>

<PlandexReplacement>
	<PlandexOld>function2...</PlandexOld>
	<PlandexNew>function2...</PlandexNew>
</PlandexReplacement>

Additional Requirements:
- The <PlandexOld> text MUST be *EXACTLY* the same as the text in the *original file* and MUST be *UNIQUE*
- Make sure sufficient context is included to match uniquely and unambiguously
- Newlines and special characters MUST also be *EXACTLY* matched
- You MUST NOT include any placeholders or reference comments
- Both <PlandexOld> and <PlandexNew> must consist of *entire* lines of code *only*
- Do NOT find and replace partial lines of code
- The smallest unit of code that can be replaced is an entire line
`

const ValidateOnlyExample = `
Here are examples showing ALL possible valid response patterns:

Example 1 - Changes Applied Correctly:

## Evaluate Diff
The new function 'someFunction' was correctly added to the end of the file, with proper indentation and spacing.

<PlandexCorrect/>
<PlandexFinish/>

Example 2 - Simple Error Case:

## Evaluate Diff
The new function 'someFunction' was incorrectly added to the end of the file - it was inserted with wrong indentation.

<PlandexIncorrect/>
<PlandexFinish/>

Example 3 - Duplicated Code Case:

## Evaluate Diff
The changes introduced duplicate code - the 'initializeConfig' function now appears twice in the file, when it should have been replaced.

<PlandexIncorrect/>
<PlandexFinish/>

Example 4 - Wrong Location Case:

## Evaluate Diff
The new validation checks were added to the wrong function. They were inserted into 'validateInput' when they should have been added to 'validateOutput'.

<PlandexIncorrect/>
<PlandexFinish/>

Example 5 - Missing Code Case:

## Evaluate Diff
The changes accidentally removed the error handling code that should have been preserved.

<PlandexIncorrect/>
<PlandexFinish/>

IMPORTANT RULES:
1. If your evaluation finds ANY issues, use Example 2-5 format and STOP.
2. If your evaluation finds NO issues, you MUST use Example 1's format EXACTLY and STOP.
3. You MUST never mix these formats or add additional sections when changes are correct.
4. The response MUST start with '## Evaluate Diff' in all cases.
5. Every response MUST end with either <PlandexCorrect/> followed by <PlandexFinish/>, or with <PlandexIncorrect/> followed by <PlandexFinish/>.
`

func getFullCorrectionExamples(evaluateDiff bool) string {
	s := ""

	if evaluateDiff {
		s += `
- Example Response if changes were applied correctly ('## Evaluate Diff' section included):

--

## Evaluate Diff

The new function 'someFunction' was correctly added to the end of the file.

<PlandexCorrect/>
<PlandexFinish/>

--

- Additional Examples of Correct Changes:

Example 1 - Simple Addition:

## Evaluate Diff
The new error handling function was correctly added with proper error types and return values.

<PlandexCorrect/>
<PlandexFinish/>

Example 2 - Multiple Changes:

## Evaluate Diff
All changes were applied correctly:
1. New validation function was added with correct parameter types
2. Error messages were updated with the requested format
3. Return types were modified as specified

<PlandexCorrect/>
<PlandexFinish/>

Example 3 - File Overwrite:

## Evaluate Diff
The file was correctly replaced with the new implementation, maintaining all required interfaces and types.

<PlandexCorrect/>
<PlandexFinish/>

--

- Example Response if changes were *not* applied correctly, but explanation and proposed changes were correct:

--

## Evaluate Diff

The new function 'someFunction' was incorrectly added to the end of the file. It should have been added at the start of the 'existingFunction1' function.

<PlandexIncorrect/>

## Comments

// ... existing code ...
Evaluation: refers to the initialization code at the start of the 'existingFunction1' function
Reference: true

// loop 15 times
Evaluation: explains the for loop below the comment
Reference: false

// ... existing function calls ...
Evaluation: refers to the function calls at the start of the for loop
Reference: true

<PlandexReplacement>
	<PlandexOld>
		function existingFunction1() {
			prepare();
			init();
			fullInit();
			// loop 10 times
			for (let i = 0; i < 10; i++) {
				originalFunction();
				someOtherFunction();
			}
		}
	</PlandexOld>
	<PlandexNew>
		function existingFunction1() {
			prepare();
			init();
			fullInit();
			// loop 15 times
			for (let i = 0; i < 15; i++) {
				originalFunction();
				someOtherFunction();
				someFunction();
				andAnotherFunction();
				andOneMoreFunction();
			}
		}
	</PlandexNew>
</PlandexReplacement>
`
	}

	s += `
- Example Response if explanation and proposed changes were *not* correct:

--
`

	if evaluateDiff {
		s += `
## Evaluate Diff

The proposed changes incorrectly removed the 'existingFunction2' function.

<PlandexIncorrect/>
`
	}

	// 	s += `
	// ## Evaluate Explanation

	// The explanation incorrectly used 'existingFunction1' as an anchor when there is code between 'existingFunction1' and the end of the file. It should have used 'existingFunction2' instead.

	// <PlandexProposedUpdatesExplanation>
	//  **Updating some/path/to/file.js:** I'll add 'someFunction' at the end of the file, immediately after 'existingFunction2'.
	// </PlandexProposedUpdatesExplanation>
	// `

	s += `
<PlandexReplacement>
	<PlandexOld>
		function existingFunction1() {
			someOtherFunction();
		}
	</PlandexOld>
	<PlandexNew>
		function existingFunction1() {
			someFunction();
			someOtherFunction();
		}
	</PlandexNew>
</PlandexReplacement>
`

	if evaluateDiff {
		s += `
FINAL REMINDER: The response pattern is strictly binary:
1. If changes are correct: ONLY output '## Evaluate Diff' with "<PlandexCorrect/>" + <PlandexFinish/>
2. If changes have ANY issues: Output '## Evaluate Diff' with "<PlandexIncorrect/>" and then continue to the next section ('## Evaluate Explanation').
There are NO other valid response patterns.`
	}

	return s
}

func getPreBuildStatePrompt(filePath, preBuildState string) string {
	if preBuildState == "" {
		return ""
	}

	return fmt.Sprintf("**The current file is %s. Original state of the file:**\n```\n%s\n```", filePath, preBuildState) + "\n\n"
}

func GetWholeFilePrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	s := WholeFileBeginning + ReferencesPrompt + WholeFileEnding + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesWithLineNums)

	return s
}

var ExampleReferences = `
A reference comment is a comment that references code in the *original file* for the purpose of making it clear where a change should be applied. Examples of reference comments include:

	- // ... existing code...
	- # Existing code...
	- /* ... */
	- // Rest of the function...
	- <!-- rest of div tag -->
	- // ... rest of function ...
	- // rest of component...
	- # other methods...
	- // ... rest of init code...
	- // rest of the class...
	- // other properties
	- // other methods
	// ... existing properties ...
	// ... existing values ...
	// ... existing text ...

Reference comments often won't exactly match one of the above examples, but they will always be referencing a block of code from the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.

Reference comments do NOT need to be valid comments for the given file type. For file types like JSON or plain text that do not use comments, reference comments in the form of '// ... existing properties ...' or '// ... existing values ...' or '// ... existing text ...' can still be present. These MUST be treated as valid reference comments regardless of the file type or the validity of the syntax.
`

const RefsBeginning = `
You are an AI that analyzes an *original file* and *proposed updates* to that file and then identifies *all* *reference comments* present in the *proposed updates*.

`

const WholeFileBeginning = `
After identifying all references, you will output the *entire file* with the *proposed updates* correctly applied. ALL references will be replaced by the appropriate code from the *original file*. You will correctly merge the code from the *original file* with the *proposed updates* and output the entire file.

`

var ReferencesPrompt = ExampleReferences + `
	
	*NOT EVERY COMMENT IS A REFERENCE.* If a comment refers to code that is present in the *proposed updates* then it is *not* a reference. Similarly, if a comment explains something about the change being made in the *proposed updates*, it is also *not* a reference.

	A reference comment MUST EXIST in the *proposed updates*. Do not include a reference comment unless it exists VERBATIM in the *proposed updates*.

	Output a section that lists *EVERY* comment in the *proposed updates*, including the line number of each comment prefixed by 'pdx-new-'. Below each comment, evaluate whether it is a reference comment. Focus on whether the comment is clearly referencing a block of code in the *original file*, whether it is explaining a change being made, or whether it is a comment that was carried over from the *original file* but does *not* reference any code that was left out of the *proposed updates*. After this evaluation, state whether each comment is a reference comment or not. Only list valid *comments* for the given programming language in the comments section. Do not include non-comment lines of code in the comments section.
	
	Example:

	---
	Comments:

	// ... existing code to start transaction ...
	Evaluation: refers the code at the beginning of the 'update' function that starts the database transaction.
	Reference: true

	// verify user permission before performing update
	Evaluation: describes the change being made. Does not refer to any code in the *original file*.
	Reference: false

	// ... existing update code ...	
	Evaluation: refers the code inside the 'update' function that updates the user.
	Reference: true

	--

	If there are no comments in the *proposed updates*, output just the string 'No comments' and continue.
`

const WholeFileEnding = `
	*

	Now output the entire file with the *proposed updates* correctly applied. ALL identified references MUST be replaced by the appropriate code from the *original file*. You MUST correctly merge the code from the *original file* with the *proposed updates* and output the *entire* resulting file. The resulting file MUST NOT include any reference comments.

	The resulting file MUST be syntactically and semantically correct. All code structures must be properly balanced.
	
	The full resulting file should be output within a <PlandexWholeFile> element, like this:

	<PlandexWholeFile>
		package main

		import "logger"

		function main() {
			logger.info("Hello, world!");
			exec()
		}
	</PlandexWholeFile>

	Do NOT include line numbers in the <PlandexWholeFile> element. Do NOT include reference comments in the <PlandexWholeFile> element. Output the ENTIRE file, no matter how long it is, with NO EXCEPTIONS. Include the resulting file *only* with no other text. Do NOT wrap the file output in triple backticks or any other formatting, except for the <PlandexWholeFile> element tags.

	Do NOT include any additional text after the <PlandexWholeFile> element. The output must end after </PlandexWholeFile>. DO NOT use the string <PlandexWholeFile> anywhere else in the output. ONLY use it to start the <PlandexWholeFile> element.
`
