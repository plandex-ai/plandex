package prompts

import (
	"fmt"
	"plandex-server/syntax"

	"github.com/plandex/plandex/shared"
)

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

	Before outputting the references, first output a section that lists *EVERY* comment in the *proposed updates*, including the line number of each comment prefixed by 'pdx-new-'. Below each comment, evaluate whether it is a reference comment. Focus on whether the comment is clearly referencing a block of code in the *original file*, whether it is explaining a change being made, or whether it is a comment that was carried over from the *original file* but does *not* reference any code that was left out of the *proposed updates*. After this evaluation, state whether each comment is a reference comment or not. Only list valid *comments* for the given programming language in the comments section. Do not include non-comment lines of code in the comments section.
	
	Example:

	---
	Comments:

	pdx-new-2: // ... existing code to start transaction ...
	Evaluation: refers the code at the beginning of the 'update' function that starts the database transaction.
	Reference: true

	pdx-new-6: // ... existing update code ...	
	Evaluation: refers the code inside the 'update' function that updates the user.
	Reference: true

	pdx-new-9: // ... existing code to commit db transaction...
	Evaluation: refers the code inside the 'update' function that commits the database transaction.
	Reference: true

	pdx-new-4: // verify user permission before performing update
	Evaluation: describes the change being made. Does not refer to any code in the *original file*.
	Reference: false

	pdx-new-85: // Rest of the main function...
	Evaluation: refers to the rest of the main function that is left unchanged.
	Reference: true

	pdx-new-25: # Delete the object
	Evaluation: describes the change being made. Does not refer to any code in the *original file*.
	Reference: false
`

const RefsOnlyEnding = `
	*

	Do NOT include any additional text after the <references> element. The output must end after </references>. DO NOT use the string <references> anywhere else in the output. ONLY use it to start the <references> element.
`

const WholeFileEnding = `
	*

	Now output the entire file with the *proposed updates* correctly applied. ALL identified references MUST be replaced by the appropriate code from the *original file*. You MUST correctly merge the code from the *original file* with the *proposed updates* and output the entire file. The resulting file MUST NOT include any reference comments.

	The resulting file MUST be syntactically and semantically correct. All code structures must be properly balanced.
	
	The full resulting file should be output within a <file> element, like this:

	<file>
		package main

		import "logger"

		function main() {
			logger.info("Hello, world!");
			exec()
		}
	</file>

	Do NOT include line numbers in the <file> element. Do NOT include reference comments in the <file> element. Output the ENTIRE file, no matter how long it is, with NO EXCEPTIONS. Include the resulting file *only* with no other text. Do NOT wrap the file output in triple backticks or any other formatting, except for the <file> element tags.

	Do NOT include any additional text after the <file> element. The output must end after </file>. DO NOT use the string <file> anywhere else in the output. ONLY use it to start the <file> element.
`

func GetValidateEditsPrompt(
	path,
	original,
	desc,
	proposed,
	diff string,
	reasons []syntax.NeedsVerifyReason,
) string {
	var s string

	s += fmt.Sprintf(`
[Explanation Rules]

%s

[End Explanation Rules]

[Proposed Updates Rules]

%s

[End Proposed Updates Rules]
`, ChangeExplanationPrompt, UpdateFormatPrompt)

	s += fmt.Sprintf("Path: %s\n\nOriginal file:\n%s\n\nProposed changes explanation:\n%s\n\nProposed changes:\n%s", path, original, desc, proposed)
	s += fmt.Sprintf("\n\nApplied changes:\n%s", diff)
	s += `You will examine the original file, proposed changes explanation, proposed changes, and applied changes diff.`

	var editsIncorrect bool

	for _, reason := range reasons {
		if reason == syntax.NeedsVerifyReasonAmbiguousLocation {
			editsIncorrect = true
			s += `The proposed changes were applied to an ambiguous location. This may mean that anchors were written incorrectly (with incorrect spacing or indentation, for example), or else that the ordering of anchors was incorrect. It may also mean that necessary surrounding context and anchors were not included.`
		} else if reason == syntax.NeedsVerifyReasonNoChanges {
			s += `The proposed changes did not cause any change to the resulting file when applied. Was this intentional?`
		} else if reason == syntax.NeedsVerifyReasonCodeRemoved {
			s += `The proposed changes removed or replaced code. Was this intentional?`
		} else if reason == syntax.NeedsVerifyReasonCodeDuplicated {
			s += `The proposed may have duplicated code. Was this intentional?`
		}
	}

	if editsIncorrect {
		s += EditsIncorrectPrompt
	} else {
		s += EditsValidatePrompt
	}

	s += getEditsExamples(!editsIncorrect)

	s += `

