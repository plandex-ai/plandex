package prompts

const AutoContextPreamble = `
[CONTEXT INSTRUCTIONS:]

You are operating in 'auto-context mode'. You have access to the directory layout of the project as well as a map of definitions (like function/method/class signatures, types, top-level variables, and so on).
    
In response to the user's latest prompt, do the following:

  - Decide whether you've been given enough information to load necessary context and make a plan (if you've been given a task) or give a helpful response to the user (if you're responding in chat form). In general, do your best with whatever information you've been provided. Only if you have very little to go on or something is clearly missing or unclear should you ask the user for more information. If you really don't have enough information, ask the user for more information and stop there. 'Information' here refers to direction from the user, not context, since you are able to load context yourself if needed when in auto-context mode.

  - Reply with a brief, high level overview of how you will approach implementing the task (if you've been given a task) or responding to the user (if you're responding in chat form). Since you are managing context automatically, there will be an additional step where you can make a more detailed plan with the context you load. Do not state that you are creating a final or comprehensive plan—that is not the purpose of this response. This is a high level overview that will lead to a more detailed plan with the context you load. Do not call this overview a 'plan'—the purpose is only to help you exmaine the codebase to determine what context to load. You will then make a plan in the next step.

  - If you already have enough information from the directory layout, map, and current context to make a detailed plan or respond effectively to the user and so you won't need to load any additional context, then explicitly say "No context needs to be loaded." and continue on to the instructions below.

  - In a section titled '### Context Categories', list one or more categories of context that are relevant to the user's task, question, or message. For example, if the user is asking you to implement an API endpoint, you might list 'API endpoints', 'database operations', 'frontend code', 'utilities', and so on. Make sure any and all relevant categories are included, but don't include more categories than necessary—if only a single category is relevant, then only list that one. Do not include file paths, symbols, or explanations—only the categories.

  - Using the map and file tree in context, output a '### Relevant Symbols' list of potentially relevant *symbols* (like functions, methods, types, variables, etc.) that seem like they could be relevant to the user's task, question, or message based on their name, usage, or other context. Include the file path (surrounded by backticks) and the names of all potentially relevant symbols. File paths *absolutely must* be surrounded by backticks like this: ` + "`" + "path/to/file.go" + "`" + `. Any symbols that are referred to in the user's prompt must be included. You MUST organize the list by category using the categories from the '### Context Categories' section—ensure each category is represented in the list.

  ` + ContextLoadingRules + `

` + FileMapScanningRules + `

` + ContextCompletionCriteria + `

After outputting the '### Relevant Symbols' section, end your response. Do not output any additional text after that section.

[END OF CONTEXT INSTRUCTIONS]
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

const FileMapScanningRules = `When considering relevant categories in the '### Context Categories' and relevant symbols in the '### Relevant Symbols' sections:

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

const ContextCompletionCriteria = `When considering relevant categories in the '### Context Categories' and relevant symbols in the '### Relevant Symbols' sections, make sure to include:

1. ALL interface files for any implementations
2. ALL type definitions related to the task or prompt
3. ALL similar feature files for reference
4. ALL utility files that might be related to the task or prompt
5. ALL files with reference relationships (like function calls, variable references, etc.)
`
