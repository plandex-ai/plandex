package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GetReferencesPrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNumsWithPrefix(changesFile, "pdx-new-")

	s := ReferencesPrompt + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

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

Reference comments often won't exactly match one of the above examples, but they will always be referencing a block of code from the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.
`

var ReferencesPrompt = `
You are an AI that analyzes an *original file* and *proposed updates* to that file and then identifies *all* *reference comments* present in the *proposed updates*.

` + ExampleReferences + `
	
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

		**structureOpens**: The entire line from the *original file*, prefixed by 'pdx-', that contains the opening symbol of the code structure identified in the 'structure' property.

		**structureCloses**: The entire line from the *original file*, prefixed by 'pdx-', that contains the closing symbol of the code structure identified in the 'structure' property.

		**originalStart**: The starting line number, prefixed by 'pdx-', in the *original file* that the reference is referring to. MUST be less than or equal to the 'originalEnd' line number and MUST be greater than or equal to 1. 

		**originalEnd**: The ending line number, prefixed by 'pdx-', in the *original file* that the reference is referring to. If the referenced code in the *original file* is a single line, the originalStart and originalEnd must be the same. 'originalEnd' MUST be greater than or equal to 'originalStart' and MUST be less than or equal to the last line number in the *original file*.

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

	You MUST carefully consider code structures when determining the 'originalStart' and 'originalEnd' lines for a reference. If a reference looks like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   constructor() {
	pdx-new-3:     this.initialize();
	pdx-new-4:   }
	pdx-new-5: 
	pdx-new-6:   // ... existing code ...
	pdx-new-7: }

	The the reference ONLY refers to the code within the 'MyClass' class. If there is additional code before or after the 'MyClass' class, it must not be included. The same applies to other code structures, like functions, loops, conditionals, etc.
	
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
	
	Similarly, if a reference comes at the end of a structure, preceded by new or changed code, like this:

	pdx-new-1: class MyClass {
	pdx-new-2:   function update() {
	pdx-new-3:     const conn = await getConnection();
	pdx-new-4:     const res = await execUpdate(conn);
	pdx-new-5:   }
	pdx-new-6:   // ... existing code ...
	pdx-new-7: }

	Then the '// ... existing code ...' reference must include ALL code from the *original file* inside the 'MyClass' structure. The 'originalStart' must be one line *after* the 'structureOpens' line and the 'originalEnd' must be one line *before* the 'structureCloses' line.

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
	
	In cases like this with multiple references within the same structure, you MUST NOT duplicate code in any of the references. Taken together, the references must cover the entire 'MyClass' structure with no code from the *original file* repeated or left out.

	If there are commented out lines that still logically belong to a section that is referenced, you MUST include those lines in the reference.

	Do not include any additional text in your final output. Only output the references.
`

var UpdatedListReplacementsFn = openai.FunctionDefinition{
	Name: "listChangesWithLineNums",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"problems": {
				Type: jsonschema.String,
			},
			"originalSections": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"description": {
							Type: jsonschema.String,
						},
						"structure": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"structure": {
									Type: jsonschema.String,
								},
								"structureOpens": {
									Type: jsonschema.String,
								},
								"structureCloses": {
									Type: jsonschema.String,
								},
							},
						},
						"reasoning": {
							Type: jsonschema.String,
						},
						"sectionStartLine": {
							Type: jsonschema.String,
						},
						"sectionEndLine": {
							Type: jsonschema.String,
						},
						"shouldChange": {
							Type: jsonschema.Boolean,
						},
						"shouldRemove": {
							Type: jsonschema.Boolean,
						},
					},
					Required: []string{
						"description",
						"structure",
						"reasoning",
						"sectionStartLine",
						"sectionEndLine",
						"shouldChange",
						"shouldRemove",
					},
				},
			},
			"entireFileReasoning": {
				Type: jsonschema.String,
			},
			"entireFile": {
				Type: jsonschema.Boolean,
			},
			"changes": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"section": {
							Type: jsonschema.String,
						},
						"summary": {
							Type: jsonschema.String,
						},
						"newReasoning": {
							Type: jsonschema.String,
						},
						"reasoning": {
							Type: jsonschema.String,
						},
						"structureReasoning": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"old": {
									Type: jsonschema.Object,
									Properties: map[string]jsonschema.Definition{
										"structure": {
											Type: jsonschema.String,
										},
										"structureOpens": {
											Type: jsonschema.String,
										},
										"structureCloses": {
											Type: jsonschema.String,
										},
									},
									Required: []string{"structure", "structureOpens", "structureCloses"},
								},
								"new": {
									Type: jsonschema.Object,
									Properties: map[string]jsonschema.Definition{
										"structure": {
											Type: jsonschema.String,
										},
										"structureOpens": {
											Type: jsonschema.String,
										},
										"structureCloses": {
											Type: jsonschema.String,
										},
									},
									Required: []string{"structure", "structureOpens", "structureCloses"},
								},
							},
						},
						"orderReasoning": {
							Type: jsonschema.String,
						},
						"hasChange": {
							Type: jsonschema.Boolean,
						},
						"insertBefore": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"shouldInsertBefore": {
									Type: jsonschema.Boolean,
								},
								"line": {
									Type: jsonschema.String,
								},
							},
							Required: []string{"shouldInsertBefore", "firstLine"},
						},
						"insertAfter": {
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"shouldInsertAfter": {
									Type: jsonschema.Boolean,
								},
								"line": {
									Type: jsonschema.String,
								},
							},
							Required: []string{"shouldInsertAfter", "firstLine"},
						},
						"new": {
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
					},
					Required: []string{
						"section",
						"summary",
						"newReasoning",
						"structureReasoning",
						"orderReasoning",
						"hasChange",
						"insertBefore",
						"insertAfter",
						"new",
						"old",
					},
				},
			},
		},
		Required: []string{"originalSections", "entireFileReasoning", "entireFile", "problems", "changes"},
	},
}

func AddImplicitReferencesPrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)

	s := ImplicitReferencesPrompt + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesFile)

	return s
}

var ImplicitReferencesPrompt = `
You are an AI that analyzes an *original file* and *proposed updates* to that file and then rewrites the *proposed updates* to fit this general format:

