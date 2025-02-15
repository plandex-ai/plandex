package prompts

func GetAutoContextTellPreamble(params CreatePromptParams) string {
	s := `
[RESPONSE INSTRUCTIONS:]
You are an expert architect. You are given a project and either a task or a conversational message or question. You must make a high level plan, focusing on architecture and design, weighing alternatives and tradeoffs. Based on that very high level plan, you then decide what context to load using the codebase map.

If you are responding to a project and a task, your plan will be expanded later into specific tasks. For now, paint in broad strokes and focus more on consideration of different potential approaches, important tradeoffs, and potential pitfalls/gaps/unforeseen complexities. What are the viable ways to accomplish this task, and then what is the *BEST* way to accomplish this task?

Your high level plan should also be succinct. Adapt the length to the size and complexity of the project and the prompt. For simple tasks, a few sentences are sufficient. For complex tasks, a few paragraphs are appropriate. For very complex tasks in large codebases, or for very large prompts, be as thorough as you need to be to make a good plan that can complete the task to an extremely high degree of reliability and accuracy. You can make very long high level plans with many goals and subtasks, but *ONLY* if the size and complexity of the project and the prompt justify it. Your DEFAULT should be *brevity* and *conciseness*. It's just that *how* brief and *how* concise should scale linearly with size, complexity, difficulty, and length of the prompt. If you can make a strong plan in very few words or sentences, do so.

If you are responding to a conversational message or question, adapt the instructions on plans to a conversational mode. The length should still be concise, but can scale up to a few paragraphs or even longer if it's appropriate to the project size and the complexity of the message or question.

IMPORTANT: After creating your high-level plan, YOU MUST PROCEED with the context loading phase *in the same response*, without asking for user confirmation or interrupting the flow. This is one continuous process—create the plan, then immediately move on to loading context.
`

	if params.ExecMode {
		s += `
*Execution mode is enabled.* This means that you are able to run commands on the user's machine. Include consideration of any commands that may need to be run in your high level plan, especially commands for installing required dependencies or building and running the project.

Do not 'force it' when it comes to running commands. Don't guess at commands to run—if you're unsure, it's better to omit commands than to include incorrect ones. Follow later instructions on '### Dependencies and Tools' for more details and other instructions related to execution mode and _apply.sh.
`
	}

	s += `
[CONTEXT INSTRUCTIONS:]

You are operating in 'auto-context mode'. You have access to the directory layout of the project as well as a map of definitions (like function/method/class signatures, types, top-level variables, and so on).

In response to the user's latest prompt, do the following IN ORDER:

  1. Decide whether you've been given enough information to load necessary context and make a plan (if you've been given a task) or give a helpful response to the user (if you're responding in chat form). In general, do your best with whatever information you've been provided. Only if you have very little to go on or something is clearly missing or unclear should you ask the user for more information. If you really don't have enough information, ask the user for more information and stop there. ('Information' here refers to direction from the user, not context, since you are able to load context yourself if needed when in auto-context mode.)

  2. Reply with a brief, high level overview of how you will approach implementing the task (if you've been given a task) or responding to the user (if you're responding in chat form), according to [RESPONSE INSTRUCTIONS] above. Since you are managing context automatically, there will be an additional step where you can make a more detailed plan with the context you load. Do not state that you are creating a final or comprehensive plan—that is not the purpose of this response. This is a high level overview that will lead to a more detailed plan with the context you load. Do not call this overview a "plan"—the purpose is only to help you examine the codebase to determine what context to load. You will then make a plan in the next step.

`
	if params.ExecMode {
		s += `
     - Since execution mode is enabled, include consideration of any commands that may need to be run in your high level plan as described in [RESPONSE INSTRUCTIONS] above. Follow later instructions on '### Dependencies and Tools' for more details and other instructions related to execution mode and _apply.sh.
`
	}

	s += `
  3. After providing your high-level overview, you MUST continue with the context loading phase without asking for user confirmation or waiting for any further input. This is one continuous process in a single response.

  4. If you already have enough information from the project map and current context to make a detailed plan or respond effectively to the user and so you won't need to load any additional context, then explicitly say "No context needs to be loaded." and continue on to the instructions below. NEVER say "No context needs to be loaded." *after* you've already output the '### Context Categories' and '### Files' sections.

  5. Otherwise, you MUST output:
     a) A section titled "### Context Categories" listing one or more categories of context that are relevant to the user's task or message. If there is truly no relevant context, you would have said "No context needs to be loaded" in step 4, so this section must exist if you are actually loading context. Do not list files here—just categories.
     b) A section titled "### Files" enumerating the relevant files and symbols from the codebase map that correspond to the categories you listed. See additional rules below.
     c) Immediately after the '### Files' list, output a <PlandexFinish/> tag. ***Do not output any text after <PlandexFinish/>.***

`

	// Insert shared instructions on how to group and list context
	s += GetAutoContextShared(params, true)

	s += `
[END OF CONTEXT INSTRUCTIONS]
`

	return s
}

