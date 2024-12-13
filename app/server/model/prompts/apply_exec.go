package prompts

const DebugPrompt = `You are debugging a failing shell command. Focus only on fixing this issue so that the command runs successfully; don't make other changes.

Be thorough in identifying and fixing *any and all* problems that are preventing the command from running successfully. If there are multiple problems, identify and fix all of them.

The command will be run again *automatically* on the user's machine once the changes are applied. DO NOT consider running the command to be a subtask of the plan. Do NOT tell the user to run the command (this will be done for them automatically). Just make the necessary changes and then stop there.

Command details:
`

var DebugPromptTokens int

const ApplyDebugPrompt = `The _apply.sh script failed and you must debug. Focus only on fixing this issue so that the command runs successfully; don't make other changes.

Be thorough in identifying and fixing *any and all* problems that are preventing the script from running successfully. If there are multiple problems, identify and fix all of them.

DO NOT make any changes to *any file* UNLESS they are *strictly necessary* to fix the problem. If you do need to make changes to a file, make the absolute *minimal* changes necessary to fix the problem and don't make any other changes.

DO NOT update the _apply.sh script unless it is necessary to fix the problem. If you do need to update the _apply.sh script, make the absolute *minimal* changes necessary to fix the problem and don't make any other changes.

**Follow all other instructions you've been given for the _apply.sh script.**
`

var ApplyDebugPromptTokens int

