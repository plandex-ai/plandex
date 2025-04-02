package prompts

import (
	"fmt"
	shared "plandex-shared"
	"time"
)

const SharedPromptWrapperFormatStr = "# The user's latest prompt:\n```\n%s\n```\n\n" + `Please respond according to the 'Your instructions' section above.

Do not ask the user to do anything that you can do yourself. Do not say a task is too large or complex for you to complete--do your best to break down the task and complete it even if it's very large or complex.

If a high quality, well-respected open source library is available that can simplify a task or subtask, use it.

The current UTC timestamp is: %s — this can be useful if you need to create a new file that includes the current date in the file name—database migrations, for example, often follow this pattern.

Do NOT create or update a binary image file, audio file, video file, or any other binary media file using code blocks. You can create svg files if appropriate since they are text-based, but do NOT create or update other image files like png, jpg, gif, or jpeg, or audio files like mp3, wav, or m4a.

User's operating system details:
%s

---
%s
---
`

func GetContextLoadingPromptWrapperFormatStr(params CreatePromptParams) string {
	s := SharedPromptWrapperFormatStr + `
	` + GetArchitectContextSummary(params.ContextTokenLimit)

	return s
}

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
3. 🚀 Run the new test with cargo
Uses: ` + "`_apply.sh`" + `
	`
	}

	s += `
<PlandexFinish/>

After you have broken a task up in to multiple subtasks and output a '### Tasks' section, you *ABSOLUTELY MUST ALWAYS* output a <PlandexFinish/> tag and then end the response. You MUST ALWAYS output the <PlandexFinish/> tag at the end of the '### Tasks' section.

Output a <PlandexFinish/> tag after the '### Tasks' section. NEVER output a '### Tasks' section without also outputting a <PlandexFinish/> tag.

Use your judgment on the paths of new files you create. Keep directories well organized and if you're working in an existing project, follow existing patterns in the codebase. ALWAYS use *complete* *relative* paths for new files.

Modular Project Structure: When creating new files for a project or feature, prioritize modularity and separation of concerns by creating separate files for each component/responsibility area, even if everything could initially fit in one file.

Ongoing File Management: If a file you initially created grows complex or tightly couples different responsibilities, progressively break it into smaller, more focused files rather than letting it become monolithic.

Forward-Thinking Design: Organize code to accommodate growth and evolution, following language conventions while keeping files small, focused, and maintainable.

IMPORTANT: During this planning phase, you must NOT implement any code or create any code blocks. Your ONLY JOB is to break down the work into subtasks. Code implementation will happen in a separate phase after planning is complete. The planning phase is ONLY for breaking the work into subtasks.

Do not attempt to write any code or show any implementation details at this stage.

