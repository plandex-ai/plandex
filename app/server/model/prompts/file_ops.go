package prompts

const FileOpsPlanningPrompt = `
## File Operations Planning

You can create special subtasks for file operations that move, remove, or reset changes to files that are in context or have pending changes. These operations *can only* be used on files that are in context or have pending changes. They *cannot* be used on other files or directories in the user's project (or any other files/directories). *ONLY* use these sections for files that are in context or have pending changes.

## Important Notes On Planning File Operations

1. These sections can only operate on files that are:
  - Already in context, OR
  - Have pending changes from earlier in the plan
  - All files that are in context or have pending changes will be listed in your prompt

2. You cannot:
  - Move, remove, or reset files that aren't in context or pending
  - Create new directories (they will be created as needed by the operations)
  - Move a file to a path that is *already* in context or pending (and would therefore overwrite the existing file)

3. Updated State:
  - Note that when you *move* a file, any further updates to that file must be applied to the *new* location. The context in your prompt will be updated to reflect the new location. Ensure the new path takes precedence over any updates to the old path in the conversation history.
  - Note that when you *remove* a file, applying further updates to that file will require *creating a new file*. The file must be considered to not exist unless you explicitly create it again. The context in your prompt will be updated to reflect the file's removal. Ensure the file's removal takes precedence over any updates to the file in the conversation history.

In most cases, these special file operations are *not* used when initially implementing a plan, since in that case you are only creating files and updating them, and possibly writing to the _apply.sh script if execution mode is enabled and you need to take actions on the user's machine when the plan is applied. The only exception is if the users specifically asks you to move or remove files in context in the initial prompt. Otherwise, do not use these operations when initially implementing a plan.

In most cases, file operations are only useful for revising a plan with pending changes in response to another prompt from the user. For example, if you have created several files and the user asks you to create them in a different directory, you can use a move operation to move them to the new directory. Similarly, if a user tells you that a file you have created is not needed, you can use a remove operation to remove it. Similarly, if a user tells you that your changes to a particular file are incorrect or not needed, you can use a reset operation to clear the pending changes to that file.

You MUST NOT implement any file operations in this section. You MUST only plan the file operations by including them in the ### Tasks section as subtasks. They will be implemented in subsequent responses.
`

const FileOpsImplementationPrompt = `
## File Operations Implementation

You can perform file operations using special sections in your response. These sections allow you to move, remove, or reset changes to files that are in context or have pending changes. These special sections *can only* be used on files that are in context or have pending changes. They *cannot* be used on other files or directories in the user's project (or any other files/directories). *ONLY* use these sections for files that are in context or have pending changes.

You ABSOLUTELY MUST end every file operation section with a <EndPlandexFileOps/> tag.

*Move Files Section:*

Use the '### Move Files' section to move or rename files:

### Move Files
- ` + "`source/path.tsx` → `dest/path.tsx`" + `
- ` + "`components/button.tsx` → `pages/button.tsx`" + `
<EndPlandexFileOps/>

Rules for the Move Files section:
- Each line must start with a dash (-)
- Source and destination paths must be wrapped in backticks (` + "`" + `)
- Paths must be separated by → (Unicode arrow, NOT ->)
- Can only move individual files (not directories)
- All source paths MUST match a path in context or that has pending changes
- Destination path must be in the same base directory as files in context
- Destination path MUST NOT already exist in context or pending files—i.e. you cannot move a file to a path that is *already* in context or pending (and would therefore overwrite the existing file)
- You CAN move a file to a directory that does not exist yet—it will be created as needed automatically
- You MUST end the '### Move Files' section with a <EndPlandexFileOps/> tag

*Remove Files Section:*

Use the '### Remove Files' section to remove/delete files:

### Remove Files
- ` + "`components/page.tsx`" + `
- ` + "`layouts/header.tsx`" + `
<EndPlandexFileOps/>

Rules for the Remove Files section:
- Each line must start with a dash (-)
- Paths must be wrapped in backticks (` + "`" + `)
- Can only remove individual files (not directories)
- All paths MUST match a path in context or that has pending changes
- Each path must be on its own line
- You MUST end the '### Remove Files' section with a <EndPlandexFileOps/> tag

*Reset Changes Section:*

Use the '### Reset Changes' section to clear pending changes for files:

### Reset Changes
- ` + "`components/page.tsx`" + `
- ` + "`layouts/header.tsx`" + `
<EndPlandexFileOps/>

Rules for the Reset Changes section:
- Each line must start with a dash (-)
- Paths must be wrapped in backticks (` + "`" + `)
- Can only reset individual files (not directories)
- Can only reset files that have pending changes
- Each path must be on its own line
- You MUST end the '### Reset Changes' section with a <EndPlandexFileOps/> tag

## Important Notes

1. These sections can only operate on files that are:
  - Already in context, OR
  - Have pending changes from earlier in the plan
  - All files that are in context or have pending changes will be listed in your prompt
  - '### Reset Changes' can *only* reset files that have pending changes

2. You cannot:
  - Move, remove, or reset files that aren't in context or pending
  - Create new directories (they will be created as needed by the operations)
  - Include comments or additional text within these sections
  - Move a file to a path that is *already* in context or pending (and would therefore overwrite the existing file)

3. Format Rules:
  - Section headers must be exactly as shown (### Move Files, ### Remove Files, ### Reset Changes)
  - All file paths must be wrapped in backticks (` + "`" + `)
  - Move operations must use the → arrow character (Unicode arrow, NOT ->)
  - Each operation must be on its own line starting with a dash (-)
  - Empty lines between operations are allowed
  - No additional text or comments are allowed within these sections
  - You MUST end each file operation section with a <EndPlandexFileOps/> tag

4. Updated State
  - Note that when you *move* a file, any further updates to that file must be applied to the *new* location. The context in your prompt will be updated to reflect the new location. Ensure the new path takes precedence over any updates to the old path in the conversation history.
  - Note that when you *remove* a file, applying further updates to that file will require *creating a new file*. The file must be considered to not exist unless you explicitly create it again. The context in your prompt will be updated to reflect the file's removal. Ensure the file's removal takes precedence over any updates to the file in the conversation history.

You must follow the specified format *exactly* for each of these sections.
`

const FileOpsImplementationPromptSummary = `
Use special sections to perform file operations on files in context or with pending changes:

Key instructions for file operations:

- ONLY use on files that are in context or have pending changes
- Three available sections with exact formatting:
    - '### Move Files' (using ` + "`source` → `dest`" + ` format)
    - '### Remove Files' (using backtick paths)
    - '### Reset Changes' (using backtick paths)
- Every path MUST be wrapped in backticks (` + "`" + `)
- Every line MUST start with a dash (-)
- Can ONLY operate on individual files (not directories)
- DO NOT UNDER ANY CIRCUMSTANCES:
    - Include comments or additional text in these sections
    - Use on files not in context or pending
- These sections are for REVISING plans, not initial implementation
- When making changes, choose between:
    - Iterating on current pending changes
    - Using '### Reset Changes' to start fresh on a file
- You MUST end each file operation section with a <EndPlandexFileOps/> tag
`
