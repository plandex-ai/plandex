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

var ReferencesPrompt = `
You are an AI that analyzes an *original file* and *proposed updates* to that file and then identifies *all* *reference comments* present in the *proposed updates*.

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

	Comments often won't exactly match one of the above examples, but they will always be referencing a block of code from the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.
	
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
			originalStart="pdx-456" 
			originalEnd="pdx-460" 
		/>
	</references>

	Explanation of 'reference' tag attributes:

		**comment**: The comment that includes the reference.

		**description**: A brief description of the reference. Make a special note of where the reference *begins* and *ends in the *original file* so that it does *not* include any lines that are already present in the *proposed updates*.

		**proposedLine**: The line number, prefixed by 'pdx-new-', in the *proposed updates* that the reference is referring to. MUST be a line that exists in the *proposed updates* and contains the reference comment in 'comment' attribute.

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
			originalStart="pdx-2"
			originalEnd="pdx-3"
		/>
		<reference 
			comment="// ... existing update code ..."
			description="Code inside the canUpdate() conditional for updating the user"
			proposedLine="pdx-new-6"
			originalStart="pdx-5"
			originalEnd="pdx-6"
		/>
		<reference 
			comment="// ... existing code to commit db transaction..."
			description="Code for committing the database transaction and handling errors"
			proposedLine="pdx-new-9"
			originalStart="pdx-8"
			originalEnd="pdx-17"
		/>
	</references>

	---

	Note that in the example, the "// verify user permission before performing update" comment is *not* a reference since it refers to code that is included in the *proposed updates* and does *not* refer to code in the *original file* that was left out of the *proposed updates*.

	In description attributes, make a special note of where the reference *begins* and *ends in the *original file* so that it does *not* include any lines that are already present in the *proposed updates*. Note that in the above example, the function signature and opening brace of the 'update' function is *not included* in the reference since it's already present in the *proposed updates*. And node in the last reference, the closing brace of the 'update' function is *not included* since it's already present in the *proposed updates*.

	If there are no clear references in the *proposed updates*, output an empty <references> element. The *proposed updates* often will not inlcude any reference comments. In that case, output an empty <references> element. Do NOT include reference comments that are not present in the *proposed updates* or comments that are not clearly references to a block of code in the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.

	You MUST carefully consider code structures when determining the 'originalStart' and 'originalEnd' lines for a reference. If a reference looks like this:

	pdx-1: class MyClass {
	pdx-2:   constructor() {
	pdx-3:     this.initialize();
	pdx-4:   }
	pdx-5: 
	pdx-6:   // ... existing code ...
	pdx-7: }

	The the reference ONLY refers to the code within the 'MyClass' class. If there is additional code before or after the 'MyClass' class, it must not be included. The same applies to other code structures, like functions, loops, conditionals, etc. 

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
							Type: jsonschema.String,
						},
						"closingSyntaxReasoning": {
							Type: jsonschema.String,
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
						"closingSyntaxReasoning",
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
