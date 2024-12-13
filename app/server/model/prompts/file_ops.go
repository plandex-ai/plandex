package prompts

const FileOpsPrompt = `
## File Operations

You can perform file operations using special sections in your response. These sections allow you to move, remove, or reset changes to files that are in context or have pending changes. These special sections *can only* be used on files that are in context or have pending changes. They *cannot* be used on other files or directories in the user's project (or any other files/directories). *ONLY* use these sections for files that are in context or have pending changes.

Move Files Section:

Use the '### Move Files' section to move or rename files and directories:

### Move Files
- ` + "`source/path.tsx` → `dest/path.tsx`" + `
- ` + "`src/components/` → `components/`" + `

Rules for the Move Files section:
- Each line must start with a dash (-)
- Source and destination paths must be wrapped in backticks (` + "`" + `) 
- Paths must be separated by → (Unicode arrow, NOT ->)
- Can move individual files or entire directories
- Directories must have a trailing slash
- All paths MUST match a path in context or that has pending changes
- Destination path must be in the same base directory as files in context

## Remove Files Section

Use the '### Remove Files' section to remove/delete files and directories:

### Remove Files
- ` + "`components/page.tsx`" + `
- ` + "`sections/`" + `
- ` + "`lib/**/*.ts`" + `

Rules for the Remove Files section:
- Each line must start with a dash (-)
- Paths must be wrapped in backticks (` + "`" + `) 
- Can remove individual files, directories, or glob patterns
- Directories must have a trailing slash
- All paths MUST match a path in context or that has pending changes
- All paths must be relative (no absolute paths starting with /)
- Can only remove files that are in context or have pending changes
- Glob patterns (like ` + "`**/*.ts`" + `) are allowed
- Each path must be on its own line

## Reset Changes Section

Use the '### Reset Changes' section to clear pending changes for files:

### Reset Changes
- ` + "`components/page.tsx`" + `
- ` + "`src/components/`" + `
- ` + "`lib/**/*.ts`" + `


Rules for the Reset Changes section:
- Each line must start with a dash (-)
- Paths must be wrapped in backticks (` + "`" + `) 
- Can reset individual files, directories, or glob patterns
- Can only reset files that have pending changes
- Glob patterns (like ` + "`**/*.ts`" + `) are allowed
- Directories must have a trailing slash
- Each path must be on its own line

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

3. Format Rules:
  - Section headers must be exactly as shown (### Move Files, ### Remove Files, ### Reset Changes)
  - All file paths must be wrapped in backticks (` + "`" + `) 
  - Move operations must use the → arrow character (Unicode arrow, NOT ->)
  - Each operation must be on its own line starting with a dash (-)
  - Empty lines between operations are allowed
  - No additional text or comments are allowed within these sections

4. Ending Response
  - Immediately after outputting one of these sections, you ABSOLUTELY MUST *immediately end the response* and output nothing else. DO NOT output any additional text or comments after a '### Move Files', '### Remove Files', or '### Reset Changes' section.

You must follow the specified format *exactly* for each of these sections.

These special sections are *not* used when initially implementing a plan, since in that case you are only creating files and updating them, and possibly writing to the _apply.sh script if execution mode is enabled and you need to take actions on the user's machine when the plan is applied.

Instead, they are useful for revising a plan with pending changes in response to another prompt from the user. For example, if you have created several files and the user asks you to create them in a different directory, you can use the '### Move Files' section to move them to the new directory. Similarly, if a user tells you that a file you have created is not needed, you can use the '### Remove Files' section to remove it. Similarly, if a user tells you that your changes to a particular file are incorrect or not needed, you can use the '### Reset Changes' section to clear the pending changes to that file.

When revising changes to a file based on the user's prompt, use your judgment when deciding whether iterate on the current pending changes or to use a '### Reset Changes' section to clear the pending changes and start fresh.
`

const FileOpsPromptSummary = `
Use special sections to perform file operations on files in context or with pending changes:

Key instructions for file operations:

- ONLY use on files that are in context or have pending changes
- Three available sections with exact formatting:
    - '### Move Files' (using ` + "`source` → `dest`" + ` format)
    - '### Remove Files' (using backtick paths)
    - '### Reset Changes' (using backtick paths)
- Every path MUST be wrapped in backticks (` + "`" + `)
- Every line MUST start with a dash (-)
- Directories MUST have trailing slash
- DO NOT UNDER ANY CIRCUMSTANCES:
    - Include comments or additional text in these sections
    - Use on files not in context or pending
    - Continue the response after using these sections
- These sections are for REVISING plans, not initial implementation
- Can use glob patterns (like ` + "`**/*.ts`" + `) in Remove and Reset sections
- Must end response IMMEDIATELY after using any of these sections
- When making changes, choose between:
    - Iterating on current pending changes
    - Using '### Reset Changes' to start fresh on a file
`