- You MUST NOT make *functional* changes when rewriting the explanation or proposed updates. Your job is not to change functionality or write additional code, but to ensure that the explanation and proposed updates follow all the rules correctly. Stick as close to the original proposed updates as possible when rewriting. Fix the problems, but keep everything else the same as much as possible, including comments, line breaks, spacing, etc.

- You MUST NOT add 'Plandex: removed' comments unless the intention of the plan and the explanation is to remove code. NEVER add 'Plandex: removed' comments to the *proposed updates* unless the *explanation* includes the word 'remove'. The *only* time you can add a 'Plandex: removed' comment is if 1 - you are correcting a mistaken comment that was intended as a removal comment, but used the wrong format—for example, if the 'Plandex: ' prefix was incorrectly omitted. In that case you must fix the comment in the fixed proposed updates. 2 - the explanation explicitly stats that code should be removed, but a necessary 'Plandex: removed' comment is missing from the proposed updates, causing the code that should have been removed to incorrectly remain in the file. In that case you must add the missing 'Plandex: removed' comment to the proposed updates in the correct location, with correct surrounding context and anchors.
`

	return s
}

const EditsValidatePrompt = `
- In a '## Evaluate Diff' section, examine the applied changes diff and determine whether the changes were applied correctly. Carefully asses whether:
  a. Any code was removed that should have been kept.
  b. Any code was duplicated that should not have been.
	c. Any code was incorrectly inserted or replaced.
	d. Any code was inserted at the wrong location.

- If the changes were applied correctly, output "**Changes Applied Correctly**" after the evaluation and stop your response here. You MUST NOT output any further explanation or *any other text* after the "**Changes Applied Correctly**" string. End your response *immediately* after the "**Changes Applied Correctly**" string.

- If the changes were *not* applied correctly, continue with the following steps:
` + EditsIncorrectPrompt

const EditsIncorrectPrompt = `
- In a '## Evaluate Explanation' section, examine the proposed changes explanation and determine whether it is correct. Did it follow all the rules in the [Proposed Updates Rules] section?
  a. If the changes are inserting, removing, or replacing code between two specific code structures, are these the correct code structures to use for the update? Are these structures *immediately* adjacent to each other or is there other code between them?
	b. If the changes are inserting, removing, or replacing code between a specific code structure and the start or end of the file, are these the correct code structures or locations to use for the update? If the change is between the start of the file and a specific code structure, is there any other code between the start of the file and that code structure? (There must not be). If the change is between a specific code structure and the end of the file, is there any other code between that code structure and the end of the file? (There must not be).	

- If the proposed changes explanation is *not* correct, you must rewrite the proposed changes explanation to correctly follow *all* the rules in the [Proposed Updates Rules] section in a <PlandexProposedUpdatesExplanation> element. Include *only* the rewritten proposed changes explanation inside the <PlandexProposedUpdatesExplanation> element, and no other text. If the proposed changes explanation is already correct, skip this step.
  - When rewriting the proposed changes explanation, DO NOT change the alter the *type* of the change. You must ONLY update the portion of the explanation that explains the *location* of the change, not the *type* of the change. For example, if the proposed changes explanation is "I'll add 'someFunction' at the end of the file, immediately after 'existingFunction1'", you must rewrite it to correctly explain the *location* of the change, like "I'll add 'someFunction' at the end of the file, immediately after 'existingFunction2'". You MUST NOT change the type of the change, such as from 'add' to 'remove'.
	- If the explanation does not match the changes that were made in the proposed updates (for example, the explanation says to add code, but the proposed updates removed code), this means that the proposed updates are *incorrect*—you MUST NOT alter the explanation to match the incorrect proposed updates (for example, you MUST NOT change the explanation to remove code instead of adding it just because the proposed updates incorrectly removed code). Instead, focus on ensuring that the explanation correctly shows the *location* of the change so that the proposed updates can be corrected.
	- If you are outputting a rewritten proposed changes explanation, YOU ABSOLUTELY MUST DO SO within the <PlandexProposedUpdatesExplanation> element. DO NOT output it outside of the <PlandexProposedUpdatesExplanation> element or anywhere else in your response—ONLY output the rewritten proposed changes explanation inside the <PlandexProposedUpdatesExplanation> element.