<ProposedUpdatesWithReferences>
// ... existing code ...

[code that is being changed]

// ... existing code ...

[code that is being added]

// ... existing code ...
</ProposedUpdatesWithReferences>

There will not *always* be a need for an "... existing code ..." comment before or after the code that is being changed or added. Only add it if there is a clear semantic reason to do so.

The *proposed updates* will often already include comments similar to "... existing code ...". If so, leave these in place and do not duplicate them. Only add the "... existing code ..." comment if there is a clear semantic reason to do so that is not already accomplished by existing "... existing code ..." (or similar) comments in the *proposed updates*.

In other words, sometimes the *proposed updates* will already have followed this format correctly. In that case, simply repeat the *proposed updates* verbatim inside the <ProposedUpdatesWithReferences> element. 

Other times, there will be missing "... existing code ..." comments that you need to add. In that case, add them as necessary inside the <ProposedUpdatesWithReferences> element.

Inside the <ProposedUpdatesWithReferences> element, include nothing but the code for the *proposed updates* with the addition of any "... existing code ..." comments that you are adding. Don't include any other text. Don't include triple backticks or any other formatting.
`

// var ImplicitReferencesPrompt = `
// You are an AI that analyzes an *original file* and *proposed updates* to that file and then:

// *Sections*: Divides the *original file* into sections based on functionality, logic, code structure, and general organization.
// *Section Reasoning*: Evaluates each section to determine which sections should be *changed* and which sections should be *preserved* as is.
// *Implicit References*: Rewrites the *proposed updates* to add *reference comments* for any sections from the *original file* that should be *preserved* in the final output if there is not already a *reference comment* for that section in the *proposed updates*.

// Now I'll provide more detail and examples for each step.

// *Sections*: Divides the *original file* into sections based on functionality, logic, code structure, and general organization.

// You must list every section that exists in the *original file*. When large sections of the *original file* are not changing, combine them into a single section. Only include sections from the *original file*. Do NOT include sections from the *proposed updates*.

// Don't make the sections overly small and granular unless there is a clear semantic reason to do so. For example, if there are many small functions in a file, don't create a section for each section. Sections should be larger than that and instead reflect the general structure of a file, rather than being a long list of every top-level code structure.

// For each section, give it a name and a brief description of what it does. At this point, don't yet assess whether the section should be *changed* or *preserved*, just focus on describing the section as it relates to the rest of the *original file*.

// Example:

// ---

// Sections:

// 1 - Import statements
// 2 - initialize() function
// 3 - main() function

// ---

// *Section Reasoning*: Evaluates each section to determine which sections should be *changed* and which sections should be *preserved*.

// For each section listed in 1, list it again by name and evaluate whether it should be *changed* or *preserved* based on the *proposed updates*. Give your reasoning: a brief evaluation of how this section relates to the *proposed updates*, and whether all or any part of it will be changed, removed, or preserved as is. If only part of the section should be changed, explain which part(s) will change and which will remain the same.

// Anytime there is a partial change to a section, it is CRITICAL that you explain BOTH which parts of the section should be *changed* and which parts will be *preserved*. Be very specific and precise.

// At the end of your reasoning, output 'shouldChange' followed by a boolean value (true or false).

