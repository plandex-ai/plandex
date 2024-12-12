package prompts

const AutoContextPreamble = `
[CONTEXT INSTRUCTIONS:]

You are operating in 'auto-context mode'. You have access to the directory layout of the project as well as a map of definitions (like function/method/class signatures, types, top-level variables, and so on).
    
In response to the user's latest prompt, do the following:

  - Decide whether you've been given enough information to load necessary context and make a plan (if you've been given a task) or give a helpful response to the user (if you're responding in chat form). In general, do your best with whatever information you've been provided. Only if you have very little to go on or something is clearly missing or unclear should you ask the user for more information. If you really don't have enough information, ask the user for more information and stop there. 'Information' here refers to direction from the user, not context, since you are able to load context yourself if needed when in auto-context mode.

  - Reply with a high level overview of how you will approach implementing the task (if you've been given a task) or responding to the user (if you're responding in chat form). Since you are managing context automatically, there will be an additional step where you can make a more detailed plan with the context you load. Still, try to consider *everything* the task will require and all the areas of the project it will need to touch. Be thorough and exhaustive in your plan, and don't leave out any steps. For example, if you're being asked to implement an API handler, don't forget that you will need to add the necessary routes to the router as well. Think carefully through details like these and strive not to leave out anything. Do not state that you are creating a final or comprehensive plan—that is not the purpose of this response. This is a high level overview that will lead to a more detailed plan with the context you load.
  
  - State something to the effect of: "I'll examine the codebase to determine which files I need."
  
  - Using the directory layout and the map, explain how the project is organized, with particular focus on areas that may be relevant to the user's task, question, or message.

  - Using the map and file tree in context, output a '### Relevant Symbols' list of potentially relevant *symbols* (like functions, methods, types, variables, etc.) that seem like they could be relevant to the user's task, question, or message based on their name, usage, or other context. Include the file path and the name of all potentially relevant symbols. 

  ` + ContextLoadingRules + `

` + FileMapScanningRules + `

` + ContextCompletionCriteria + `

- At the end of your response, list *all* of the files that are relevant and helpful to the user's task, question, or message in this EXACT format. You ABSOLUTELY MUST ensure that *all* files listed in the plan steps are included in the 'Load Context' list:
  
  ` + LoadContextFormatPrompt + `
  
  - If instead you already have enough information from the directory layout, map, and current context to make a detailed plan or respond effectively to the user and so you won't need to load any additional context, then explicitly say "No context needs to be loaded." and continue on to the instructions below.

  - Every response MUST end with either the 'Load Context' list in the exact format described above or the exact phrase "No context needs to be loaded." and nothing else. This MUST be the final text in your response.

Don't output multiple lists of the files to load in your response. There MUST only be one 'Load Context' list in your response and no other list of the files to load.

[END OF CONTEXT INSTRUCTIONS]
`

const LoadContextFormatPrompt = `
### Load Context
- ` + "`src/main.rs`" + `
- ` + "`lib/term.go`" + `

You MUST use the exact same format as shown directly above. First the '### Load Context' header, then a blank line, then the list of files, with a '-' and a space before each file path. List files individually—do not list directories. List file paths exactly as they are in the directory layout and map, and surround them with single backticks like this: ` + "- `src/main.rs`." + ` Each file path in the list MUST be on its own line. Use this EXACT format.

After the 'Load Context' list, you MUST ALWAYS immediately *stop there* so these files can be loaded before you continue.

The 'Load Context' list MUST *ONLY* include files that are present in the directory layout or map and are not *already* in context. You MUST NOT include any files that are already in context or that do not exist in the directory layout or map. **Do NOT include files that need to be created**—only files that already exist.

Again, *DO NOT* load files that are already in context. *DO NOT* load files that do not exist *right now* in the directory layout or map.

If any code similar to what the user is asking for exists in the project, include at least one example so that you can use it as a reference for how to implement the user's request in a way that is consistent with the existing code. If a user asks you to implement an API endpoint, for example, and you see that there is an existing endpoint in the project that is similar, include it in the context so that you can use it as a reference. Similarly, if a task requires implementing a database migration, and you see that there is an existing migration, include it in the context so that you can use it as a reference. If you are asked to implement a frontend or UI feature, and you see that there are existing UI components, pages, styles, or other related code, include some of it so that you can use it as a reference and keep a consistent style, both in terms of the code and the user experience.

If you'll be using any definitions from a file—calling a function or method, instantiating a class, accessing a variable, using a type, and so on—include that file in the context. Include the file even if it's not being modified. For example, if you are creating an object that uses a type defined in a 'types.ts' file, include the 'types.ts' file in the context even if you're not updating the 'types.ts' file. If you're calling a function from a 'utils.py' file, include the 'utils.py' file in the context even if you're not updating the 'utils.py' file. This will ensure you correctly call functions/methods, use types, use variables and constants, etc. so it is *critical* that you include all files necessary to make a good plan that is well-integrated with the existing code.

Include any files that the user has mentioned directly or indirectly in the prompt. If a user has mentioned files by name, path, by describing them, by referring to definitions or other code they contain, or by referring to them in any other way, include those files.

If you aren't sure whether a file will be helpful or not, but you think it might be, include it in the 'Load Context' list. It's better to load more context than you need than to miss an important or helpful file.

If context is included your prompt, you ABSOLUTELY MUST consider the context that is *ALREADY INCLUDED* and not load additional unnecessary context. It's EXTREMELY IMPORTANT that you do *NOT* load unnecessary context that is already included in the prompt. It's also EXTREMELY IMPORTANT you only list files that will be *UPDATED* during the plan. YOU ABSOLUTELY MUST NEVER include files that will be *CREATED* during the plan.

If you have previously loaded context with a 'Load Context' list in an earlier response during this conversation, focus only on loading any *ADDITIONAL* context that are now helpful or relevant based on the user's latest prompt.

If you have previously loaded context with a 'Load Context' list in an earlier response during this conversation, you DO NOT need to output an '### Additional Context' section. Load any additional context that is now helpful or relevant based on the user's latest prompt and then STOP there *immediately*.

Once you've loaded all necessary context, move on to the instructions below.
`

var AutoContextPreambleNumTokens int

const ContextLoadingRules = `When assessing relevant context, you MUST follow these rules:

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

Remember: It's better to load more context than you need than to miss an important file. If you're not sure whether a file will be helpful, include it.`

const FileMapScanningRules = `When examining the codebase, you MUST:

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
   Example: For 'api/methods.go', look for 'types/api.go', 'api/methods_test.go'`

const ContextCompletionCriteria = `Before finalizing your context loading, verify you have:

1. ALL interface files for any implementations
2. ALL type definitions related to the task or prompt
3. ALL similar feature files for reference
4. ALL utility files that might be related to the task or prompt
5. ALL files with reference relationships (like function calls, variable references, etc.)

You ABSOLUTELY MUST reason through these points in a section titled "### Additional Context". EVERY response MUST include this section *before* the "Load Context" list.

When reasoning through these points in "### Additional Context", *list specific file paths* that fit the criteria and have not yet been included in the response so far. DO NOT output a line like "Need API interface file" or "Need settings files". You MUST also include the specific file paths that fit the criteria. For example, "Need API interface file:` + "`types/api.go`" + ` or "Need settings files:` + "`settings/config.go`" + `,` + "`settings/user.go`" + `  ".

If necessary files for one of these points has already been listed in the response, do not list them again. This section is for adding any additional files that haven't been mentioned previously. If all necessary files have already been listed, and so no additional context is necessary, say so and move on. 

You MUST explicitly state if you're missing any of these categories and load additional context if needed.`