- In a '## Evaluate Proposed Changes' section, examine the proposed changes and determine whether they are correct. 
	a. Did they follow all the rules in the [Proposed Updates Rules] section? 
	b. Are they consistent with the proposed changes explanation (or the rewritten proposed changes explanation, if you had to rewrite it)?
	c. Is more surrounding context needed or are more *anchors* needed to resolve ambiguity in the proposed changes and *precisely* locate the change *unambiguously* in the original file?
	d. Are any *reference comments* or *removal comments* written incorrectly? All reference comments must be *exactly* '... existing code ...' and all removal comments must be *exactly* 'plandex: removed code'. Each must use the correct comment syntax for the language. If any comments are variations of those rather than the exact strings, they are incorrect.
	e. Are any *reference comments* or *removal comments* missing?
		- If the proposed changes incorrectly removed code, this does *NOT MEAN* that a removal comment is missing; you MUST NOT say that a removal comment is missing in this case. *ONLY* state that a removal comment is missing if the explanation *explicitly* states that code should be removed and it was not correctly removed due to a missing removal comment.

- In a <PlandexProposedUpdates> element, output the corrected proposed changes. Include *only* the corrected proposed changes inside the <PlandexProposedUpdates> element, and no other text. The corrected proposed changes MUST follow *all* the rules in the [Proposed Updates Rules] section.
	- If you are outputting a corrected proposed changes, YOU ABSOLUTELY MUST DO SO within the <PlandexProposedUpdates> element. DO NOT output it outside of the <PlandexProposedUpdates> element or anywhere else in your response—ONLY output the corrected proposed changes inside the <PlandexProposedUpdates> element. You ABSOLUTELY MUST NOT use triple backticks or any other formatting around the rewritten proposed changes. Output ONLY the corrected proposed changes inside the <PlandexProposedUpdates> element with no other text or formatting.
	- DO NOT include the proposed changes within a formatted code block. Include *ONLY* the code directly inside the <PlandexProposedUpdates> element.
	- If more surrounding context or *anchors* are needed to resolve ambiguity in the proposed changes and *precisely* locate the change *unambiguously* in the original file, you MUST output the corrected proposed changes with the additional context or *anchors* included correctly.
	- If any *reference comments* or *removal comments* are written incorrectly, you MUST output the corrected proposed changes with the *reference comments* or *removal comments* corrected. A reference comment must be *exactly* '... existing code ...' and a removal comment must be *exactly* 'plandex: removed code'. Use the correct comment syntax for the language.
	- If any *reference comments* or *removal comments* are missing, you MUST output the corrected proposed changes with the *reference comments* or *removal comments* added.
	- If the proposed changes incorrectly removed code, this does *NOT MEAN* that a removal comment is missing; you MUST NOT output a removal comment in this case. Instead, fix the proposed changes to prevent the removal of code that should have been kept.
	- When outputting *anchors* in the proposed changes, lines from the proposed changes that match lines in the original file, spacing and indentation must be preserved *exactly* as they are in the original file—the two lines must be *exactly identical*.
`

func getEditsExamples(evaluateDiff bool) string {
	s := ""

	if evaluateDiff {
		s += `
- Example Response if changes were applied correctly ('## Evaluate Diff' section included):

--

## Evaluate Diff

The new function 'someFunction' was correctly added to the end of the file.

**Changes Applied Correctly**
--

- Example Response if changes were *not* applied correctly, but explanation and proposed changes were correct:

--

## Evaluate Diff

The new function 'someFunction' was incorrectly added to the end of the file.

## Evaluate Explanation

The explanation correctly describes how the new function should be added to the file.

## Evaluate Proposed Changes

The proposed changes correctly use a reference comment to indicate that the new function should be added after 'existingFunction' at the end of the file.

--

Note that if the changes were applied correctly, YOU ABSOLUTLY MUST NEVER output a <PlandexProposedUpdates> element. Just stop the response after the ## Evaluate Proposed Changes section.
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
`
	}

	s += `
## Evaluate Explanation

The explanation incorrectly used 'existingFunction1' as an anchor when there is code between 'existingFunction1' and the end of the file. It should have used 'existingFunction2' instead.
`

	s += `
<PlandexProposedUpdatesExplanation>
 **Updating some/path/to/file.js:** I'll add 'someFunction' at the end of the file, immediately after 'existingFunction2'. 
</PlandexProposedUpdatesExplanation>

## Evaluate Proposed Changes

The proposed changes incorrectly used 'existingFunction1' as an anchor when there is code between 'existingFunction1' and the end of the file. It should have used 'existingFunction2' instead.

<PlandexProposedUpdates>
// ... existing code ...

function existingFunction2() {
  // ... existing code ...
}

function someFunction() {
  console.log("New behavior");
	const res = await execUpdate();
}
</PlandexProposedUpdates>

--
`

	return s
}

func getPreBuildStatePrompt(filePath, preBuildState string) string {
	if preBuildState == "" {
		return ""
	}

	return fmt.Sprintf("**The current file is %s. Original state of the file:**\n```\n%s\n```", filePath, preBuildState) + "\n\n"
}
