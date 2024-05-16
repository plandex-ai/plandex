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
