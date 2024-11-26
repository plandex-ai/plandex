package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
)

func GetSemanticAnchorsPrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	s := SemanticAnchorsPrompt + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesWithLineNums)

	// fmt.Printf("SemanticAnchorsPrompt: %s\n", s)

	return s
}

func GetReferencesPrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	s := RefsBeginning + ReferencesPrompt + RefsOnlyEnding + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesWithLineNums)

	return s
}

func GetWholeFilePrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	s := WholeFileBeginning + ReferencesPrompt + WholeFileEnding + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesWithLineNums)

	return s
}

var SemanticAnchorsPrompt = `
You are an AI that analyzes an *original file* and *proposed updates* to that file and then identifies all *semantic anchors* and *reference comments* present in the *proposed updates*.

- 

### Semantic Anchors

A semantic anchor is a line in the *proposed updates* that is not exactly equal to a line in the *original file* but is nonetheless intended to match a line in the *original file*.

For example, if the *original file* has:

	pdx-1: function update(name string, id string) {
	pdx-2:   const tx = await client.startTransaction();
	pdx-3:   tx.setMode(TransactionMode.Serializable);
	pdx-4:   const updateQuery = 'UPDATE users SET name = $1 WHERE id = $2';
	pdx-5:   await tx.execute(updateQuery, name, id);
	pdx-6:   await client.commit();
	pdx-7: }

And the *proposed updates* have:

	pdx-new-1: function update(name string, id string, log bool) {
	pdx-new-2:   // ... existing code ...
	pdx-new-3:   if (log) {
	pdx-new-4:     console.log("Updating user");
	pdx-new-5:   }
	pdx-new-6: }

Then the line 'pdx-new-1: function update(name string, id string, log bool) {' is a semantic anchor since it is not exactly equal to a line in the *original file* (due to the addition of the 'log' parameter) but is clearly intended to match the line 'pdx-1: function update(name string, id string) {' in the *original file*.

The line 'pdx-new-6: }' is *not* a semantic anchor since it is exactly equal to a line in the *original file*.

Comments that are modified in the *proposed updates* (but are clearly still referring to the same comment in the *original file*) can also be semantic anchors.

A line that is exactly equal (including whitespace) to a line in the *original file* MUST NEVER UNDER ANY CONCEIVABLE CIRCUMSTANCES be marked as a semantic anchor.

For example, if the *original file* has:

pdx-1: function main() {
pdx-2:    // Get response
pdx-3:    const response = await getResponse();
pdx-4:    return response;
pdx-5: }

And the *proposed updates* have:

pdx-new-1: function main() {
pdx-new-2:    // Get response and parse JSON body
pdx-new-3:    let response = await getResponse();
pdx-new-4:    response = jsonResponse(response);
pdx-new-5:    return response;
pdx-new-6: }

Then both the lines 'pdx-new-2:    // Get response and parse JSON body' and 'pdx-new-3:    let response = await getResponse();' are semantic anchors since they are not exactly equal to a line in the *original file* but are clearly intended to match the lines 'pdx-2:    // Get response' and 'pdx-3:    const response = await getResponse();' in the *original file*. The line 'pdx-new-4:    response = jsonResponse(response);' is *not* a semantic anchor since it doesn't refer to any code in the *original file*. The line 'pdx-new-6:    return response;' is *not* a semantic anchor since it is exactly equal to a line in the *original file*.

To mark a line a semantic anchor, it must be very clear from the description of the change and the *proposed updates* that a line from the *proposed updates* is intended to match a line in the *original file*. Do NOT mark lines as semantic anchors simply because they are similar to a line in the *original file*. The line must be clearly intended to match a line in the *original file*. If it's ambiguous whether a line is intended to match a line in the *original file*, it is *not* a semantic anchor.

If a line in the *proposed updates* is identical to a line in the *original file*, but has changed position or is being used in a different context, it is *still NOT* a semantic anchor. You *MUST NOT* under any circumstances mark any line that is identical to a line in the *original file* as a semantic anchor. Semantic anchors are only for lines that have *changed* in the *proposed updates* but are still clearly intended to match a line from the *original file*.

If a line in the *proposed updates* is a semantic anchor and there is a comment (or multiple comments) associated with the line that is being modified, carefully consider if the comment should also be marked as a semantic anchor. If the comment or comments clearly map to a corresponding comment in the *original file*, and the comment is modified in the *proposed updates*, then it MUST be marked as a semantic anchor. Correctly marking comments as sematic anchors is just as important as marking other lines of code as semantic anchors.

- 

### Reference Comments

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

For some file types that don't use comments like JSON or plain text, reference comments in the form of '// ... existing properties ...' or '// ... existing values ...' or '// ... existing text ...' can still be present. These MUST be treated as valid reference comments regardless of the validity of the comment syntax.

-

### Removal Comments

A removal comment is a comment that indicates that a section of code should be removed from the *original file*. Examples of removal comments include:

	- // removed code
	- # removed start of function
	- <!-- removed 'container' div -->
	- // remove 'checkMigration' method

Removal comments often won't exactly match one of the above examples, but they will always be indicating that a section of code should be removed from the *original file*.

For some file types that don't use comments like JSON or plain text, removal comments like '// removed keys ...' can still be present. These MUST be treated as valid removal comments regardless of the validity of the comment syntax.

- 

### Output

**First,** output a single *brief* paragraph of general reasoning about the *proposed updates* and how they refer to the *original file*, focusing on the structure of each, which elements are changing, and how the changes map to the *original file*. Also make a brief note of any comments that are being modified or introduced in the *proposed updates* and whether/how they map to comments in the *original file*. Do NOT output a list of semantic anchors in this paragraph—save that for the next section.

**Next,** output a <PlandexSummary> element that gives a very brief, one-sentence summary of the changes being made. Example:

<PlandexSummary>
	Adds a 'log' parameter to the 'update' function and modifies the 'update' function to use it.
</PlandexSummary>

**Next,** output a section that lists *EVERY* comment in the *proposed updates*, including the line number of each comment prefixed by 'pdx-new-'. Below each comment, evaluate whether it is a *reference comment*, a *removal comment*, or neither.

*To determined whether a comment is a reference comment*, focus on whether the comment is clearly referencing a block of code in the *original file*, whether it is explaining a change being made, or whether it is a comment that was carried over from the *original file* but does *not* reference any code that was left out of the *proposed updates*.

*To determine whether a comment is a removal comment*, focus on whether the comment is clearly indicating that a section of code should be removed from the *original file*.

After this evaluation, state whether each comment is a *reference comment*, a *removal comment*, or neither. Only list valid *comments* for the given programming language in the comments section. Do not include non-comment lines of code in the comments section. The only exception to this rule is for file types that don't use comments like JSON or plain text—for these treat lines beginning with '//' as reference comments. A *removal comment* *MUST NOT* also be marked as a *reference comment* and a *reference comment* *MUST NOT* also be marked as a *removal comment*.

Reference comments example:

---
Comments:

pdx-new-2: // ... existing code to start transaction ...
Evaluation: refers the code at the beginning of the 'update' function that starts the database transaction.
Reference: true
Removal: false

pdx-new-6: // ... existing update code ...	
Evaluation: refers the code inside the 'update' function that updates the user.
Reference: true
Removal: false

pdx-new-9: // Removed 'checkMigration' method
Evaluation: refers the 'checkMigration' method that is being removed.
Reference: false
Removal: true

pdx-new-15: // verify user permission before performing update
Evaluation: describes the change being made. Does not refer to any code in the *original file*.
Reference: false
Removal: false

pdx-new-85: // Rest of the main function...
Evaluation: refers to the rest of the main function that is left unchanged.
Reference: true
Removal: false

pdx-new-25: # removed rest of init code
Evaluation: indicates that the rest of the init code should be removed from the *original file*.
Reference: false
Removal: true
---

**Next,** for each reference (if there are any), output valid xml with this structure:

<PlandexReferences>
	<Reference
		comment="// ... rest of function ..."
		proposedLine="pdx-new-10"
	/>
</PlandexReferences>

If there are no reference comments, output an empty <PlandexReferences> element.

**Next,** for each removal (if there are any), output valid xml with this structure:

<PlandexRemovals>
	<Removal
		comment="# removed rest of init code"
		proposedLine="pdx-new-25"
	/>
</PlandexRemovals>

If there are no removal comments, output an empty <PlandexRemovals> element.

**Last,** output xml with this structure:

<PlandexSemanticAnchors>
	<Anchor
		reasoning="'update function' signature with 'log' parameter added"
		proposedLine="pdx-new-1"		
		originalLine="pdx-1"
	/>
</PlandexSemanticAnchors>

Explanation of 'Anchor' tag attributes:

	**reasoning**: A brief explanation of why this line in the *proposed updates* is a *semantic anchor*—why it's intended to match a line in the *original file* (even though it's *not* exactly equal to that line).

	**proposedLine**: The line number, prefixed by 'pdx-new-', in the *proposed updates* that is intended to match a line in the *original file*. MUST be a line that exists in the *proposed updates*.	

	**originalLine**: The line number, prefixed by 'pdx-', in the *original file* that the anchor is referring to. MUST be a line that exists in the *original file* and MUST ALWAYS be prefixed by 'pdx-' (never pdx-new-).

If there are no semantic anchors, output an empty <PlandexSemanticAnchors> element. Do NOT invent semantic anchors if there are none. It's common for there to be no semantic anchors in the *proposed updates*. In that case, output an empty <PlandexSemanticAnchors> element.

A *reference comment* or *removal comment* *MUST NOT* also be marked a semantic anchor. You MUST NEVER WITHOUT EXCEPTION mark *reference comments* or *removal comments* as semantic anchors.

Every response must include a single <PlandexSummary> element and a single <PlandexSemanticAnchors> element.

Do NOT include any other text or comments in your output.
`

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

	---

	Next, for each reference (if there are any), output valid xml with this structure:

	<references>
		<reference 
			comment="// ... existing code..." 
			description="Code after the addition of the conditional statement in the 'update' function" 
			proposedLine="pdx-new-10"
			structure="'update' function"
			structureOpens="opening '{' on pdx-455"
			structureCloses="closing '}' on pdx-461"
			originalStart="pdx-456" 
			originalEnd="pdx-460" 
		/>
	</references>

	Explanation of 'reference' tag attributes:

		**comment**: The comment that includes the reference.

		**description**: A brief description of the reference. Make a special note of where the reference *begins* and *ends in the *original file* so that it does *not* include any lines that are already present in the *proposed updates*.

		**proposedLine**: The line number, prefixed by 'pdx-new-', in the *proposed updates* that the reference is referring to. MUST be a line that exists in the *proposed updates* and contains the reference comment in 'comment' attribute.

		**structure**: The *code structure* (e.g. 'function', 'class', 'loop', 'conditional', etc.) that this reference is contained within. If it's not contained within a code structure and is instead at the top level of the file, output 'top level'. This must be the MOST specific, deeply nested code structure that contains the reference. You must output only a single structure or 'top level'. Identify the structure unambiguously.

		**structureOpens**: The entire line from the *original file*, prefixed by 'pdx-', that contains the opening symbol of the code structure identified in the 'structure' property. If the structure is 'top level', this MUST be an empty string. Otherwise, it MUST be a line that exists in the *original file* and MUST ALWAYS be prefixed by 'pdx-' (never pdx-new-).

		**structureCloses**: The entire line from the *original file*, prefixed by 'pdx-', that contains the closing symbol of the code structure identified in the 'structure' property. If the structure is 'top level', this MUST be an empty string. Otherwise, it MUST be a line that exists in the *original file* and MUST ALWAYS be prefixed by 'pdx-' (never pdx-new-).

		**originalStart**: The starting line number, prefixed by 'pdx-', in the *original file* that the reference is referring to. MUST be less than or equal to the 'originalEnd' line number and MUST be greater than or equal to 1. MUST be a line that exists in the *original file* and MUST ALWAYS be prefixed by 'pdx-' (never pdx-new-).

		**originalEnd**: The ending line number, prefixed by 'pdx-', in the *original file* that the reference is referring to. If the referenced code in the *original file* is a single line, the originalStart and originalEnd must be the same. 'originalEnd' MUST be greater than or equal to 'originalStart' and MUST be less than or equal to the last line number in the *original file*. MUST be a line that exists in the *original file* and MUST ALWAYS be prefixed by 'pdx-' (never pdx-new-).

  **Example:**

	---

	**Original file:**

	` + "```" + `
	pdx-1: function update(name string, id string) {
	pdx-2:   const tx = await client.startTransaction();
	pdx-3:   tx.setMode(TransactionMode.Serializable);
	pdx-4:     
	pdx-5:   const updateQuery = 'UPDATE users SET name = $1 WHERE id = $2';
	pdx-6:   await tx.execute(updateQuery, name, id);
	pdx-7: 
	pdx-8:   try {
	pdx-9: 	  await client.commit();
	pdx-10:   } catch (error) {
	pdx-11: 	  if (isRetryableError(error)) {
	pdx-12: 		  await client.rollback();
	pdx-13: 		  update();
	pdx-14: 	  } else {
	pdx-15: 	  	throw error;
	pdx-16: 	  }
	pdx-17:   }
	pdx-18: }
	` + "```" + `

	**Proposed changes:**

	Now we'll add a permission check so that the update only runs if the user has the necessary permissions.

	` + "```" + `
	pdx-new-1: function update(name string, id string) {
	pdx-new-2: 	// ... existing code to start transaction ...
	pdx-new-3:     
	pdx-new-4: 	// verify user permission before performing update
	pdx-new-5: 	if (canUpdate()) {
	pdx-new-6: 		// ... existing update code ...
	pdx-new-7: 	}
	pdx-new-8: 
	pdx-new-9: 	// ... existing code to commit db transaction...
	pdx-new-10: }
	` + "```" + `

	References:

	<references>
		<reference 
			comment="// ... existing code to start transaction ..."
			description="Code for starting database transaction"
			proposedLine="pdx-new-2"
			structure="'update' function"
			structureOpens="opening '{' on pdx-1"
			structureCloses="closing '}' on pdx-18"
			originalStart="pdx-2"
			originalEnd="pdx-3"
		/>
		<reference 
			comment="// ... existing update code ..."
			description="Code inside the canUpdate() conditional for updating the user"
			proposedLine="pdx-new-6"
			structure="'update' function"
			structureOpens="opening '{' on pdx-1"
			structureCloses="closing '}' on pdx-18"
			originalStart="pdx-5"
			originalEnd="pdx-6"
		/>
		<reference 
			comment="// ... existing code to commit db transaction..."
			description="Code for committing the database transaction and handling errors"
			proposedLine="pdx-new-9"
			structure="'update' function"
			structureOpens="opening '{' on pdx-1"
			structureCloses="closing '}' on pdx-18"
			originalStart="pdx-8"
			originalEnd="pdx-17"
		/>
	</references>

	---

	Note that in the example, the "// verify user permission before performing update" comment is *not* a reference since it refers to code that is included in the *proposed updates* and does *not* refer to code in the *original file* that was left out of the *proposed updates*.

	In description attributes, make a special note of where the reference *begins* and *ends in the *original file* so that it does *not* include any lines that are already present in the *proposed updates*. Note that in the above example, the function signature and opening brace of the 'update' function is *not included* in the reference since it's already present in the *proposed updates*. And node in the last reference, the closing brace of the 'update' function is *not included* since it's already present in the *proposed updates*.

	If there are no clear references in the *proposed updates*, output an empty <references> element. The *proposed updates* often will not inlcude any reference comments. In that case, output an empty <references> element. Do NOT include reference comments that are not present in the *proposed updates* or comments that are not clearly references to a block of code in the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.

	*

	You MUST carefully consider code structures when determining the 'originalStart' and 'originalEnd' lines for a reference. If a reference looks like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   constructor() {
	pdx-new-3:     this.initialize();
	pdx-new-4:   }
	pdx-new-5: 
	pdx-new-6:   // ... existing code ...
	pdx-new-7: }

	The the reference ONLY refers to the code within the 'MyClass' class. If there is additional code before or after the 'MyClass' class, it must not be included. The same applies to other code structures, like functions, loops, conditionals, etc.

	*
	
	When setting the 'originalStart' and 'originalEnd' lines for a reference, it is critically important that the code referenced in the *original file* falls COMPLETELY *within* the code structure specified in the 'structure' attribute. Do NOT include the lines that open or close the code structure in the 'originalStart' and 'originalEnd' lines, since these already exist in the *proposed updates*. Only the lines that *do not already exist* in the *proposed updates* can be included in the reference.

	If a reference comes at the beginning of a structure, followed by new or changed code, like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   // ... existing code ...
	pdx-new-3:   
	pdx-new-4:   function update() {
	pdx-new-5:     const conn = await getConnection();
	pdx-new-6:     const res = await execUpdate(conn);
	pdx-new-7:   }
	pdx-new-8: }

	Then the '// ... existing code ...' reference must include ALL code from the *original file* inside the 'MyClass' structure. The 'originalStart' must be one line *after* the 'structureOpens' line and the 'originalEnd' must be one line *before* the 'structureCloses' line.

	*
	
	Similarly, if a reference comes at the end of a structure, preceded by new or changed code, like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   function update() {
	pdx-new-3:     const conn = await getConnection();
	pdx-new-4:     const res = await execUpdate(conn);
	pdx-new-5:   }
	pdx-new-6:   // ... existing code ...
	pdx-new-7: }

	Then the '// ... existing code ...' reference must include ALL code from the *original file* inside the 'MyClass' structure. The 'originalStart' must be one line *after* the 'structureOpens' line and the 'originalEnd' must be one line *before* the 'structureCloses' line.

	*

	If a structure has references at both the beginning and the end, with new or changed code in between, like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   // ... existing code ...
	pdx-new-3:   
	pdx-new-4:   function update() {
	pdx-new-5:     const conn = await getConnection();
	pdx-new-6:     const res = await execUpdate(conn);
	pdx-new-7:   }
	pdx-new-8: 
	pdx-new-9:   // ... existing code ...
	pdx-new-10: }

	Then you must use your judgement to determine the location that the new code should be inserted in the final results. Based on where the new code is inserted, the first "pdx-new-2: // ... existing code ..." reference must include ALL code from the *original file* inside the 'MyClass' structure that should come *before* the new code. The 'originalStart' of the first reference must be one line *after* the 'structureOpens' line.
	
	The last "pdx-new-9: // ... existing code ..." reference must include ALL code from the *original file* inside the 'MyClass' structure that should come *after* the new code. The 'originalEnd' of the last reference must be one line *before* the 'structureCloses' line.
	
	In cases like the above example with multiple references within the same structure, you MUST NOT duplicate code in any of the references. Taken together, the references must cover the entire 'MyClass' structure with no code from the *original file* repeated or left out.

  *

	If there are commented out lines that still logically belong to a section that is referenced, you MUST include those lines in the reference.

	You MUST ensure that each reference includes the full logical section that it describes, including any comments that are part of that section. For example, if the description of a reference is "Imports", the reference must include *all* import statements in the section up to the next line that is included in the *proposed updates*, regardless of line breaks, comments, commented out imports, etc.

	For example, if the original file has:

	pdx-1: import { something } from "some-package";
	pdx-2: import { anotherThing } from "another-package";
	pdx-3: // import "another";
	pdx-4:
	pdx-5:
	pdx-6: import "yet-another-package";
	pdx-7: import { exec } from "exec-package";
	pdx-8:
	pdx-9: function main() {
	pdx-10:   exec();
	pdx-11: }

	And the *proposed updates* have:

	pdx-new-1: // ... existing code ...
	pdx-new-2: import { logger } from "logger-package";
	pdx-new-3:
	pdx-new-4: function main() {
	pdx-new-5:   exec();
	pdx-new-6:   logger.info("Hello, world!");
	pdx-new-7: }

	Then the reference should look like this:

	<references>
		<reference 
			comment="// ... existing code ..."
			description="Imports"
			proposedLine="pdx-new-1"
			structure="top level"
			structureOpens=""
			structureCloses=""
			originalStart="pdx-1"
			originalEnd="pdx-7"
		/>
	</references>

	Note that ALL the import statements are included in the reference.

	*

	If a reference *includes* a code structure, DO NOT treat the reference as if it were *inside* the code structure. For example, if the *original file* has:

	pdx-1: class MyClass { 
	pdx-2:   function update() {
	pdx-3:     const conn = await getConnection();
	pdx-4:     const res = await execUpdate(conn);
	pdx-5:   }
	pdx-6: }
	pdx-7:
	pdx-8: class AnotherClass {
	pdx-9:   function doSomething() {
	pdx-10:     const conn = getConnection();
	pdx-11:     const res = execUpdate(conn);
	pdx-12:   }
	pdx-13: }

	And the *proposed updates* have:

	pdx-new-1: class MyClass {
	pdx-new-2:   function update() {
	pdx-new-3:     const conn = getConnection();
	pdx-new-4:     const res = execUpdate(conn);
	pdx-new-5:   }
	pdx-new-6: }
	pdx-new-7:
	pdx-new-8: // ... existing code ...
	
	Then the "// ... existing code ..." reference must include the *entire* 'AnotherClass' structure, including the class definition and closing brace. Do NOT treat the reference as if it were *inside* the 'AnotherClass' structure.

	The reference for the above example would look like this:

	<references>
		<reference 
			comment="// ... existing code ..."
			description="AnotherClass"
			proposedLine="pdx-new-8"
			structure="top level"
			structureOpens=""
			structureCloses=""
			originalStart="pdx-8"
			originalEnd="pdx-13"
		/>
	</references>
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
