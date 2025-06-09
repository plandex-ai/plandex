## Server Version 2.1.8
- Fix for potential hang in file map queue

## Server Version 2.1.7
- Fix for "conversation summary timestamp not found in conversation" error (https://github.com/plandex-ai/plandex/issues/274)
- Fix for potential panic/crash during plan stream (https://github.com/plandex-ai/plandex/issues/275)
- Better protection against panics/crashes in server goroutines across the board

## Server Version 2.1.6+1
See CLI 2.1.6+1 release notes.

## Server Version 2.1.6
See CLI 2.1.6 release notes.

## Server Version 2.1.5
See CLI 2.1.5 release notes.

## Server Version 2.1.4
- Fix to remove occasional extraneous blank lines from start/end of edited files.

## Server Version 2.1.3
- Fix for 'panic in execTellPlan' error when using a model pack that doesn't explicitly set the 'coder' or 'whole-file-builder' roles

## Server Version 2.1.2
- Fix for auto-load context error: 'Error decoding response → EOF'

## Server Version 2.1.1+1
- Improve error handling to catch yet another "context length exceeded" error message variation from Anthropic.

## Server Version 2.1.1
See CLI 2.1.1 release notes.

## Server Version 2.1.0+1
- Fix for context length exceeded error that still wasn't being caught and retried by the fallback correctly.

## Server Version 2.1.0
See CLI 2.1.0 release notes.

## Server Version 2.0.6
- Improvements to process management and cleanup for command execution
- Remove extraneous model request logging

## Server Version 2.0.5
- Fix for a bug that was causing occasional model errors. Model calls should be much more reliable now.
- Better error handling and error messages for model errors (rate limits or other errors).
- No error retries for rate limit errors.
- Fixed bug that caused retries to add the prompt to the conversation multiple times.
- Error responses with no output no longer create a log entry.

## Server Version 2.0.4
- **Stability**
  - Enhanced database locking mechanisms.
  - Improved error notifications.

- **API Enhancements**
  - Added endpoints for managing custom models and updating model packs.

- **Execution**
  - Increased robustness in plan execution and subprocess lifecycle management.

- **Observability**
  - Real-time internal notifications for critical errors implemented.

- **Consistency**
  - Improved token management.
  - Enhanced summarization accuracy.

## Server Version 2.0.3
- Fix for potential crash during chat/tell operation.
- Panic handling to prevent crashes in general.
- Fix for local queue handling bug during builds that could cause queue to get stuck and cause subsequent operations to hang.

## Server Version 2.0.2
Server-side fix for context auto-load hanging when there's no valid context to load (for example, if they're all directories, which is only discovered client-side, and which can't be auto-loaded)

## Server Version 2.0.0+2
- Version tag sanitation fix for GitHub Action to build and push server image to DockerHub

## Server Version 2.0.0+1
- Fix for custom model creation (https://github.com/plandex-ai/plandex/issues/214)
- Fix for version check on self-hosted (https://github.com/plandex-ai/plandex/issues/213)
- Fix for GitHub Action to build and push server image to DockerHub

## Server Version 2.0.0
See CLI 2.0.0 release notes.

## Version 1.1.1
- Improvements to stream handling that greatly reduce flickering in the terminal when streaming a plan, especially when many files are being built simultaneously. CPU usage is also reduced on both the client and server side.
- Claude 3.5 Sonnet model and model pack (via OpenRouter.ai) is now built-in.

## Version 1.1.1
- Improvements to stream handling that greatly reduce flickering in the terminal when streaming a plan, especially when many files are being built simultaneously. CPU usage is also reduced on both the client and server side.
- Claude 3.5 Sonnet model and model pack (via OpenRouter.ai) is now built-in.

## Version 1.1.1
- Improvements to stream handling that greatly reduce flickering in the terminal when streaming a plan, especially when many files are being built simultaneously. CPU usage is also reduced on both the client and server side.
- Claude 3.5 Sonnet model and model pack (via OpenRouter.ai) is now built-in.

## Version 1.1.0
- Give notes added to context with `plandex load -n 'some note'` automatically generated names in `context ls` list.
- Fixes for summarization and auto-continue issues that could Plandex to lose track of where it is in the plan and repeat tasks or do tasks out of order, especially when using `tell` and `continue` after the initial `tell`.
- Improvements to the verification and auto-fix step. Plandex is now more likely to catch and fix placeholder references like "// ... existing code ..." as well as incorrect removal or overwriting of code.
- After a context file is updated, Plandex is less likely to use an old version of the code from earlier in the conversation--it now uses the latest version much more reliably.
- Increase wait times when receiving rate limit errors from OpenAI API (common with new OpenAI accounts that haven't spent $50).

## Version 1.0.1
- Fix for occasional 'Error getting verify state for file' error
- Fix for occasional 'Fatal: unable to write new_index file' error
- Fix for occasional 'nothing to commit, working tree clean' error
- When hitting OpenAI rate limits, Plandex will now parse error messages that include a recommended wait time and automatically wait that long before retrying, up to 30 seconds (https://github.com/plandex-ai/plandex/issues/123)
- Some prompt updates to encourage creation of multiple smaller files rather than one mega-file when generating files for a new feature or project. Multiple smaller files are faster to generate, use less tokens, and have a lower error rate compared to a continually updated large file.

## Version 1.0.0
##   ☄️  🌅   gpt-4o is the real deal for coding

- gpt-4o, OpenAI's latest model, is the new default model for Plandex. 4o is much better than gpt-4-turbo (the previous default model) in early testing for coding tasks and agent workflows.
- If you have not used `plandex set-model` or `plandex set-model default` previously to set a custom model, you will now be use gpt-4o by default. If you *have* used one of those commands, use `plandex set-model` or `plandex set-model default` and select the new `gpt-4o-latest` model-pack to upgrade. 
 
##   🛰️  🏥   Reliability improvements: 90% reduction in syntax errors in early testing

- Automatic syntax and logic validation with an auto-correction step for file updates.
- Significantly improves reliability and reduces syntax errors, mistaken duplication or removal of code, placeholders that reference other code and other similar issues. 
- With a set of ~30 internal evals spanning 5 common languages, syntax errors were reduced by over 90% on average with gpt-4o. 
- Logical errors are also reduced (I'm still working on evals for those to get more precise numbers).
- Plandex is now much better at handling large files and plans that make many updates to the same file. Both could be problematic in previous versions.
- Plandex is much more resilient to incorrectly labelled file blocks when the model uses the file label format incorrectly to explain something rather than for a file. i.e. "Run this script" and then a bash script block. Previously Plandex would mistakenly create a file called "Run this script". It now ignores blocks like these.

##   🧠  🚞   Improvements to core planning engine: better memory and less laziness allow you to accomplish larger and more complex tasks without errors or stopping early

- Plandex is now much better at working through long plans without skipping tasks, repeating tasks it's already done, or otherwise losing track of what it's doing.
- Plandex is much less likely to leave TODO placeholders in comments instead of fully completing a task, or to otherwise leave a task incomplete.
- Plandex is much less likely to end a plan before all tasks are completed.

##   🏎️  📈   Performance improvements: 2x faster planning and execution

- gpt-4o is twice as fast as gpt-4-turbo for planning, summarization, builds, and more.
- If you find it's streaming too fast and you aren't able to review the output, try using the `--stop / -s` flag with `plandex tell` or `plandex continue`. It will stop the plan after a single response so you can review it before proceeding. Use `plandex continue` to proceed with the plan once you're ready.
- Speaking of which, if you're in exploratory mode and want to use less tokens, you can also use the `--no-build / -n` flag with `plandex tell` and `plandex continue`. This prevents Plandex from building files until you run `plandex build` manually.

##   💰  🪙   2x cost reduction: gpt-4o is half the per-token price of gpt-4-turbo

- For the same quantity of tokens, with improved quality and 2x speed, you'll pay half-price.

##   👩‍💻  🎭   New `plandex-dev` and `pdxd` alias in development mode

- In order to avoid conflicts/overwrites with the `plandex` CLI and `pdx` alias, a new `plandex-dev` command and `pdxd` alias have been added in development mode. 

##  🐛  🛠️   Bug fixes

- Fix for a potential panic during account creation (https://github.com/plandex-ai/plandex/issues/76)
- Fixes for some account creation flow issues (https://github.com/plandex-ai/plandex/issues/106)
- Fix for occasional "Stream buffer tokens too high" error (https://github.com/plandex-ai/plandex/issues/34).
- Fix for potential panic when updating model settings. Might possibly be the cause of or somehow related to https://github.com/plandex-ai/plandex/issues/121 but hard to be sure (maybe AWS was just being flakey).
- Attempted fix for rare git repo race condition @jesseswell_1 caught that gives error ending with: 
```
Exit status 128, output
      * Fatal: unable to write new_index file
```

##   📚  🤔   Readme updates

- The [readme](https://github.com/plandex-ai/plandex) has been revamped to be more informative and easier to navigate.

##  🏡  📦   Easy self-contained startup script for local mode and self-hosting

```bash
git clone https://github.com/plandex-ai/plandex.git
cd plandex/app
./start_local.sh
``` 

- Sincere thanks to @ZanzyTHEbar aka @daofficialwizard on Discord who wrote the script! 🙏🙏

##   🚀  ☝️   Upgrading   

- As always, cloud has already been updated with the latest version. To upgrade the CLI, run any `plandex` command (like `plandex version` or `plandex help` or whatever command you were about to run anyway 🙂)

##   💬  📆   Join me for office hours every Friday 12:30-1:30pm PST in Discord, starting May 17th

- I'll be available by voice and text chat to answer questions, talk about the new version, and hear about your use cases. Come on over and hang out! 
- Join the discord to get a reminder when office hours are starting: https://discord.gg/plandex-ai

## Version 0.9.1
- Improvements to auto-continue check. Plandex now does a better job determining whether a plan is finished or should automatically continue by incorporating the either the latest plan summary or the previous conversation message (if the summary isn't ready yet) into the auto-continue check. Previously the check was using only the latest conversation message.
- Fix for 'exit status 128' errors in a couple of edge case scenarios.
- Data that is piped into `plandex load` is now automatically given a name in `context ls` via a call to the `namer` role model (previously it had no name, making multiple pipes hard to disambiguate).

## Version 0.9.0
- Support for custom models, model packs, and default models (see CLI 0.9.0 release notes for details).
- Better accuracy for updates to existing files.
- Plandex is less likely to screw up braces, parentheses, and other code structures.
- Plandex is less likely to mistakenly remove code that it shouldn't.
- Plandex is now much better at working through very long plans without skipping tasks, repeating tasks it's already done, or otherwise losing track of what it's doing.
- Server-side support for `plandex diff` command to show pending plan changes in `git diff` format.
- Server-side support for archiving and unarchiving plans.
- Server-side support for `plandex summary` command.
- Server-side support for `plandex rename` command.
- Descriptive top-line for `plandex apply` commit messages instead of just "applied pending changes".
- Better message in `plandex log` when a single piece of context is loaded or updated.
- Fixes for some rare potential deadlocks and conflicts when building a file or stopping astream.

## Version 0.8.4
- Add support for new OpenAI models: `gpt-4-turbo` and `gpt-4-turbo-2024-04-09`
- Make `gpt-4-turbo` model the new default model for the planner, builder, and auto-continue roles -- in testing it seems to be better at reasoning and significantly less lazy than the previous default for these roles, `gpt-4-turbo-preview` -- any plan that has not previously had its model settings modified will now use `gpt-4-turbo` by default (those that have been modified will need to be updated manually) -- remember that you can always use `plandex set-model` to change models for your plans
- Fix for handling files that are loaded into context and later deleted from the file system (https://github.com/plandex-ai/plandex/issues/47)
- Handle file paths with ### prefixes (https://github.com/plandex-ai/plandex/issues/77)
- Fix for occasional race condition during file builds that causes error "Fatal: Unable to write new index file"

## Version 0.8.3
- SMTP_FROM environment variable for setting from address when self-hosting and using SMTP (https://github.com/plandex-ai/plandex/pull/39)
- Add support for OPENAI_ENDPOINT environment variable for custom OpenAI endpoints (https://github.com/plandex-ai/plandex/pull/46)
- Add support for OPENAI_ORG_ID environment variable for setting the OpenAI organization ID when using an API key with multiple OpenAI organizations.
- Fix for unhelpful "Error getting plan, context, convo, or summaries" error message when OpenAI returns an error for invalid API key or insufficient credits (https://github.com/plandex-ai/plandex/issues/32)

## Version 0.8.2
- Fix for creating an org that auto-adds users based on email domain (https://github.com/plandex-ai/plandex/issues/24)
- Fix for possible crash after error in file build
- Added crash prevention measures across the board
- Fix for occasional "replacements failed" error
- Reliability and improvements for file updates
- Fix for role name of auto-continue model

## Version 0.8.1
- Fixes for two potential server crashes
- Fix for server git repo remaining in locked state after a crash, which caused various issues
- Fix for server git user and email not being set in some environments (https://github.com/plandex-ai/plandex/issues/8)
- Fix for 'replacements failed' error that was popping up in some circumstances
- Fix for build issue that could cause large updates to fail, take too long, or use too many tokens in some circumstances
- Clean up extraneous logging
- Prompt update to prevent ouputting files at absolute paths (like '/etc/config.txt')
- Prompt update to prevent sometimes using file block format for explanations, causing explanations to be outputted as files
- Prompt update to prevent stopping before the plan is really finished 
- Increase maximum number of auto-continuations to 50 (from 30)

## Version 0.8.0
- User management improvements and fixes
- Backend support for `plandex invite`, `plandex users`, and `plandex revoke` commands
- Improvements to copy for email verification emails
- Fix for org creation when creating a new account
- Send an email to invited user when they are invited to an org
- Add timeout when forwarding requests from one instance to another within a cluster

## Version 0.7.1
- Fix for SMTP email issue
- Add '/version' endpoint to server

## Version 0.7.0
Initial release
