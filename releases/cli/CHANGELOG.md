## CLI Version 2.0.0-rc.4
Placeholder for v2 release testing

## CLI Version 2.0.0-rc.3
Placeholder for v2 release testing

## CLI Version 2.0.0-rc.2
Placeholder for v2 release testing

## CLI Version 2.0.0-rc.1
Placeholder for v2 release testing

## Version 1.1.1
## Fix for terminal flickering when streaming plans 📺

Improvements to stream handling that greatly reduce flickering in the terminal when streaming a plan, especially when many files are being built simultaneously. CPU usage is also reduced on both the client and server side.

## Claude 3.5 Sonnet model pack is now built-in 🧠

You can now easily use Claude 3.5 Sonnet with Plandex through OpenRouter.ai.

1. Create an account at [OpenRouter.ai](https://openrouter.ai) if you don't already have one.
2. [Generate an OpenRouter API key](https://openrouter.ai/keys).
3. Run `export OPENROUTER_API_KEY=...` in your terminal.
4. Run `plandex set-model`, select `choose a model pack to change all roles at once` and then choose either `anthropic-claude-3.5-sonnet` (which uses Claude 3.5 Sonnet for all heavy lifting and Claude 3 Haiku for lighter tasks) or `anthropic-claude-3.5-sonnet-gpt-4o` (which uses Claude 3.5 Sonnet for planning and summarization, gpt-4o for builds, and gpt-3.5-turbo for lighter tasks)

[plandex-claude-3.5-sonnet](https://github.com/plandex-ai/plandex/blob/main/releases/images/cli/1.1.1/clause-3-5-sonnet.gif)

Remember, you can run `plandex model-packs` for details on all built-in model packs.

## Version 1.1.0
## Support for loading images into context with gpt-4o 🖼️

- You can now load images into context with `plandex load path/to/image.png`. Supported image formats are png, jpeg, non-animated gif, and webp. So far, this feature is only available with the default OpenAI GPT-4o model.

![plandex-load-images](https://github.com/plandex-ai/plandex/blob/main/releases/images/cli/1.1.0/plandex-images.gif)

## No more hard OpenAI requirement for builder, verifier, and auto-fix roles 🧠

- Non-OpenAI models can now be used for *all* roles, including the builder, verifier, and auto-fix roles, since streaming function calls are no longer required for these roles.

- Note that reliable function calling is still required for these roles. In testing, it was difficult to find models that worked reliably enough for these roles, despite claimed support for function calling. For this reason, using non-OpenAI models for these roles should be considered experimental. Still, this provides a path forward for using open source, local, and other non-OpenAI models for these roles in the future as they improve.

## Reject pending changes with `plandex reject` 🚫

- You can now reject pending changes to one or more files with the `plandex reject` command. Running it with no arguments will reject all pending changes after confirmation. You can also reject changes to specific files by passing one or more file paths as arguments.

![plandex-reject](https://github.com/plandex-ai/plandex/blob/main/releases/images/cli/1.1.0/plandex-reject.gif)

## Summarization and auto-continue fixes 🛤 ️

- Fixes for summarization and auto-continue issues that could Plandex to lose track of where it is in the plan and repeat tasks or do tasks out of order, especially when using `tell` and `continue` after the initial `tell`.

## Verification and auto-fix improvements 🛠️

- Improvements to the verification and auto-fix step. Plandex is now more likely to catch and fix placeholder references like "// ... existing code ..." as well as incorrect removal or overwriting of code.

## Stale context fixes 🔄

- After a context file is updated, Plandex is less likely to use an old version of the code from earlier in the conversation--it now uses the latest version much more reliably.

## `plandex convo` command improvements 🗣️

- Added a `--plain / -p` flag to `plandex convo` and `plandex summary` that outputs the conversation/summary in plain text with no ANSI codes.
- `plandex convo` now accepts a message number or range of messages to display (e.g. `plandex convo 1`, `plandex convo 1-5`, `plandex convo 2-`). Use `plandex convo 1` to show the initial prompt.

## Context management improvements 📄

- Give notes added to context with `plandex load -n 'some note'` an auto-generated name in the `context ls` list.
- `plandex rm` can now accept a range of indices to remove (e.g. `plandex rm 1-5`)
- Better help text if `plandex load` is run with incorrect arguments
- Fix for `plandex load` issue loading paths that begin with `./`

## Better rate limit tolerance 🕰️

- Increase wait times when receiving rate limit errors from OpenAI API (common with new OpenAI accounts that haven't spent $50)

## Built-in model updates 🧠

- Removed 'gpt-4-turbo-preview' from list of built-in models and model packs

## Other fixes 🐞

- Fixes for some occasional rendering issues when streaming plans and build counts
- Fix for `plandex set-model` model selection showing built-in model options that aren't compatible with the selected role--now only compatible models are shown

## Help updates 📚

- `plandex help` now shows a brief overview on getting started with Plandex rather than the full command list
- `plandex help --all` or `plandex help -a` shows the full command list

## Version 1.0.0
- CLI updates for the 1.0.0 release
- See the [server/v1.0.0 release notes](https://github.com/plandex-ai/plandex/releases/tag/server%2Fv1.0.0) for full details

## Version 0.9.1
- Fix for occasional stream TUI panic during builds with long file paths (https://github.com/plandex-ai/plandex/issues/105)
- If auto-upgrade fails due to a permissions issue, suggest re-running command with `sudo` (https://github.com/plandex-ai/plandex/issues/97 - thanks @kalil0321!)
- Include 'openrouter' in list of model providers when adding a custom model (https://github.com/plandex-ai/plandex/issues/107)
- Make terminal prompts that shouldn't be optional (like the Base URL for a custom model) required across the board (https://github.com/plandex-ai/plandex/issues/108)
- Data that is piped into `plandex load` is now automatically given a name in `context ls` via a call to the `namer` role model (previously it had no name, making multiple pipes hard to disambiguate).
- Still show the '(r)eject file' hotkey in the `plandex changes` TUI when the current file isn't scrollable. 

## Version 0.9.0
## Major file update improvements 📄
- Much better accuracy for updates to existing files.
- Plandex is much less likely to screw up braces, parentheses, and other code structures.
- Plandex is much less likely to mistakenly remove code that it shouldn't.

## Major improvements to long plans with many steps 🛤️
- Plandex's 'working memory' has been upgraded. It is now much better at working through very long plans without skipping tasks, repeating tasks it's already done, or otherwise losing track of what it's doing.

## 'plandex diff' command ⚖️

![plandex-diff](https://github.com/plandex-ai/plandex/blob/03263a83d76785846fd472693aed03d36a68b86c/releases/images/cli/0.9.0/plandex-diff.gif)

- New `plandex diff` command shows pending plan changes in `git diff` format.

## Plans can be archived 🗄️

![plandex-archive](https://github.com/plandex-ai/plandex/blob/03263a83d76785846fd472693aed03d36a68b86c/releases/images/cli/0.9.0/plandex-archive.gif)

- If you aren't using a plan anymore, but you don't want to delete it, you can now archive it.
- Use `plandex archive` (or `plandex arc` for short) to archive a plan.
- Use `plandex plans --archived` (or `plandex plans -a`) to see archived plans in the current directory.
- Use `plandex unarchive` (or `plandex unarc`) to restore an archived plan.

## Custom models!! 🧠
### Use Plandex with models from OpenRouter, Together.ai, and more

![plandex-models](https://github.com/plandex-ai/plandex/blob/03263a83d76785846fd472693aed03d36a68b86c/releases/images/cli/0.9.0/plandex-models.gif)

- Use `plandex models add` to add a custom model and use any provider that is compatible with OpenAI, including OpenRouter.ai, Together.ai, Ollama, Replicate, and more.
- Anthropic Claude models are available via OpenRouter.ai. Google Gemini 1.5 preview is also available on OpenRouter.ai but was flakey in initial testing. Tons of open source models are available on both OpenRouter.ai and Together.ai, among other providers.
- Some built-in models and model packs (see 'Model packs' below) have been included as a quick way to try out a few of the more powerful model options. Just use `plandex set-model` to try these.
- You can use a custom model you've added with `plandex set-model`, or add it to a model pack (see 'Model packs' below) with `plandex model-packs create`. Delete custom models you've added with `plandex models delete`.
- The roles a custom model can be used for depend on its OpenAI compatibility.
- Each model provider has an `ApiKeyEnvVar` associated with it, like `OPENROUTER_API_KEY`, `TOGETHER_API_KEY`, etc. You will need to have the appropriate environment variables set with a valid api key for each provider that you're using.
- Because all of Plandex's prompts have been tested against OpenAI models, support for new models should be considered **experimental**.
- If you find prompting patterns that are effective for certain models, please share them on Discord (https://discord.gg/plandex-ai) or GitHub (https://github.com/plandex-ai/plandex/discussions) and they may be included in future releases.

## Model packs 🎛️
- Instead of changing models for each role one by one, a model packs let you switch out all roles at once.
- Use `plandex model-packs create` qto create your own model packs. 
- Use `plandex model-packs` to list built-in and custom model packs. 
- Use `plandex set-model` to load a model pack.
- Use `plandex model-packs delete` to remove a custom model pack.

## Model defaults ⚙️
- Instead of only changing models on a per-plan basis, you can set model defaults that will apply to all new plans you start.
- Use `plandex models default` to see default model settings and `plandex set-model default` to update them. 

## More commands 💻
- `plandex summary` to see the latest plan summary
- `plandex rename` to rename the current plan

## Quality of life improvements 🧘‍♀️
- Descriptive top-line for `plandex apply` commit messages instead of just "applied pending changes".

![plandex-commit](https://github.com/plandex-ai/plandex/blob/03263a83d76785846fd472693aed03d36a68b86c/releases/images/cli/0.9.0/plandex-commit.png)

- Better message in `plandex log` when a single piece of context is loaded or updated.
- Abbreviate really long file paths in `plandex ls`.
- Changed `OPENAI_ENDPOINT` env var to `OPENAI_API_BASE`, which is more standardized. OPENAI_ENDPOINT is still quietly supported.
- guides/ENV_VARS.md now lists environment variables you can use with Plandex (and a few convenience varaiables have been addded) - thanks @knno! → https://github.com/plandex-ai/plandex/pull/94

## Bug fixes 🐞
- Fix for potential crash in `plandex changes` TUI.
- Fixes for some rare potential deadlocks and conflicts when building a file or stopping a plan stream.

## Version 0.8.3
- Add support for new OpenAI models: `gpt-4-turbo` and `gpt-4-turbo-2024-04-09`
- Make `gpt-4-turbo` model the new default model for the planner, builder, and auto-continue roles -- in testing it seems to be better at reasoning and significantly less lazy than the previous default for these roles, `gpt-4-turbo-preview` -- any plan that has not previously had its model settings modified will now use `gpt-4-turbo` by default (those that have been modified will need to be updated manually) -- remember that you can always use `plandex set-model` to change models for your plans
- Fix for `set-model` command argument parsing (https://github.com/plandex-ai/plandex/issues/75)
- Fix for panic during plan stream when a file name's length exceeds the terminal width (https://github.com/plandex-ai/plandex/issues/84)
- Fix for handling files that are loaded into context and later deleted from the file system (https://github.com/plandex-ai/plandex/issues/47)
- Fix to prevent loading of duplicate files, directory trees, or urls into context (https://github.com/plandex-ai/plandex/issues/57)

## Version 0.8.2
- Fix root level --help/-h to use custom help command rather than cobra's help message (re: https://github.com/plandex-ai/plandex/issues/25)
- Include 'survey' fork (https://github.com/plandex-ai/survey) as a proper module instead of a local reference (https://github.com/plandex-ai/plandex/pull/37)
- Add support for OPENAI_ENDPOINT environment variable for custom OpenAI endpoints (https://github.com/plandex-ai/plandex/pull/46)
- Add support for OPENAI_ORG_ID environment variable for setting the OpenAI organization ID when using an API key with multiple OpenAI organizations.

## Version 0.8.1
- Fix for missing 'host' key when creating an account or signing in to a self-hosted server (https://github.com/plandex-ai/plandex/issues/11)
- `add` alias for `load` command + `unload` alias for `rm` command (https://github.com/plandex-ai/plandex/issues/12)
- Add `invite`, `revoke`, and `users` commands to `plandex help` output
- A bit of cleanup of extraneous logging

## Version 0.8.0
- `plandex invite` command to invite users to an org
- `plandex users` command to list users and pending invites for an org
- `plandex revoke` command to revoke an invite or remove a user from an org
- `plandex sign-in` fixes
- Fix for context update of directory tree when some paths are ignored
- Fix for `plandex branches` command showing no branches immediately after plan creation rather than showing the default 'main' branch

## Version 0.7.3
- Fixes for changes TUI replacement view
- Fixes for changes TUI text encoding issue
- Fixes context loading
- `plandex rm` can now remove a directory from context
- `plandex apply` fixes to avoid possible conflicts
- `plandex apply` ask user whether to commit changes
- Context update fixes
- Command suggestions can be disabled with PLANDEX_DISABLE_SUGGESTIONS environment variable

## Version 0.7.2
- PLANDEX_SKIP_UPGRADE environment variable can be used to disable upgrades
- Color fixes for light backgrounds

## Version 0.7.1
- Fix for re-running command after an upgrade
- Fix for user input prompts