The MOST IMPORTANT THING to remember is that you are in the PLANNING phase. Even though you see examples of implementation in your conversation history, you MUST NOT do any implementation at this stage. Your ONLY JOB is to make a plan and output a list of tasks, even if there is only *one* task in your list. That is your ONLY JOB at this stage. It may seem more natural to just respond to the user with code for small tasks, but it is ABSOLUTELY CRITICAL that you devote sufficient attention that you never make this mistake. It is critical that you have a 100%% success rate at giving correct output according to the stage.
`

	if params.IsUserDebug {
		s += UserPlanningDebugPrompt
	} else if params.IsApplyDebug {
		s += ApplyPlanningDebugPrompt
	} else if !params.ExecMode {
		s += NoApplyScriptPlanningPrompt
	}

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

	- Do NOT include large sections of the file that are not changing. Output ONLY code that is changing and code that is necessary to understand the changes, the code structure, and where the changes should be applied. Use references comments for sections of the file that are not changing. ONLY use exactly '... existing code ...' (with appropriate comment symbol(s) for the language) for reference comments—no other variations are allowed.

	- Include enough code from the original file to precisely and unambiguously locate where the changes should be applied and their level of nesting.

	- Match the indentation of the original file exactly.

	- Do NOT include line numbers in the <PlandexBlock> tag. While line numbers are included in the original file in context (prefixed with 'pdx-', like 'pdx-10: ') in context to assist you with describing the location of changes in the 'Action Explanation', they ABSOLUTELY MUST NOT be included in the <PlandexBlock> tag.

	- Do NOT output multiple references with no changes in between them.

	- Do NOT add superfluous newlines around reference comments.

	- Use a removal comment to denote code that is being removed from a file. As with reference comments, removal comments must be surrounded by enough context so that the location and nesting depth of the code being removed is clear and unambiguous.

	- When replacing code from the original file with *new code*, you MUST make it unambiguously clear exactly which code is being replaced by including surrounding context.

	- Unless you are fully overwriting the entire file, you ABSOLUTELY MUST ALWAYS include at least one "... existing code ..." comment before or after the change to account for all the code before or after the change.

	- Even if the location of new code is not important and could be placed anywhere in the file, you still MUST determine *exactly* where the new code should be placed and include sufficient surrounding context so that the location and nesting depth of the code being added is clear and unambiguous.

	- Never remove existing functionality unless explicitly instructed to do so.

	- DO NOT remove comments, logging statements, code that is commented out, or ANY code that is not related to the specific task at hand.

	- Do NOT escape newlines within the <PlandexBlock> tag unless there is a specific reason to do so, like you are outputting newlines in a quoted JSON string. For normal code, do NOT escape newlines.
	
	- Strive to make changes that are minimally intrusive and do not change the existing code beyond what is necessary to complete the task.

	- Show enough surrounding context to understand the code structure.

	- When outputting the explanation, do *NOT* insert code between two code structures that aren't *immediately adjacent* in the original file.

  -	Every code block that *updates* an existing file MUST ALWAYS be preceded by an explanation of the change that *exactly matches* one of the formats listed in the "### Action Explanation Format" section. Do *NOT* UNDER ANY CIRCUMSTANCES use an explanation like "I'll update the code to..." that does not match one of these formats.

	- If you are replacing or removing code, you MUST include an exhaustive list of all symbols/sections that are being removed—ALL removed code must be accounted for. That MUST be followed by a line number range of lines in the original file that are being replaced. Use the exact format: '(original file lines [startLineNumber]-[endLineNumber])' — e.g. '(original file lines 10-20)' or for a single line, '(original file line [lineNumber])' — e.g. '(original file line 10)'

	- CRITICAL: When writing the Context field in an Action Explanation:
		- The symbols/structures mentioned MUST be code that is NOT being changed
		- These symbols serve as ANCHORS to precisely locate where the change should be applied
		- Every symbol/structure mentioned in the Context MUST appear in the code block
		- These anchors MUST be immediately adjacent to where the change occurs
		- Do NOT use distant symbols with other code between them and the change
		- All symbols must be surrounded with backticks
		- The code block MUST include these anchors to unambiguously locate the change
		- If you mention "Located between ` + "`functionA`" + "` and `" + "`functionB`" + `, both functions MUST appear in your code block

		FAILURE TO INCLUDE THE CONTEXT SYMBOLS IN THE CODE BLOCK MAKES CHANGES IMPOSSIBLE TO APPLY CORRECTLY AND IS A CRITICAL ERROR.

When *creating a new file*, follow the instructions in the "### Action Explanation Format" section for creating a new file.
 
  - The Type field MUST be exactly 'new file'.
  - The Summary field MUST briefly describe the new file and its purpose.
	- The file path MUST be included in the code block label.
	- The code itself MUST be written within a <PlandexBlock> tag.
	- The <PlandexBlock> tag MUST include both a 'lang' attribute and a 'path' attribute as described in the instructions above. It must not include any other attributes.
	- The <PlandexBlock> tag MUST NOT include any other text or formatting. It must only contain the code for the code block and NOTHING ELSE. Do NOT wrap the code block in triple backticks, CDATA tags, or any other text or formatting. Output ONLY the code and nothing else within the <PlandexBlock> tag.
	- The code block MUST include the *entire file* to be created. Do not omit any code from the file.
	- Do NOT use placeholder code or comments like '// implement authentication here' to indicate that the file is incomplete. Implement *all* functionality.
	- Do NOT use reference comments ('// ... existing code ...'). Those are only used for updating existing files and *never* when creating new files.
	- Include the *entire file* in the code block.


If multiple changes are being made to the same file in a single subtask, you MUST ALWAYS combine them into a SINGLE code block. Do NOT use multiple code blocks for multiple changes to the same file. Instead:

	- Include all changes in a single code block that follows the file's structure
	- Use "... existing code ..." comments between changes
	- Show enough context around each change for unambiguous location
	- Maintain the original file's order of elements
	- Only reproduce parts of the file necessary to show structure and locate changes
	- Make all changes in a single pass from top to bottom of the file

	When writing the explanation for multiple changes that will be included in a single code block, list each change independently like this:

	**Updating  + "server/handlers/auth.go" + **
	Change 1. 
		Type: remove
		Summary: Remove unused  + "validateLegacyTokens" +  function and its helper  +    "checkTokenFormat" + . Removes  + "validateLegacyTokens and checkTokenFormat" +  functions (original file lines 25-85).
		Context: Located between  + "parseAuthHeader" +  and  + "validateJWT" +  functions
	Change 2.
		Type: append
		Summary: Append just-removed + "checkTokenFormat" + function to the end of the file"	


Only list out subtasks once for the plan--after that, do not list or describe a subtask that can be implemented in code without including a code block that implements the subtask.

Do not implement a task partially and then give up even if it's very large or complex--do your best to implement each task and subtask **fully**.

Do NOT repeat any part of your previous response. Always continue seamlessly from where your previous response left off. 

ALWAYS complete subtasks in order and never go backwards in the list of subtasks. Never skip a subtask or work on subtasks out of order. Never repeat a subtask that has been marked implemented in the latest summary or that has already been implemented during conversation.

` + CurrentSubtaskPrompt + `

` + MarkSubtaskDonePrompt + `

` + FileOpsImplementationPromptSummary

	file := ".gitignore"
	if !params.IsGitRepo {
		file = ".plandexignore"
	}

	s += fmt.Sprintf(`
- Create or update the %s file if necessary.
- If you write commands to _apply.sh, consider if output should be added to %s.
`, file, file)

	s += `
## Is the task done or in progress?

Remember, you must follow these instructions on marking tasks as done or in progress:

- When a subtask is *completed*, you *must* either: 

1. Mark it as 'done' in the format described in the 'Marking Tasks as Done Or In Progress' section.
2. Mark it as 'in progress' by explaining that the task is not yet complete and will be continued in the next response.

Remember, you must WAIT until the subtask is *fully implemented* before marking it as done. If a subtask is large, this may require multiple responses. If you have only implemented part of a subtask, do NOT mark it as done. It will be continued in one or more subsequent responses, and the last one of those reponses will mark the subtask as done. If you mark the subtask done prematurely, you will stop it from being fully implemented, which will prevent the plan from being implemented correctly.

## The Most Critical Factor

Remember, the MOST critical factor in creating code blocks correctly is to locate them unambiguously in the file using the definitions that are immediately before and immediately after the the section of code that is being changed or extended. Pay special attention to the 'Context' field in the Action Explanation. ALWAYS include at least a few additional lines of code before and after the section that is changing. And even if you need to include many lines to reach the *definitions* that are immediately before and after the section that is changing, do so.

Definitions in the original file that are outside of the section that is changing are like "hooks" that determine where in the resulting file the new code you write will be placed.

This is why it's critical for you to ALWAYS include enough immediately surrounding code to unambiguously locate ALL the new code you write. All the blocks of new code you write must hook in correctly using the hooks you supply from the original file when you include additional lines of code from the original file before and after the section that is changing.

Even though you should include the definitions before and after the section, don't reproduce large sections of the original file. Use '... existing code ...' reference comments to 'collapse' large sections of the original file that are not changing.

It's not easy to be 100% consistent in writing code blocks that follow these rules, but you are capable of doing it with sufficient attention.

This disambiguation technique is the *most important* part of correctly implementing a plan.
`

	return s
}

