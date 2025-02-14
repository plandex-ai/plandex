package prompts

import (
	"fmt"
	"time"
)

const SharedPromptWrapperFormatStr = "# The user's latest prompt:\n```\n%s\n```\n\n" + `Please respond according to the 'Your instructions' section above.

Do not ask the user to do anything that you can do yourself. Do not say a task is too large or complex for you to complete--do your best to break down the task and complete it even if it's very large or complex.

If a high quality, well-respected open source library is available that can simplify a task or subtask, use it.

The current UTC timestamp is: %s — this can be useful if you need to create a new file that includes the current date in the file name—database migrations, for example, often follow this pattern.

User's operating system details:
%s

---
%s
---
`

func GetPlanningPromptWrapperFormatStr(params CreatePromptParams) string {
	s := SharedPromptWrapperFormatStr + `

` + GetPlanningFlowControl(params) + `

Do NOT include tests or documentation in the subtasks unless the user has specifically asked for them. Do not include extra code or features beyond what the user has asked for. Focus on the user's request and implement only what is necessary to fulfill it.

` + ReviseSubtasksPrompt + `

` + CombineSubtasksPrompt + `

At the end of the '### Tasks' section, you ABSOLUTELY MUST ALWAYS include a <PlandexFinish/> tag, then end the response.

Example:
`

	if params.ExecMode {
		s += `
### Commands

The _apply.sh script is empty. I'll create it with commands to compile the project and run the new test with cargo.
`
	}

	s += `
### Tasks

1. Create a new file called 'src/main.rs' with a 'main' function that returns 'Hello, world!'
Uses: ` + "`src/main.rs`" + `

2. Write a basic test for the 'main' function
Uses: ` + "`src/main.rs`"

	if params.ExecMode {
		s += `
3. Run the new test with cargo
Uses: ` + "`src/main.rs`" + `
	`
	}

	s += `
<PlandexFinish/>

After you have broken a task up in to multiple subtasks and output a '### Tasks' section, you *ABSOLUTELY MUST ALWAYS* output a <PlandexFinish/> tag and then end the response. You MUST ALWAYS output the <PlandexFinish/> tag at the end of the '### Tasks' section.

Output a <PlandexFinish/> tag after the '### Tasks' section. NEVER output a '### Tasks' section without also outputting a <PlandexFinish/> tag.

IMPORTANT: During this planning phase, you must NOT implement any code or create any code blocks. Your only task is to break down the work into subtasks. Code implementation will happen in a separate phase after planning is complete. The planning phase is ONLY for breaking the work into subtasks.

Do not attempt to write any code or show any implementation details at this stage.
`

	return s
}

