package prompts

import (
	"fmt"
	"time"
)

const sharedPromptWrapperFormatStr = "# The user's latest prompt:\n```\n%s\n```\n\n" + `Please respond according to the 'Your instructions' section above.

Do not ask the user to do anything that you can do yourself. Do not say a task is too large or complex for you to complete--do your best to break down the task and complete it even if it's very large or complex.

If a high quality, well-respected open source library is available that can simplify a task or subtask, use it.

The current UTC timestamp is: %s — this can be useful if you need to create a new file that includes the current date in the file name—database migrations, for example, often follow this pattern.

User's operating system details:
%s

---
%s
---
`

const planningPromptWrapperFormatStr = sharedPromptWrapperFormatStr + `

Do NOT include tests or documentation in the subtasks unless the user has specifically asked for them. Do not include extra code or features beyond what the user has asked for. Focus on the user's request and implement only what is necessary to fulfill it.

` + CombineSubtasksPrompt + `

At the end of the '### Tasks' section, you ABSOLUTELY MUST ALWAYS include a <EndPlandexTasks/> tag, then end the response.

Example:

### Tasks

1. Create a new file called 'src/main.rs' with a 'main' function that returns 'Hello, world!'

2. Write a basic test for the 'main' function

<EndPlandexTasks/>
`

var PlanningPromptWrapperTokens int

const implementationPromptWrapperFormatStr = sharedPromptWrapperFormatStr + `

If you're making a plan, remember to label code blocks with the file path *exactly* as described in point 2, and do not use any other formatting for file paths. **Do not include explanations or any other text apart from the file path in code block labels.**

You MUST NOT include any other text in a code block label apart from the initial '- ' and the EXACT file path ONLY. DO NOT UNDER ANY CIRCUMSTANCES use a label like 'File path: src/main.rs' or 'src/main.rs: (Create this file)' or 'File to Create: src/main.rs' or 'File to Update: src/main.rs'. Instead use EXACTLY 'src/main.rs:'. DO NOT include any explanatory text in the code block label like 'src/main.rs: (Add a new function)'. It is EXTREMELY IMPORTANT that the code block label includes *only* the initial '- ', the file path, and NO OTHER TEXT whatsoever. If additional text apart from the initial '- ' and the exact file path is included in the code block label, the plan will not be parsed properly and you will have failed at the task of generating a usable plan. 

Always use an opening <PlandexBlock> tag to start a code block and a closing </PlandexBlock> tag to end a code block.

The <PlandexBlock> tag MUST ONLY contain the code for the code block and NOTHING ELSE. Do NOT wrap the code block in triple backticks, CDATA tags, or any other text or formatting. Output ONLY the code and nothing else within the <PlandexBlock> tag.

You MUST follow the instructions you've been given on how to update code in code blocks:

- Do NOT include large sections of the file that are not changing. Output ONLY code that is changing and code that is necessary to understand the changes, the code structure, and where the changes should be applied. Use references comments for sections of the file that are not changing.

- Include enough code from the original file to precisely and unambiguously locate where the changes should be applied and their level of nesting.

- Match the indentation of the original file exactly.

- Do NOT output multiple references with no changes in between them.

- Do NOT add superfluous newlines around reference comments.

- Use a removal comment to denote code that is being removed from a file. As with reference comments, removal comments must be surrounded by enough context so that the location and nesting depth of the code being removed is clear and unambiguous.

- When replacing code from the original file with *new code*, you MUST make it unambiguously clear exactly which code is being replaced by including surrounding context.

- Unless you are fully overwriting the entire file, you ABSOLUTELY MUST ALWAYS include at least one "... existing code ..." comment before or after the change to account for all the code before or after the change.

- Even if the location of new code is not important and could be placed anywhere in the file, you still MUST determine *exactly* where the new code should be placed and include sufficient surrounding context so that the location and nesting depth of the code being added is clear and unambiguous.

- Never remove existing functionality unless explicitly instructed to do so.

- Show enough surrounding context to understand the code structure.

Every code block that is *updating* an existing file in context MUST ALWAYS be preceded by an explanation of the change that *exactly matches* one of the formats listed in the "### Action Explanation Format" section. Do *NOT* UNDER ANY CIRCUMSTANCES use an explanation like "I'll update the code to..." that does not match one of these formats.

- When outputting the explanation, do *NOT* insert code between two code structures that aren't *immediately adjacent* in the original file.

- When creating a *new* file, do NOT include this explanation. Include *one* explanation in this format per code block that *updates* an existing file. Do NOT include multiple explanations in the same code block.

Only list out subtasks once for the plan--after that, do not list or describe a subtask that can be implemented in code without including a code block that implements the subtask.

Do not implement a task partially and then give up even if it's very large or complex--do your best to implement each task and subtask **fully**.

Do NOT repeat any part of your previous response. Always continue seamlessly from where your previous response left off. 

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

If you break up a task into subtasks, only include subtasks that can be implemented directly in code by creating or updating files. Only include subtasks that require executing code or commands if execution mode is enabled. Do not include subtasks that require user testing, deployment, or other tasks that go beyond coding.

` + CurrentSubtaskPrompt + `

` + MarkSubtaskDonePrompt + `

` + FileOpsImplementationPromptSummary