type UserPromptParams struct {
	CreatePromptParams
	Prompt                     string
	OsDetails                  string
	CurrentStage               shared.CurrentStage
	UnfinishedSubtaskReasoning string
}

func GetWrappedPrompt(params UserPromptParams) string {
	currentStage := params.CurrentStage

	prompt := params.Prompt
	osDetails := params.OsDetails

	var promptWrapperFormatStr string
	if currentStage.TellStage == shared.TellStagePlanning {
		if currentStage.PlanningPhase == shared.PlanningPhaseContext {
			promptWrapperFormatStr = GetContextLoadingPromptWrapperFormatStr(params.CreatePromptParams)
		} else {
			promptWrapperFormatStr = GetPlanningPromptWrapperFormatStr(params.CreatePromptParams)
		}
	} else {
		promptWrapperFormatStr = GetImplementationPromptWrapperFormatStr(params.CreatePromptParams)
	}

	// If we're in the context loading stage, we don't need to include the apply script summary
	var applyScriptSummary string
	if currentStage.TellStage == shared.TellStagePlanning && currentStage.PlanningPhase == shared.PlanningPhaseTasks {
		applyScriptSummary = ApplyScriptPlanningPromptSummary
	} else if currentStage.TellStage == shared.TellStageImplementation {
		applyScriptSummary = ApplyScriptImplementationPromptSummary
	}

	ts := time.Now().Format(time.RFC3339)

	s := "The current stage is: "
	if currentStage.TellStage == shared.TellStagePlanning {
		if currentStage.PlanningPhase == shared.PlanningPhaseContext {
			s += "CONTEXT"
		} else {
			s += "PLANNING"
		}
	} else if currentStage.TellStage == shared.TellStageImplementation {
		s += "IMPLEMENTATION"
	}
	s += "\n\n"
	s += fmt.Sprintf(promptWrapperFormatStr, prompt, ts, osDetails, applyScriptSummary)

	if currentStage.TellStage == shared.TellStageImplementation && params.UnfinishedSubtaskReasoning != "" {
		s += "\n\n" + `
The current task was not completed in the previous response and remains unfinished. Here is the reasoning for why it was not completed:

` + params.UnfinishedSubtaskReasoning + `

You MUST address these issues in the next response and ensure the task is fully completed. You MUST continue working on the current task until it is fully completed. Do NOT work on any other tasks. If you are able to finish it in this response, state explicitly that the task is finished as described in your instructions. If not, state what you have finished and what remains to be done—it will be finished in a later response.
		`

	}

	return s
}