func GetImplementationPromptWrapperFormatStr(params CreatePromptParams) string {
	s := SharedPromptWrapperFormatStr + `

If you're making a plan, remember to label code blocks with the file path *exactly* as described in point 2, and do not use any other formatting for file paths. **Do not include explanations or any other text apart from the file path in code block labels.**

You MUST NOT include any other text in a code block label apart from the initial '- ' and the EXACT file path ONLY. DO NOT UNDER ANY CIRCUMSTANCES use a label like 'File path: src/main.rs' or 'src/main.rs: (Create this file)' or 'File to Create: src/main.rs' or 'File to Update: src/main.rs'. Instead use EXACTLY 'src/main.rs:'. DO NOT include any explanatory text in the code block label like 'src/main.rs: (Add a new function)'. It is EXTREMELY IMPORTANT that the code block label includes *only* the initial '- ', the file path, and NO OTHER TEXT whatsoever. If additional text apart from the initial '- ' and the exact file path is included in the code block label, the plan will not be parsed properly and you will have failed at the task of generating a usable plan. 

Always use an opening <PlandexBlock> tag to start a code block and a closing </PlandexBlock> tag to end a code block.

The <PlandexBlock> tag content MUST ONLY contain the code for the code block and NOTHING ELSE. Do NOT wrap the code block in triple backticks, CDATA tags, or any other text or formatting. Output ONLY the code and nothing else within the <PlandexBlock> tag.

The <PlandexBlock> tag MUST ALWAYS include both a 'lang' attribute and a 'path' attribute as described in the instructions above. It must not include any other attributes.

When *updating an existing file*, you MUST follow the instructions you've been given on how to update code in code blocks:

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

	- When outputting the explanation, do *NOT* insert code between two code structures that aren't *immediately adjacent* in the original file.

  -	Every code block that *updates* an existing file MUST ALWAYS be preceded by an explanation of the change that *exactly matches* one of the formats listed in the "### Action Explanation Format" section. Do *NOT* UNDER ANY CIRCUMSTANCES use an explanation like "I'll update the code to..." that does not match one of these formats.

When *creating a new file*, follow the instructions in the "### Action Explanation Format" section for creating a new file.
 
  - The Type field MUST be exactly 'new file'.
  - The Summary field MUST briefly describe the new file and its purpose.
	- The file path MUST be included in the code block label.
	- The code itself MUST be written within a <PlandexBlock> tag.
	- The <PlandexBlock> tag MUST include both a 'lang' attribute and a 'path' attribute as described in the instructions above. It must not include any other attributes.
	- The <PlandexBlock> tag MUST NOT include any other text or formatting. It must only contain the code for the code block and NOTHING ELSE. Do NOT wrap the code block in triple backticks, CDATA tags, or any other text or formatting. Output ONLY the code and nothing else within the <PlandexBlock> tag.
	- The code block MUST include the *entire file* to be created. Do not omit any code from the file.
	- Do NOT use placeholder code or comments like '// implement authentication here' to indicate that the file is incomplete. Implement *all* functionality.
	- Do NOT use reference comments like '// ... existing code ...'. Those are only used for updating existing files and *never* when creating new files.
	- Include the *entire file* in the code block.


Only list out subtasks once for the plan--after that, do not list or describe a subtask that can be implemented in code without including a code block that implements the subtask.

Do not implement a task partially and then give up even if it's very large or complex--do your best to implement each task and subtask **fully**.

Do NOT repeat any part of your previous response. Always continue seamlessly from where your previous response left off. 

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

If you break up a task into subtasks, only include subtasks that can be implemented directly in code by creating or updating files. Only include subtasks that require executing code or commands if execution mode is enabled. Do not include subtasks that require user testing, deployment, or other tasks that go beyond coding.

` + CurrentSubtaskPrompt + `

` + MarkSubtaskDonePrompt + `

` + FileOpsImplementationPromptSummary

	return s
}

func GetFollowUpRequiredPrompt(params CreatePromptParams) string {
	s := `
[MANDATORY FOLLOW-UP FLOW]

CRITICAL FLOW CONTROL:
1. You MUST FIRST respond naturally to what the user has said/asked
2. Then classify the prompt as either:
   A. Update/revision to tasks (A1/A2/A3)
   B. Conversation prompt (question/comment)

3. IF classified as A (update/revision):
   - You MUST create/update the task list with ### Tasks
   - You MUST output <PlandexFinish/> immediately after the task list
   - You MUST end your response immediately after <PlandexFinish/>
   - You ABSOLUTELY MUST NOT proceed to implementation
   - You MUST follow planning format exactly
   Even if:
   - The change is small
   - You know the exact code to write
   - You're continuing an existing plan

4. IF classified as B (conversation):
   - Continue conversation naturally
   - Do not create tasks or implement code

5. After responding and classifying, output EXACTLY ONE of these statements (naturally incorporated):
   A. "I have the context I need to continue."
   B. "I have the context I need to respond."
   C. "I need more context to continue. <PlandexFinish/>"
   D. "I need more context to respond. <PlandexFinish/>"
   E. "This is a significant update to the plan. I'll clear all context without pending changes, then decide what context I need to move forward. <PlandexFinish/>"
   F. "This is a new task that is distinct from the plan. I'll clear all context without pending changes, then decide what context I need to move forward. <PlandexFinish/>"

For statements A/B: You may rephrase naturally while keeping the meaning.
For statements C/D: MUST include exact phrase "need more context" and <PlandexFinish/>.
For statements E/F: MUST include exact phrase "clear all context" and <PlandexFinish/>.

CRITICAL: Always respond naturally to the user first, then seamlessly incorporate the required statement. Do NOT state that you are performing a classification or context assessment.
`

	return s
}

type UserPromptParams struct {
	CreatePromptParams
	Prompt             string
	OsDetails          string
	IsPlanningStage    bool
	IsFollowUp         bool
	ApplyScriptSummary string
}