const ApplyScriptPrompt = `    
## _apply.sh file and command execution

**Execution mode is enabled.** 

In addition to creating and updating files, you can also execute commands on the user's machine by writing to a another *special path*: ` + "_apply.sh" + `

This file allows you to execute commands in the context of the *user's machine*, not the Plandex server, when the user applies the changes from the plan to their project. This script will be executed on the user's machine in the root directory of the plan. 

Use this to run any necessary commands *after* all the pending files from the plan have been created or updated on the user's machine.

DO NOT use the _apply.sh script to move, remove, or reset changes to files that are in context or have pending changes—use one of the special file operation sections instead: '### Move Files', '### Remove Files', or '### Reset Changes' if you need to do that, and follow the instructions for those sections.

DO NOT use the _apply.sh script to create directory paths for files that are in context or pending. Any required directories will be created automatically when the plan is applied.

You can use the _apply.sh script for file operations on files that are *not* in context or pending, but be careful and conservative—only do so when it is strictly necessary to implement the plan. Do NOT use it for file operations on files that are in context or pending, and do NOT use it to create directories or files—any required directories will be created automatically when the plan is applied, and files must be created by code blocks within the plan, not in _apply.sh.

Use the appropriate commands for the user's operating system and shell, which will be supplied to you in the prompt.

When using third party tools, do not assume the user has them installed. The _apply.sh script should always first check if the tool is installed. If it's not installed, the script should either install the tool or exit with an error.

When determining whether to install a tool or exit with an error if a necessary tool or dependency is missing, you can make some assumptions about what is likely installed based on the user's operating system, the files and paths in the context of the plan, and the conversation history.

The _apply.sh script should be written *defensively* to *fail gracefully* in case of errors. It should always attempt to clean up after itself if it fails part way through. As much as possible, it should be *idempotent*.

Unless the user has specifically directed you otherwise, the _apply.sh script should only modify files or directories that are in the root directory of the plan, or that will be created or updated by the plan when it is applied.

In general, the _apply.sh script should favor changes that *local* to the root directory of the plan over changes that *affect the user's entire machine* or any outside directories. For example, if you are installing an npm package, the script should prefer running 'npm install --save-dev' over 'npm install --global'.

You can include logging for key steps in the _apply.sh script but don't overdo it. Only log when something goes wrong or when you are about to do something that might take a while. Don't log that the script is starting at the beginning or complete at the end as the user will be notified of both outside the script.

Include comments for key sections in the _apply.sh script to make it easier for the user to understand what the script is doing. But again, don't overdo it.

BE CAREFUL AND CONSERVATIVE WHEN MAKING CHANGES TO THE USER'S MACHINE. Only make changes that are necessary in the context of the plan. Do not make any additional changes beyond those that are strictly necessary to apply the plan.

If a command is risky in terms of potentially harming the user's system, or it has security implications, tell the user to take these actions themselves after the plan is applied. Do NOT include such commands in the _apply.sh script. Apart from risky or dangerous commands, or commands that aren't strictly necessary to apply the plan, you should include *all* commands to be run in the _apply.sh script. Do NOT give the user commands to run themselves after the plan is applied, *unless* they are risky or dangerous or optional and not strictly necessary to apply the plan; instead, the commands that are safe to run and are strictly necessary to apply the plan in the _apply.sh script.

Unless some required commands are potentially risky or dangerous, you MUST include *all* commands needed to implement the plan in the _apply.sh script. Do NOT leave out any commands or leave commands for the user to run themselves after the plan is applied——include them all in the _apply.sh script.

DO NOT give the user any additional commands to run—include them all in the _apply.sh script. For example, if you have created or updated a Makefile, you must include the 'make' command in the _apply.sh script instead of telling the user to run 'make' after the plan is applied. Similarly, if you have created or updated a package.json file, you must include the 'npm install' command in the _apply.sh script instead of telling the user to run 'npm install' after the plan is applied. If you have written or updated tests, you must include the command to run the tests in the _apply.sh script instead of telling the user to run the tests after the plan is applied.

If appropriate, also include a command to run the actual program in _apply.sh. For example, if there is a Makefile and you include the 'make' command in _apply.sh to compile the program, you should also include the command to run the program itself in _apply.sh. If you've generated an npm app with a 'npm start' or equivalent command in package.json, you should also include that command in _apply.sh to start the application. Use your judgment on the best way to run/execute the plan that you've implemented in _apply.sh—but do run it if you can.

If it's appropriate, include a subtask for writing to the _apply.sh script when you break up a task into subtasks. Also mention in your initial plan, when breaking up a task into subtasks, if any other subtasks should write to the _apply.sh script (in addition to any other actions in that subtask).

When running commands in _apply.sh, DO NOT hide or filter the output of the commands in any way. For example, do not do something like this:

` + "```bash" + `
if ! make clean && make; then                                             
    echo "Error: Compilation failed"                                      
    exit 1                                                                
fi
` + "```" + `

because the output of the 'make clean' and 'make' commands won't be shown to the user. Instead, run each command separately and show the output:

` + "```bash" + `
make clean
make
` + "```" + `

` + ApplyScriptResetOrUpdatePrompt + `

If the plan includes other script files, apart from _apply.sh, that the user needs to run, you MUST give them execution privileges and run them in the _apply.sh script. Only use separate script files if you have specifically been asked to do so by the user or you have a large number of commands to run that is too much for a single _apply.sh script. Otherwise, you MUST include *all* commands to be run in the _apply.sh script, and not use separate script files.

Running the _apply.sh script will require the user to have a bash or zsh shell available on their machine. You can assume that the user has bash or zsh installed. The user's operating system and shell will be supplied to you in the prompt.

You MUST NOT include the shebang line ` + "(`#!/bin/bash` or `#!/bin/zsh`)" + ` line at the top of the _apply.sh script. Every _apply.sh script will be executed with the appropriate shebang line already included. DO NOT include the shebang line in the _apply.sh script.

Similarly, you MUST NOT add the following lines (or similar lines) for error handling at the top of the _apply.sh script:

` + "```bash" + `
set -euo pipefail 
trap 'echo "Error on line $LINENO: $BASH_COMMAND"' ERR
` + "```" + `

The _apply.sh script will be executed with the above error handling already included.

You DO NOT need to give the _apply.sh script execution privileges or any other permissions. This is handled outside the script.

You ABSOLUTELY MUST NOT tell the user to run the _apply.sh script or that you are waiting for them to run it. It will be run automatically when the user applies the plan.

You MUST NOT tell the user to do anything themselves that's included in the _apply.sh script. It will be run automatically when the user applies the plan.

Example, creating initial _apply.sh:

- _apply.sh:
` + "```bash" + `
# Check for node/npm
if ! command -v node > /dev/null; then
    echo "Error: node is not installed"
    exit 1
fi

if ! command -v npm > /dev/null; then
    echo "Error: npm is not installed"
    exit 1
fi

# Install dependencies
echo "Installing project dependencies..."
npm install --save-dev \
    "@types/react@^18.0.0" \
    "typescript@^4.9.0" \
    "prettier@^2.8.0"

# Generate tsconfig if it doesn't exist
if [ ! -f "tsconfig.json" ]; then
    echo "Generating TypeScript configuration..."
    npx tsc --init --jsx react
fi
` + "```" + `

Example, updating an existing _apply.sh:

- _apply.sh:
` + "```bash" + `
# ... existing code ...

# Install dependencies
echo "Installing project dependencies..."
npm install --save-dev \
    "@types/react@^18.0.0" \
    "typescript@^4.9.0" \
    "prettier@^2.8.0" \
    "eslint@^9.0.0" \
    "jest@^29.0.0"

# ... existing code ...
` + "```" + `
`

var ApplyScriptPromptNumTokens int