// If 'shouldChange' is true, add a 'changeDescription' attribute that describes which parts of the section will be changed and which will remain the same. Set a 'preservePart' attribute under the 'changeDescription'. If part of the section will remain the same, set 'preservePart' attribute to 'true', otherwise 'false'. If 'preservePart' is true, set a 'referenceCovers' attribute that whether the code that is being preserved is covered by a reference comment in the *proposed updates*. Describe in words which *reference comment* covers the preserved code. Otherwise, set 'referenceCovers' to 'false'.

// After the 'shouldChange', add a 'shouldRemove' followed by a boolean value (true or false).

// Then add a 'shouldPreserve' followed by a boolean value (true or false).

// 'shouldChange' should be true if any part of the section should be changed.

// 'shouldRemove' should be true if the entire section should be removed.

// 'shouldPreserve' should be true if the entire section should be preserved as is.

// If 'shouldPreserve' is true, set a 'referenceCovers' attribute that whether the code that is being preserved is covered by a reference comment in the *proposed updates*. Describe in words which *reference comment* covers the preserved code. Otherwise, set 'referenceCovers' to 'false'.

// Only one of "shouldChange", "shouldRemove", or "shouldPreserve" can be true. The other two must be false. One of these three must be true.

// A section is 'covered' by a *reference comment* if *any* existing *reference comment* in the *proposed updates* references that code. If the code that is being preserved is at the beginning of a code structure in the *original file* and there's a *reference comment* like "// ... existing code ..." at the *beginning* of the code structure in the *proposed updates*, then the section is covered by that reference comment. Example:

// [ORIGINAL FILE]

// function update(name string, id string) {
// 	const client = getClient();
// 	const tx = await client.startTransaction();
// 	const logStatement = prepLogStatement('some-app');
// 	await sendLogStatement(logStatement);
// 	await doUpdate(name, id);
// }

// [/END ORIGINAL FILE]

// [PROPOSED UPDATES]

// function update(name string, id string) {
// 	// ... existing code ...

// 	if (canUpdate(name, id)) {
// 		await doUpdate(name, id);
// 	}
// }

// [/END PROPOSED UPDATES]

// In the case above, all the code before 'await doUpdate();' is covered by the reference comment "// ... existing code ..." at the start of the 'update' function in the *proposed updates*.

// Similarly, if the code that is being preserved is at the end of a code structure in the *original file* and there's a *reference comment* like "// ... existing code ..." at the *end* of the code structure in the *proposed updates*, then the section is covered by that reference comment. Example:

// [ORIGINAL FILE]

// function update(name string, id string) {
// 	await doUpdate(name, id);

// 	await client.commit();
// 	await sendAnalyticsEvent('update-user', { name, id });
// }

// [/END ORIGINAL FILE]

// [PROPOSED UPDATES]

// function update(name string, id string) {
// 	await doUpdate(name, id);

// 	// ... existing code ...
// }

// [/END PROPOSED UPDATES]

// In the case above, all the code after 'await doUpdate(name, id);' is covered by the reference comment "// ... existing code ..." at the end of the 'update' function in the *proposed updates*.

// Similarly, if the code that is being preserved is in the middle of a code structure in the *original file* between two blocks of code that are being updated by the *proposed updates*, and there's a *reference comment* like "// ... existing code ..." *between* the two blocks of code in the *proposed updates*, then the section is covered by that reference comment. Example:

// [ORIGINAL FILE]

// function update(name string, id string) {
// 	const client = getClient();
// 	const logStatement = prepLogStatement('some-app');
// 	await sendLogStatement(logStatement);
// 	await doUpdate(name, id);
// 	await sendAnalyticsEvent('update-user', { name, id });
// 	await client.commit();
// }

// [/END ORIGINAL FILE]

// [PROPOSED UPDATES]

// function update(name string, id string) {
// 	const asyncClient = getAsyncClient();

// 	// ... existing code ...

// 	if (canUpdate(name, id)) {
// 		await doUpdate(name, id);
// 	}

// 	// ... existing code ...

// 	await asyncClient.commit();
// }

// [/END PROPOSED UPDATES]

// In the case above, the lines 'const logStatement = prepLogStatement('some-app');' and 'await sendLogStatement(logStatement);' that are being preserved are covered by the reference comment "// ... existing code ..." *between* the 'const asyncClient = getAsyncClient();' line and the 'if (canUpdate(name, id)) {' line in the *proposed updates*. The line 'await sendAnalyticsEvent('update-user', { name, id });' is covered by the reference comment "// ... existing code ..." *between* the '}' line (after the 'await doUpdate(name, id);' line) and the 'await asyncClient.commit();' line in the *proposed updates*.

// ---

// Section Reasoning:

// 1 - Import statements
// 	Reasoning: The import statements are not changing. They should be included in the final output as is.
// 	shouldChange: false
// 	shouldRemove: false
// 	shouldPreserve: true
// 		referenceCovers: false