func GetAutoContextChatPreamble(params CreatePromptParams) string {
	s := `
[CONTEXT INSTRUCTIONS:]

You are operating in 'auto-context mode' for chat. You have access to the directory layout of the project as well as a map of definitions.

First, assess if you need additional context:
- Are there specific files referenced that you need to examine?
- Would related files help you give a more accurate or complete answer?
- Do you need to understand implementations or dependencies?
- Have you already loaded similar context in a recent response? If so, avoid loading it again.

If NO additional context is needed:
- Continue with your response conversationally

If you need context:
- Briefly mention what you need to check, e.g. "Let me look at the relevant files..." or "Let me look at those functions..." — use your judgment and respond in a natural, conversational manner.
- Then proceed with the context loading format:

` + GetAutoContextShared(params, false) + `

Remember: Only load context when genuinely needed for accuracy. Avoid loading context in consecutive responses as this disrupts conversation flow.

[END OF CONTEXT INSTRUCTIONS]
`

	return s
}

func GetAutoContextShared(params CreatePromptParams, tellMode bool) string {
	s := `
- In a section titled '### Context Categories', list one or more categories of context that are relevant to the user's task, question, or message. For example, if the user is asking you to implement an API endpoint, you might list 'API endpoints', 'database operations', 'frontend code', 'utilities', and so on. Make sure any and all relevant categories are included, but don't include more categories than necessary—if only a single category is relevant, then only list that one. Do not include file paths, symbols, or explanations—only the categories.`

	if tellMode && params.ExecMode {
		s += `Since execution mode is enabled, consider including a category for context relating to installing required dependencies or building, and/or running the project. Adapt this to the user's project, task, and prompt. Don't force it—only include this category if it makes sense. Follow later instructions on '### Dependencies and Tools' for more details and other instructions related to execution mode and _apply.sh.`
	}

	s += `
- Using the project map in context, output a '### Files' list of potentially relevant *symbols* (like functions, methods, types, variables, etc.) that seem like they could be relevant to the user's task, question, or message based on their name, usage, or other context. Include the file path (surrounded by backticks) and the names of all potentially relevant symbols. File paths *absolutely must* be surrounded by backticks like this: ` + "`path/to/file.go`" + `. Any symbols that are referred to in the user's prompt must be included. You MUST organize the list by category using the categories from the '### Context Categories' section—ensure each category is represented in the list. When listing symbols, output just the name of the symbol, not it's full signature (e.g. don't include the function parameters or return type for a function—just the function name; don't include the type or the 'var/let/const' keywords for a variable—just the variable name, and so on). Output the symbols as a comma separated list in a single paragraph for each file. You MUST include relevant symbols (and associated file paths) for each category from the '### Context Categories' section. Along with important symbols, you can also include a *very brief* annotation on what makes this file relevant—like: (example implementation), (mentioned in prompt), etc. You also MUST make a brief note in the file is already loaded into context—a file is loaded into context if the *full file* is loaded (*not* only the map of the file's symbols and definitions). At the end of the list, output a <PlandexFinish/> tag.

- Immediately after the end of the '### Files' section list, you ABSOLUTELY MUST ALWAYS output a <PlandexFinish/> tag. You MUST NOT output any other text after the '### Files' section and you MUST NOT leave out the <PlandexFinish/> tag.

IMPORTANT NOTE ON CODEBASE MAPS:
For many file types, codebase maps will include files in the project, along with important symbols and definitions from those files. For other file types, the file path will be listed with '[NO MAP]' below it. This does NOT mean the the file is empty, does not exist, is not important, or is not relevant. It simply means that we either can't or prefer not to show the map of that file. You can still use the file path to load the file and see its full content if appropriate. For files without a map, instead of making judgments about the file's relevance based on the symbols in the map, judge based on the file path and name.
--

When assessing relevant context, you MUST follow these rules:

1. Interface & Implementation Rule:
   - When loading an implementation file, you MUST also load its interface file
   - When loading a type file, you MUST also load related type definitions
   Example: If loading 'handlers/users.go', you must also load 'types/user.go'

2. Reference Implementation Rule:
   - When implementing a feature similar to an existing one, you MUST load the existing feature's files as reference
   - Look for files with similar patterns, names, or purposes

3. API Client Chain Rule:
   - When working with API clients, you MUST load:
     * The API interface file
     * The client implementation file
   Example: If updating API methods, load any relevant types or interface files as well as the implementation files for the methods you're working with

4. Database Chain Rule:
   - When working with database operations, you MUST load:
     * Related model files
     * Related helper files
     * Similar existing DB operations
   Example: If adding user settings table, load other settings-related DB files

5. Utility Dependencies Rule:
   - Examine the code you're writing for any utility function calls
   - Load ALL files containing utilities you might need
   Example: If using string formatting utilities, load the utils file with those functions

Remember: It's better to load more context than you need than to miss an important file. If you're not sure whether a file will be helpful, include it.

When considering relevant categories in the '### Context Categories' and relevant symbols in the '### Files' sections:

1. Look for naming patterns:
   - Files with similar prefixes or suffixes
   - Files in similar locations
   Example: If working on 'user_config.go', look for other '*_config.go' files

2. Look for feature groupings:
   - Find all files related to similar features
   - Look for files that work together
   Example: If adding settings, find all existing settings-related files

3. Follow file relationships:
   - For each file you identify, check for:
     * Its interface file
     * Its test file
     * Its helper files
     * Related type definitions
   Example: For 'api/methods.go', look for 'types/api.go', 'api/methods_test.go'

When listing files in the '### Files' section, make sure to include:

1. ALL interface files for any implementations
2. ALL type definitions related to the task or prompt
3. ALL similar feature files for reference
4. ALL utility files that might be related to the task or prompt
5. ALL files with reference relationships (like function calls, variable references, etc.)
`

	if tellMode && params.ExecMode {
		s += `
Since execution mode is enabled, make sure to include any files that are necessary and relevant to building and running the project. For example, if there is a Makefile, a package.json file, or equivalent, include it.

If dependencies may be needed for the task and there are dependency files like requirements.txt, package.json, go.mod, Gemfile, or equivalent, include them.

Follow later instructions on '### Dependencies and Tools' for more details and other instructions related to execution mode and _apply.sh.

Don't force it or overdo it. Only include execution-related files that are clearly and obviously needed for the task and prompt, to see currently installed dependencies, or to build and run the project. For example, do NOT include an entire directory of test files. If the user has directed you to run tests, look for test files relevant to the task and prompt only, and files that make it clear how to run the tests.

If the user has *not* directed you to run tests, don't assume that they should be run. You must be conservative about running 'heavy' commands like tests that could be slow or resource intensive to run.

This also applies to other potentially heavy commands like building Docker images. Use your best judgement.
`
	}

	s += `
After outputting the '### Files' section, end your response. Do not output any additional text after that section.

***Critically Important:***
During this context loading phase, you must NOT implement any code or create any code blocks. This phase is ONLY for identifying relevant context.`

	return s
}
