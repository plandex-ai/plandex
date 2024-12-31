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

STOP THE RESPONSE IMMEDIATELY IF YOU HAVE OUTPUTTED A '### Tasks' SECTION. Do NOT continue on and begin implementing subtasks. You will begin on subtasks in the next response, not this one.

Do NOT output any additional text after the '### Tasks' section.
`

var PlanningPromptWrapperTokens int

const implementationPromptWrapperFormatStr = sharedPromptWrapperFormatStr + `

If you're making a plan, remember to label code blocks with the file path *exactly* as described in point 2, and do not use any other formatting for file paths. **Do not include explanations or any other text apart from the file path in code block labels.**

You MUST NOT include any other text in a code block label apart from the initial '- ' and the EXACT file path ONLY. DO NOT UNDER ANY CIRCUMSTANCES use a label like 'File path: src/main.rs' or 'src/main.rs: (Create this file)' or 'File to Create: src/main.rs' or 'File to Update: src/main.rs'. Instead use EXACTLY 'src/main.rs:'. DO NOT include any explanatory text in the code block label like 'src/main.rs: (Add a new function)'. It is EXTREMELY IMPORTANT that the code block label includes *only* the initial '- ', the file path, and NO OTHER TEXT whatsoever. If additional text apart from the initial '- ' and the exact file path is included in the code block label, the plan will not be parsed properly and you will have failed at the task of generating a usable plan. 

Always use an opening <PlandexBlock> tag to start a code block and a closing </PlandexBlock> tag to end a code block.

The <PlandexBlock> tag MUST ONLY contain the code for the code block and NOTHING ELSE. Do NOT wrap the code block in triple backticks, CDATA tags, or any other text or formatting. Output ONLY the code and nothing else within the <PlandexBlock> tag.

` + UpdateFormatPrompt + `

` + ChangeExplanationPrompt + `

Only list out subtasks once for the plan--after that, do not list or describe a subtask that can be implemented in code without including a code block that implements the subtask.

Do not implement a task partially and then give up even if it's very large or complex--do your best to implement each task and subtask **fully**.

Do NOT repeat any part of your previous response. Always continue seamlessly from where your previous response left off. 

Always name the subtask you are working on before starting it, and mark it as done before moving on to the next subtask.

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

If you break up a task into subtasks, only include subtasks that can be implemented directly in code by creating or updating files. Do not include subtasks that require executing code or commands. Do not include subtasks that require user testing, deployment, or other tasks that go beyond coding.

` + FileOpsPromptSummary

var ImplementationPromptWrapperTokens int

func GetWrappedPrompt(prompt, osDetails, applyScriptSummary string, isPlanningStage bool) string {
	var promptWrapperFormatStr string
	if isPlanningStage {
		promptWrapperFormatStr = planningPromptWrapperFormatStr
	} else {
		promptWrapperFormatStr = implementationPromptWrapperFormatStr
	}

	ts := time.Now().Format(time.RFC3339)
	return fmt.Sprintf(promptWrapperFormatStr, prompt, ts, osDetails, applyScriptSummary)
}

const UserContinuePrompt = "Continue the plan."

const AutoContinuePrompt = `Continue the plan from where you left off in the previous response. Don't repeat any part of your previous response. Don't begin your response with 'Next,'. 

Continue seamlessly from where your previous response left off. 

Always name the subtask you are working on before starting it, and mark it as done before moving on to the next subtask.

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

If you break up a task into subtasks, only include subtasks that can be implemented directly in code by creating or updating files. Do not include subtasks that require executing code or commands. Do not include subtasks that require user testing, deployment, or other tasks that go beyond coding. 

Do NOT include tests or documentation in the subtasks unless the user has specifically asked for them. Do not include extra code or features beyond what the user has asked for. Focus on the user's request and implement only what is necessary to fulfill it.`

var AutoContinuePromptTokens int

const SkippedPathsPrompt = "\n\nSome files have been skipped by the user and *must not* be generated. The user will handle any updates to these files themselves. Skip any parts of the plan that require generating these files. You *must not* generate a file block for any of these files.\nSkipped files:\n"