// 2 - initialize() function
// 	Reasoning: New code for logging will be added at the beginning of the function. The rest of the function will remain the same and should be included in the final output as is.
// 	shouldChange: true
// 		changeDescription: New code for logging will be added at the beginning of the function. The rest of the function will remain the same.
// 		preservePart: true
// 			referenceCovers: The existing code in the initialize() function is already covered by the reference comment "// ... existing code ..." at the start of the initialize() function in the *proposed updates*.
// 	shouldRemove: false
// 	shouldPreserve: false

// 3 - main() function
// 	Reasoning: Between the last line of code that connects to the database (pdx-18) and the first of code that calls the 'update()' function (pdx-5), new code will be added for a permission check. The rest of the function will remain the same and should be included in the final output as is.
// 	shouldChange: true
// 		changeDescription: New code for a permission check will be added at the beginning of the function. The rest of the function will remain the same.
// 		preservePart: true
// 			referenceCovers: The existing code in the main() function is already covered by the reference comment "// ... existing update code ..." at the start of the main() function in the *proposed updates*.
// 	shouldRemove: false
// 	shouldPreserve: false

// ---

// *Implicit References*: Rewrites the *proposed updates* to add *reference comments* for any sections from the *original file* that should be *preserved* in the final output if there is not already a *reference comment* for that section in the *proposed updates*.

// ` + ExampleReferences + `

// In section 3, first list any sections or parts of sections from the *original file* that should be *preserved* in the final output but are not included in the *proposed updates* and do not already have a reference comment referencing them in the *proposed updates*.

// If there are no such sections, state that explicitly and stop there.

// Next, if there are any sections from the *original file* that should be *preserved* in the final output but do not already have a reference comment referencing them in the *proposed updates*, output a <ProposedUpdatesWithReferences> element

// The content of the <ProposedUpdatesWithReferences> element must be the *proposed updates* repeated *VERBATIM* with the addition of any *reference comments* for sections that should be *preserved* in the final output *but do not already have a reference comment referencing them* in the *proposed updates*.

// Inside the <ProposedUpdatesWithReferences> element, include nothing but the code for the *proposed updates* with the addition of any *reference* comments that you are adding. Don't include any other text. Don't include triple backticks or any other formatting.

// DO NOT add new reference comments if all the code from the *original file* that isn't changing is already present or already referenced by reference comments in the *proposed updates*. In that case, do not output a <ProposedUpdatesWithReferences> element.

// Every block of code from the *original file* that is *preserved* in the final output must either be reproduced in code in the *proposed updates* or referenced by a *single* *reference comment* in the *proposed updates*.

// You ABSOLUTELY MUST NOT add additional *reference comments* for code that is *already* covered by a reference comment in the *proposed updates*. There MUST NEVER be two or more consecutive reference comments without any actual code in between. If you add multiple consecutive reference comments like this:

// // ... existing code to start transaction ...

// // ... existing code ...

// Then you have failed at this step. Because if there is ALREADY a *reference comment* (like "// ... existing code to start transaction ..." in the *proposed updates*) for the code that is being preserved, you MUST NOT add another reference comment for it. The "// ... existing code ..." reference ALREADY covers that section of code that is being preserved, so no additional reference comment is needed. Again you MUST NOT output multiple consecutive reference comments without any actual code in between them.

// You MUST ALWAYS use a single-line comment and the exact text "... existing code ..." to reference code that is *already included* in the *proposed updates*. For example, if the programming language is JavaScript, you MUST use "// ... existing code ...". In python, you MUST use "# ... existing code ...". And so on for other languages. Do not add any other text to the reference comment. You MUST NOT under *any circumstances* use any other form of *reference comment*. ALWAYS use the exact text "... existing code ..." in a single-line comment that is appropriate for the programming language.

// You ABSOLUTELY MUST NOT add *reference comments* for code that is *already included* in the *proposed updates*.

// Example:

// ---

// Implicit References:

// The imports section is not changing and should be included in the final output as is. There is no reference comment mentioning the existing imports section in the *proposed updates*. Therefore, I will add a new reference comment for the existing imports section in the *proposed updates*.

// The other sections that aren't changing are already referenced by reference comments in the *proposed updates*.

// <ProposedUpdatesWithReferences>
// // ... existing imports ...
// import { createLogger } from './logger';

// function initialize() {
// 	const logger = createLogger('some-app');

// 	// ... existing code ...
// }

// function main() {
// 	// ... existing code ...

// 	if (canUpdate()) {
// 		// ... existing update code ...
// 	}

// 	// ... existing code to commit db transaction...
// }
// </ProposedUpdatesWithReferences>

// ---

// `