const UserContinuePrompt = "Continue the plan according to your instructions for the current stage. Don't repeat any part of your previous response."

const AutoContinuePlanningPrompt = UserContinuePrompt

const AutoContinueImplementationPrompt = `Continue the plan from where you left off in the previous response. Don't repeat any part of your previous response. 

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

	// Execution mode handling
	if params.ExecMode {
		additionalInstructions += `
*Execution mode is enabled.*
- If you switch to tell mode, you can execute commands locally as needed
- While you remain in chat mode, you can discuss both file changes and command execution, but you cannot update files or execute commands (unless the user first switches to tell mode)
- Be specific about what commands would need to be run
- Consider build processes, testing, and deployment
- Distinguish between file changes and execution steps`
	} else {
		additionalInstructions += `
*Execution mode is disabled.*
- If you switch to tell mode, you cannot execute commands—keep this in mind when discussing the plan. If the plan requires commands to be run after switching to tell mode, the user would need to run them manually.
- You can discuss build/test/deploy conceptually, but you cannot execute commands either in chat mode or in tell mode
- Be clear when certain steps would need execution mode enabled`
	}

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
		s += `You MUST output a ### Commands section before the ### Tasks list. If you determine that commands should be added or updated in _apply.sh, you MUST include wording like "I'll add this step to the plan" and then include a subtask referencing _apply.sh in the ### Tasks list.`
	}

	s += `
   - You MUST output a new/updated ### Tasks list
	`

	if params.ExecMode {
		s += `
   - If the ### Commands section indicates that commands should be added or updated in _apply.sh, you MUST also create a subtask referencing _apply.sh in the ### Tasks list
	`
	}

	s += `
   - You MUST NOT UNDER ANY CIRCUMSTANCES start implementing code, even if you have already made a plan in a previous response and are ready to implement it—you still ABSOLUTELY MUST NOT implement code at this stage. You MUST make a plan first in the format described above.
   - You MUST follow planning phase format exactly

2. Even for small changes:
   - Create/update task list first
   - No implementation and NO CODE until planning is complete, and you have output a '### Tasks' section and a <PlandexFinish/>
   - All changes must be in task list

3. The planning stage is *ALWAYS* required. You MUST NEVER skip ahead and start writing code in this response. You MUST complete the planning stage first and output a '### Tasks' section and a <PlandexFinish/> before you can start implementing code.
`

	return s
}

// func GetFollowUpRequiredPrompt(params CreatePromptParams) string {
// 	s := `
// [MANDATORY FOLLOW-UP FLOW]

// CRITICAL FLOW CONTROL:
// 1. You MUST FIRST respond naturally to what the user has said/asked
// 2. Then classify the prompt as either:
//    A. Update/revision to tasks (A1/A2/A3)
//    B. Conversation prompt (question/comment)

// 3. IF classified as A (update/revision):
//    - You MUST create/update the task list with ### Tasks
//    - You MUST output <PlandexFinish/> immediately after the task list
//    - You MUST end your response immediately after <PlandexFinish/>
//    - You ABSOLUTELY MUST NOT proceed to implementation
//    - You MUST follow planning format exactly
//    Even if:
//    - The change is small
//    - You know the exact code to write
//    - You're continuing an existing plan

// 4. IF classified as B (conversation):
//    - Continue conversation naturally
//    - Do not create tasks or implement code

// 5. After responding and classifying, output EXACTLY ONE of these statements (naturally incorporated):
//    A. "I have the context I need to continue."
//    B. "I have the context I need to respond."
//    C. "I need more context to continue. <PlandexFinish/>"
//    D. "I need more context to respond. <PlandexFinish/>"
//    E. "This is a significant update to the plan. I'll clear all context without pending changes, then decide what context I need to move forward. <PlandexFinish/>"
//    F. "This is a new task that is distinct from the plan. I'll clear all context without pending changes, then decide what context I need to move forward. <PlandexFinish/>"

// For statements A/B: You may rephrase naturally while keeping the meaning.
// For statements C/D: MUST include exact phrase "need more context" and <PlandexFinish/>.
// For statements E/F: MUST include exact phrase "clear all context" and <PlandexFinish/>.

// CRITICAL: Always respond naturally to the user first, then seamlessly incorporate the required statement. Do NOT state that you are performing a classification or context assessment.
// `

// 	return s
// }