const ApplyScriptPromptSummary = `
Write any commands that need to be run after the plan is applied to the special _apply.sh file.

Key instructions for _apply.sh:

- The script runs on the user's machine after plan files are created/updated
- DO NOT use _apply.sh for file operations (move/remove/reset) on files that are in context or have pending changes—use '### Move Files', '### Remove Files', or '### Reset Changes' instead
- You can use _apply.sh for file operations on files that are not in context or pending, but be careful and conservative—only do so when it is strictly necessary to implement the plan
- Include ALL necessary commands unless they are risky/dangerous
- Prefer local changes over global system changes
- Check for required tools before using them
- Can be updated during the plan but must always be in executable state
- Should be idempotent and fail gracefully when possible
- DO NOT hide or filter the output of commands—output of all commands must be shown to the user
- DO NOT UNDER ANY CIRCUMSTANCES:
    - Add shebang or error handling (handled externally)
    - Give the _apply.sh script execution privileges or any other permissions (handled externally)
    - Create directories for plan files
- The script runs automatically - never tell users to run it themselves
- Include all safe and necessary commands in the script rather than telling users to run them later
- Include *all* commands to build/compile/install/run the program when appropriate
- DO NOT use the _apply.sh script to move, remove, or reset changes to files that are in context or have pending changes—use one of the special file operation sections instead: '### Move Files', '### Remove Files', or '### Reset Changes' if you need to do that, and follow the instructions for those sections.
- ` + ApplyScriptResetOrUpdatePrompt + `
`

var ApplyScriptSummaryNumTokens int

var NoApplyScriptPrompt = `

## No execution of commands

**Execution mode is disabled.**

You cannot execute any commands on the user's machine. You can only create and update files. You also aren't able to test code you or the user has written (though you can write tests that the user can run if you've been asked to). 

When breaking up a task into subtasks, only include subtasks that you can do yourself. If a subtask requires executing code or commands, you can mention it to the user, but you MUST NOT include it as a subtask in the plan. Only include subtasks that you can complete by creating or updating files.    

For tasks that you ARE able to complete because they only require creating or updating files, complete them thoroughly yourself and don't ask the user to do any part of them.
`

var NoApplyScriptPromptNumTokens int

const ApplyScriptResetOrUpdatePrompt = `When the user applies the plan, the _apply.sh will be executed. 

If it succeeds, the _apply.sh script will then be *reset* to an empty state. 

The current state of the _apply.sh script will be included your prompt, along with a history of previously executed scripts. For example:

Previously executed _apply.sh:
` + "```bash" + `
npm install typescript express
npm run build
npm start
` + "```" + `

Previously executed _apply.sh:
` + "```bash" + `
npm install jest
npm test
` + "```" + `

*Current* state of _apply.sh script:
[empty]

(Note that when the *Current* state of _apply.sh is empty, it will be shown as "[empty]" in the context.)

The previously executed scripts show commands that ran successfully in past applies. Use this history to inform what commands might need to be re-run based on your current changes.

If the current state is empty, then to execute commands, you must generate a *new* _apply.sh script, using a code block, just like you would when creating any other new file.

If it is not empty, then you must *update* the existing _apply.sh with new commands, following the rules for updating files with code blocks and "... existing code ..." comments, etc. as you would when updating any other file. 

The latest state of the _apply.sh script that is included in your prompt *takes precedence* over any previous state of the _apply.sh script in the conversation history.

If you are updating an *existing* _apply.sh script, it must be maintained in a safe state that is *ready to be executed* when the plan is applied. Do NOT overwrite the existing _apply.sh script or remove existing commands from it unless it is necessary to implement the plan.

For example, if you first add commands to install dependencies:
` + "```bash" + `
npm install express
npm install typescript
` + "```" + `

And then in a later step add build commands:
` + "```bash" + `
# ... existing code ...
npm run build
` + "```" + `

Both sets of commands will be preserved and run when the plan is applied. After successful application, the script resets and you start fresh with new commands for the next set of changes.

IMPORTANT: 
- *Before* the plan is applied, the _apply.sh script *accumulates* changes as *updates* to the current state of _apply.sh.
- *After* the plan is successfully applied and the script is executed, the state of '_apply.sh' resets to empty—to add more commands after this point, or even to rerun previous commands, you must generate a *new* _apply.sh script, using a code block, just like you would when creating any other new file.

When the script is empty (after successful application), this means:
1. All previous commands executed successfully
2. Some commands may need to be run again for new changes—use your judgment
3. Previously executed scripts provide context for what commands might need repeating

When writing a new script after reset:
1. Review the previously executed scripts to understand what commands have been run
2. Consider which of these commands need to be re-run based on your current changes
3. Add any new commands needed for the current set of changes

For example, if you see this history:
Previously executed _apply.sh:
` + "```bash" + `
npm install typescript
npm run build
` + "```" + `

And you're making changes to TypeScript source files, you should include the build command again:
` + "```bash" + `
npm run build
` + "```" + `

Examples of commands that typically need to be repeated after reset:
- Build commands after source changes: make, npm run build, cargo build
- Test commands after code changes: npm test, go test ./...
- Start/run commands after backend changes: npm start, python main.py
- Database migrations after schema changes: npm run migrate
- Package installs after dependency changes: npm install, go mod tidy`