func GetWrappedPrompt(params UserPromptParams) string {

	prompt := params.Prompt
	osDetails := params.OsDetails
	isPlanningStage := params.IsPlanningStage
	isFollowUp := params.IsFollowUp
	applyScriptSummary := params.ApplyScriptSummary

	var promptWrapperFormatStr string
	if isPlanningStage {
		promptWrapperFormatStr = GetPlanningPromptWrapperFormatStr(params.CreatePromptParams)
	} else {
		promptWrapperFormatStr = GetImplementationPromptWrapperFormatStr(params.CreatePromptParams)
	}

	// If we're in the planning stage, we don't need to include the apply script summary
	if isPlanningStage {
		applyScriptSummary = ""

		if isFollowUp {
			promptWrapperFormatStr += "\n\n" + GetFollowUpRequiredPrompt(params.CreatePromptParams)
		}
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

const SkippedPathsPrompt = "\n\nSome files have been skipped by the user and *must not* be generated. The user will handle any updates to these files themselves. Skip any parts of the plan that require generating these files. You *must not* generate a file block for any of these files.\nSkipped files:\n"

const CombineSubtasksPrompt = `
- Combine multiple steps into a single larger subtask where all of the steps are small enough to be completed in a single response (especially do this if multiple steps are closely related). Try to both size each subtask so that it can be completed in a single response, while also aiming to minimize the total number of subtasks. For subtasks involving multiple steps and/or multiple files, use bullet points to break them up into smaller sub-subtasks.

- When using bullet points to break up a subtask into multiple steps, make a note of any files that will be created or updated by each step—surround file paths with backticks like this: "` + "`path/to/some_file.txt`" + `". All paths mentioned in the bullet points of the subtask must be included in the 'Uses: ' list for the subtask.

- Do NOT break up file operations of the same type (e.g. moving files, removing files, resetting pending changes) into multiple subtasks. Group them all into a *single* subtask.

- Keep subtasks focused and manageable. While it's fine to group closely related changes (like small updates to a few tightly coupled files) into a single subtask, prefer breaking work into smaller, more focused subtasks when the changes are more substantial or independent. If a subtask involves many files or multiple distinct changes, consider whether it would be clearer and more maintainable to break it into multiple subtasks.

Here are examples of good and poor task division:

Example 1 - Poor (tasks too small and fragmented):
1. Create the product.js file
Uses: ` + "`src/models/product.js`" + `

2. Add the product schema
Uses: ` + "`src/models/product.js`" + `

3. Add the validate() method
Uses: ` + "`src/models/product.js`" + `

4. Add the save() method
Uses: ` + "`src/models/product.js`" + `

Better:
1. Create product model with core functionality
- Create product.js with schema definition
- Add validate() and save() methods
Uses: ` + "`src/models/product.js`" + `

Example 2 - Poor (task too large with unrelated changes):
1. Implement user profile features
- Add user avatar upload
- Add profile settings page
- Implement friend requests
- Add user search
- Create notification system
Uses: ` + "`src/components/Profile.tsx`" + `, ` + "`src/components/Avatar.tsx`" + `, ` + "`src/components/Settings.tsx`" + `, ` + "`src/services/friends.ts`" + `, ` + "`src/services/search.ts`" + `, ` + "`src/services/notifications.ts`" + `

Better:
1. Implement user avatar upload functionality
- Add avatar component with upload UI
- Add avatar upload service
Uses: ` + "`src/components/Avatar.tsx`" + `, ` + "`src/services/avatar.ts`" + `

2. Create profile settings page
- Add settings form components
- Implement save/load settings
Uses: ` + "`src/components/Settings.tsx`" + `, ` + "`src/services/settings.ts`" + `

3. Add friend request system
Uses: ` + "`src/services/friends.ts`" + `, ` + "`src/components/Profile.tsx`" + `

Example 3 - Good (related changes properly grouped):
1. Update error handling in authentication flow
- Add error handling to login function
- Add corresponding error states in auth context
- Update error display in login form
Uses: ` + "`src/auth/login.ts`" + `, ` + "`src/context/auth.tsx`" + `, ` + "`src/components/LoginForm.tsx`" + `

Example 4 - Good (tightly coupled file updates):
1. Rename UserType enum to AccountType
- Update enum definition
- Update all imports and usages
Uses: ` + "`src/types/user.ts`" + `, ` + "`src/auth/account.ts`" + `, ` + "`src/components/UserProfile.tsx`" + `

Notice in these examples:
- Tasks that are too granular waste responses on tiny changes
- Tasks that are too large mix unrelated changes and become hard to implement
- Good tasks group related changes that make sense to implement together
- Multiple files can be included when the changes are tightly coupled
- Bullet points describe steps in a cohesive change, not separate features
`

type ChatUserPromptParams struct {
	CreatePromptParams
	Prompt    string
	OsDetails string
}

func GetWrappedChatOnlyPrompt(params ChatUserPromptParams) string {
	// Base wrapper that's always included
	baseWrapper := "# The user's latest prompt:\n```\n%s\n```\n\n" + `Please respond according to the 'Your instructions' section above.

The current UTC timestamp is: %s

User's operating system details:
%s`

	// Build additional instructions based on parameter combinations
	var additionalInstructions string

	// Context handling - different for each autoContext + lastResponseLoadedContext combination
	if params.AutoContext {
		if params.LastResponseLoadedContext {
			additionalInstructions += `

Since you just loaded context in your previous response:
- Focus on using that context in your explanation
- Keep the conversation flowing naturally
- You ABSOLUTELY MUST NOT load additional context unless the user asks about something completely different
- Maintain conversational flow over seeking more context`
		} else {
			additionalInstructions += `

When handling context:
- Load context only when needed for accuracy
- Make the context loading feel natural and conversational
- If you need to check files, briefly mention what you're looking at
- Once you've loaded context, use it thoroughly in your response`
		}
	} else {
		additionalInstructions += `

When discussing code:
- Work with the context explicitly provided
- If you need additional context, ask the user specifically what files would help
- Make full use of any context you already have
- Be clear when you need more information to provide a complete answer`
	}

	// Execution mode handling
	if params.ExecMode {
		additionalInstructions += `

Regarding execution capabilities:
- You can discuss both file changes and command execution
- Be specific about what commands would need to be run
- Consider build processes, testing, and deployment
- Distinguish between file changes and execution steps`
	} else {
		additionalInstructions += `

Remember about execution mode:
- Focus on changes that can be made through file updates
- Mention when something would require execution mode
- You can discuss build/test/deploy conceptually
- Be clear when certain steps would need execution mode enabled`
	}

	// Always include these key reminders
	// Always include these key reminders
	additionalInstructions += `
Keep in mind:
- Stay conversational while being technically precise
- Reference and explain code when helpful, but don't output formal implementation blocks
- Focus on what's specifically asked - don't suggest extra features
- Consider existing codebase structure in your explanations
- When discussing libraries, focus on well-maintained, widely-used options
- If the user wants to implement changes, remind them about 'tell mode'
- Use error handling, logging, and security best practices in your suggestions
- Be thoughtful about code organization and structure
- Consider implications of suggested changes on the existing codebase

Remember you're in chat mode:
- Engage in natural technical discussion about code and context
- Help users understand their codebase and plan potential changes
- Provide explanations and answer questions thoroughly
- Include code snippets only when they help explain concepts
- Help debug issues by examining and explaining code
- Suggest approaches and discuss trade-offs
- Help evaluate different implementation strategies
- Consider and explain implications of different approaches
- Stay focused on understanding and planning rather than implementation

You cannot:
- Create or modify any files
- Output formal implementation code blocks
- Make plans using "### Tasks" sections
- Structure responses as if implementing changes
- Load context multiple times in consecutive responses
- Switch to implementation mode without user request

Even if a plan is in progress:
- Stay in discussion mode, don't attempt to implement anything
- You can discuss the current tasks and progress
- You can provide explanations and suggestions
- You can help debug issues or clarify approach
- But you must not output any implementation code
- Return to implementation only when user switches back to tell mode

Remember that users often:
- Switch between chat and tell mode during implementation
- Use chat mode to understand before implementing
- Need detailed technical discussion to plan effectively
- Want to explore options before committing to changes
- May need to debug or understand issues mid-implementation
- You may receive a list of tasks that are in progress, including a 'current subtask'. You MUST NOT implement any tasks—only discuss them.
`

	promptWrapperFormatStr := baseWrapper + additionalInstructions

	ts := time.Now().Format(time.RFC3339)
	return fmt.Sprintf(promptWrapperFormatStr,
		params.Prompt,
		ts,
		params.OsDetails)
}

func GetPlanningFlowControl(params CreatePromptParams) string {
	s := `
CRITICAL PLANNING RULES:
1. For ANY update/revision to tasks:
`

	if params.ExecMode {
		s += `You MUST output a ### Commands section before the ### Tasks list`
	}

	s += `
   - You MUST output a new/updated ### Tasks list
   - You MUST NOT start implementing code
   - You MUST follow planning phase format exactly

2. Even for small changes:
   - Create/update task list first
   - No implementation until planning is complete
   - All changes must be in task list

3. Task Planning Required:
   - When adding new functionality
   - When modifying existing code
   - When fixing bugs or issues
   - When continuing an existing plan
   NO EXCEPTIONS - always plan first
`

	return s
}