var ImplementationPromptWrapperTokens int

func GetWrappedPrompt(prompt, osDetails, applyScriptSummary string, isPlanningStage bool) string {
	var promptWrapperFormatStr string
	if isPlanningStage {
		promptWrapperFormatStr = planningPromptWrapperFormatStr
	} else {
		promptWrapperFormatStr = implementationPromptWrapperFormatStr
	}

	// If we're in the planning stage, we don't need to include the apply script summary
	if isPlanningStage {
		applyScriptSummary = ""
	}

	ts := time.Now().Format(time.RFC3339)
	return fmt.Sprintf(promptWrapperFormatStr, prompt, ts, osDetails, applyScriptSummary)
}

const UserContinuePrompt = "Continue the plan."

const AutoContinuePrompt = `Continue the plan from where you left off in the previous response. Don't repeat any part of your previous response. Don't begin your response with 'Next,'. 

Continue seamlessly from where your previous response left off. 

Always name the subtask you are working on before starting it, and mark it as done before moving on to the next subtask.

` + CurrentSubtaskPrompt + `

` + MarkSubtaskDonePrompt + `

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

If you break up a task into subtasks, only include subtasks that can be implemented directly in code by creating or updating files. Only include subtasks that require executing code or commands if execution mode is enabled. Do not include subtasks that require user testing, deployment, or other tasks that go beyond coding. 

Do NOT include tests or documentation in the subtasks unless the user has specifically asked for them. Do not include extra code or features beyond what the user has asked for. Focus on the user's request and implement only what is necessary to fulfill it.`

var AutoContinuePromptTokens int

const SkippedPathsPrompt = "\n\nSome files have been skipped by the user and *must not* be generated. The user will handle any updates to these files themselves. Skip any parts of the plan that require generating these files. You *must not* generate a file block for any of these files.\nSkipped files:\n"

const CombineSubtasksPrompt = `
- Combine multiple steps into a single larger subtask where all of the steps are small enough to be completed in a single response (especially do this if multiple steps are closely related). Try to both size each subtask so that it can be completed in a single response, while also aiming to minimize the total number of subtasks. For subtasks involving multiple steps and/or multiple files, use bullet points to break them up into smaller sub-subtasks.

- When using bullet points to break up a subtask into multiple steps, make a note of any files that will be created or updated by each step—surround file paths with backticks like this: "` + "`path/to/some_file.txt`" + `". All paths mentioned in the bullet points of the subtask must be included in the 'Uses: ' list for the subtask.

- Do NOT break up file operations of the same type (e.g. moving files, removing files, resetting pending changes) into multiple subtasks. Group them all into a *single* subtask.
`
