package prompts

const ExampleReferences = `
A reference comment is a comment that references code in the *original file* for the purpose of making it clear where a change should be applied. Examples of reference comments include:

  - // ... existing code...
  - # Existing code...
  - /* ... */
  - // rest of the function...
  - <!-- rest of div tag -->
  - // ... rest of function ...
  - // rest of component...
  - # other methods...
  - // ... rest of init code...
  - // rest of the class...
  - // other properties
  - // other methods
  - // ... existing properties ...
  - // ... existing values ...
  - // ... existing text ...

Reference comments often won't exactly match one of the above examples, but they will always be referencing a block of code from the *original file* that is left out of the *proposed updates* for the sake of focusing on the specific change that is being made.

Reference comments do NOT need to be valid comments for the given file type. For file types like JSON or plain text that do not use comments, reference comments in the form of '// ... existing properties ...' or '// ... existing values ...' or '// ... existing text ...' can still be present. These MUST be treated as valid reference comments regardless of the file type or the validity of the syntax.
`

const CommentClassifierPrompt = `
You must analyze the *original file* and the *proposed updates* and output a <PlandexComments> element that lists *EVERY* comment in the *proposed updates*, including the line number of each comment prefixed by 'pdx-new-'. Below each comment, evaluate whether it is a reference comment.

` + ExampleReferences + `

 For each comment in the proposed changes, focus on whether the comment is clearly referencing a block of code in the *original file*, whether it is explaining a change being made, or whether it is a comment that was carried over from the *original file* but does *not* reference any code that was left out of the *proposed updates*. After this evaluation, state whether each comment is a reference comment or not. Only list valid *comments* for the given programming language in the comments section. Do not include non-comment lines of code in the comments section.

 Example:

<PlandexComments>
pdx-new-1: // ... existing code to start transaction ...
Evaluation: refers the code at the beginning of the 'update' function that starts the database transaction.
Reference: true

pdx-new-5: // verify user permission before performing update
Evaluation: describes the change being made. Does not refer to any code in the *original file*.
Reference: false

pdx-new-10: // ... existing update code ...
Evaluation: refers the code inside the 'update' function that updates the user.
Reference: true
</PlandexComments>

If there are no comments in the *proposed updates*, output an empty <PlandexComments> element.

ONLY include valid comments for the language in this list. Do NOT include any other lines of code in the comments section. You MUST include ALL comments from the *proposed updates*.
`
