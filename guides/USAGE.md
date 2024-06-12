# Using Plandex  🛠️

## Provider API key(s)  🔑

By default, Plandex uses the OpenAI API. While you can use Plandex with many different providers and models, Plandex requires reliable function calling, which can still be a challenge to find in non-OpenAI models. For getting started, it's recommended to begin with OpenAI.

If you don't have an OpenAI account, first [sign up here.](https://platform.openai.com/signup)

Then [generate an API key here.](https://platform.openai.com/account/api-keys)

You'll also need API keys for any other providers you plan on using, like OpenRouter.ai (Anthropic, Gemini, and open source models), Together.ai (open source models), Replicate, Ollama, and more. 

## Environment variables  💻

```bash
export OPENAI_API_KEY=...

export OPENAI_API_BASE=... # optional - set a different base url for OpenAI calls e.g. https://<your-proxy>/v1
export OPENAI_ORG_ID=... # optional - set the OpenAI OrgID if you have multiple orgs

# optional - set api keys for any other providers you're using
export OPENROUTER_API_KEY=...
export TOGETHER_API_KEY...
```

## New plan  🪄

Now `cd` into your **project's directory.** Make a new directory first with `mkdir your-project-dir` if you're starting on a new project.

```bash
cd your-project-dir
```

Then **start your first plan** with `plandex new`.

```bash
plandex new
```

When you create a plan, Plandex will automatically name your plan after you give it a task, but you can also give it a name up front.

```bash
plandex new -n foo-adapters-component
```

If you don't give your plan a name up front, it will be named 'draft' until you give it a task. To keep things tidy, you can only have one active plan named 'draft'. If you create a new draft plan, any existing draft plan will be removed.

## Loading context  📄

After creating a plan, load any relevant files, images, directories, directory layouts, urls, or other data into the plan context.

```bash
plandex load component.ts action.ts reducer.ts
plandex load lib -r # loads lib and all its subdirectories
plandex load tests/**/*.ts # loads all .ts files in tests and its subdirectories
plandex load . --tree # loads the layout of the current directory and its subdirectories (file names only)
plandex load https://redux.js.org/usage/writing-tests # loads the text-only content of the url
npm test | plandex load # loads the output of `npm test`
plandex load -n 'add logging statements to all the code you generate.' # load a note into context
plandex load ui-mockup.png # load an image into context (GPT-4o only so far)
```

For loading images, png, jpeg, non-animated gif, and webp formats are supported. So far, this feature is only available with the default OpenAI GPT-4o model.

## Tasks  ⚡️

Now give the AI a task to do.

```bash
# with no arguments, vim or nano will open and you can type your task there
plandex tell 
# you can pass a task directly as a string 
# press enter for line breaks while inside the quotes
plandex tell 'build another component like this one that displays foo adapters in the table rather than bar adapters.
quote> use the same layout and styles, but update the column headers and formatting to match the new data.
quote> add the needed reducer and action as well. write tests for the new code.' 
# load a task from a file
plandex tell --file task.txt # or -f task.txt
```

## Changes  🏗️

Plandex will stream the response to your terminal and build up a set of changes along the way. It will continue as long as necessary and create or update as many files as needed to complete the task. You can stop it at any time if it starts going in the wrong direction or if feedback would be helpful.

You can review the changes that Plandex has built up so far in a user-friendly TUI changes viewer.

```bash
plandex changes
```

If you're happy with the changes, apply them to your files.

```bash
plandex apply
```

If you're in a git repo, Plandex will automatically add a commit with a nicely formatted message describing the changes. Any uncommitted changes that were present in your working directory beforehand will be unaffected.

## Rewind  ⏪  

If you want to rewind and try a different approach, you can use `log` to show a list of updates and `rewind` commands to go back in time.

```bash
plandex log # show a list of all updates to the plan, including prompts, replies, and builds
plandex rewind # go back a single step
plandex rewind 3 # go back 3 steps
plandex rewind a7c8d66 # rewind to a specific state
```

## Branches  🌱

If you want to try a different approach but also keep the current one around, you can use branches. Create a new branch before rewinding.

```bash
plandex checkout new-approach # create a new branch and switch to it
plandex rewind 2
plandex tell 'write the tests before the components.'
plandex branches # see all branches
plandex delete-branch new-approach # delete a branch
```

## Continue  ▶️

If a plan has stopped and you just want to continue where you left off, you can use the `continue` command.

```bash
plandex continue # continue the current plan
```

## Background tasks  🚞

If you want to run a command in the background, use the --bg flag.

```bash
plandex tell --bg 'now add another similar component for widget adapters'
```

To see plans that are currently running (or recently finished) and their current status, use the `ps` command. You can connect to a running plan's stream to check on it. Or you can stop it.

```bash
plandex ps # show active and recently finished plans
plandex connect # select an active plan to connect to
plandex stop # select an active plan to stop
```

## Context management  📑

You can see the plan's current context with the `ls` command. You can remove context with the `rm` command or clear it all with the `clear` command.

```bash
plandex ls # list all context in current plan
plandex rm component.ts # remove by name
plandex rm 2 # remove by number in the `plandex ls` list
plandex rm 2-5 # remove a range of indices
plandex rm lib/**/*.js # remove by glob pattern
plandex rm lib # remove whole directory
plandex clear # remove all context
```

If files in context are modified outside of Plandex, you will be prompted to update them the next time you interact with the AI. You can also update them manually with the `update` command.

```bash
plandex update # update files in context
```

## Plans  🌟

When you have multiple plans, you can list them with the `plans` command, switch between them with the `cd` command, see the current plan with the `current` command, and delete plans with the `delete-plan` command. 

You can archive plans you want to keep around but aren't currently working on with the `archive` command. You can see archived plans in the current directory with `plans --archived`. You can unarchive a plan with the `unarchive` command.

```
plandex plans # list all plans

plandex cd # select from a list of plans
plandex cd some-other-plan # cd to a plan by name
plandex cd 2 # cd to a plan by number in the `plandex plans` list

plandex current # show the current plan

plandex delete-plan # select from a list of plans to delete
plandex delete-plan some-plan # delete a plan by name
plandex delete-plan 4 # delete a plan by number in the `plandex plans` list

plandex archive # select from a list of plans to archive
plandex archive some-plan # archive a plan by name
plandex archive 2 # archive a plan by number in the `plandex plans` list

plandex unarchive # select from a list of archived plans to unarchive
plandex unarchive some-plan # unarchive a plan by name
plandex unarchive 2 # unarchive a plan by number in the `plandex plans --archived` list
```

## Conversation history  💬

You can see the full conversation history with the `convo` command.

```bash
plandex convo # show the full conversation history
```

You can output the conversation in plain text with no ANSI codes with the `--plain` or `-p` flag.

```bash
plandex convo --plain
```

You can also show a specific message number or range of messages.

```bash
plandex convo 1 # show the initial prompt
plandex convo 1-5 # show messages 1 through 5
plandex convo 2- # show messages 2 through the end of the conversation
```

## Conversation summaries  🤏

Every time the AI model replies, Plandex will summarize the conversation so far in the background and store the summary in case it's needed later. When the conversation size in tokens exceeds the model's limit, Plandex will automatically replace some number of older messages with the corresponding summary. It will summarize as many messages as necessary to keep the conversation size under the limit.

You can see the latest summary with the `summary` command.

```bash
plandex summary # show the latest conversation summary
```

As with the `convo` command, you can output the summary in plain text with no ANSI codes with the `--plain` or `-p` flag.

```bash
plandex summary --plain
```

## Model settings  🧠

You can see the current AI models and model settings with the `models` command and change them with the `set-model` command.

```bash
plandex models # show the current AI models and model settings
plandex models available # show all available models
plandex set-model # select from a list of models and settings
plandex set-model planner openrouter/anthropic/claude-opus-3 # set the main planner model to Claude Opus 3 from OpenRouter.ai
plandex set-model builder temperature 0.1 # set the builder model's temperature to 0.1
plandex set-model max-tokens 4000 # set the planner model overall token limit to 4000
plandex set-model max-convo-tokens 20000  # set how large the conversation can grow before Plandex starts using summaries
```

Model changes are versioned and can be rewound or applied to a branch just like any other change.

### Model defaults  

`set-model` udpates model settings for the current plan. If you want to change the default model settings for all new plans, use `set-model default`.

```bash
plandex models default # show the default model settings
plandex set-model default # select from a list of models and settings
plandex set-model default planner openai/gpt-4 # set the default planner model to OpenAI gpt-4
```

### Custom models

Use `models add` to add a custom model and use any provider that is compatible with OpenAI, including OpenRouter.ai, Together.ai, Ollama, Replicate, and more.

```bash
plandex models add # add a custom model
plandex models available --custom # show all available custom models
plandex models delete # delete a custom model
```

### Model packs

Instead of changing models for each role one by one, a model pack lets you switch out all roles at once. You can create your own model packs with `model-packs create`, list built-in and custom model packs with `model-packs`, and remove custom model packs with `model-packs delete`.

```bash
plandex set-model # select from a list of model packs for the current plan
plandex set-model default # select from a list of model packs to set as the default for all new plans
plandex set-model gpt-4-turbo-latest # set the current plan's model pack by name
plandex set-model default gpt-4-turbo-latest # set the default model pack for all new plans

plandex model-packs # list built-in and custom model packs
plandex model-packs create # create a new custom model pack
plandex model-packs --custom # list only custom model packs
```

## .plandex directory  ⚙️

When you run `plandex new` for the first time in any directory, Plandex will create a `.plandex` directory there for light project-level config.  

If multiple people are using Plandex with the same project, you should either:

- Put `.plandex/` in `.gitignore` 
- **Commit** the `.plandex` directory and get everyone into the same **org** in Plandex (see next section).

## Orgs  👥

When creating a new org, you have the option of automatically granting access to anyone with an email address on your domain. If you choose not to do this, or you want to invite someone from outside your email domain, you can use `plandex invite`.

To join an org you've been invited to, use `plandex sign-in`.

To list users and pending invites, use `plandex users`.

To revoke an invite or remove a user, use `plandex revoke`.

Orgs will be the basis for plan sharing and collaboration in future releases. 

## Directories  📂

So far, we've assumed you're running `plandex new` to create plans in your project's root directory. While that is the most common use case, it can be useful to create plans in subdirectories of your project too. That's because context file paths in Plandex are specified relative to the directory where the plan was created. So if you're working on a plan for just one part of your project, you might want to create the plan in a subdirectory in order to shorten paths when loading context or referencing files in your prompts. This can also help with plan organization if you have a lot of plans.

When you run `plandex plans`, in addition to showing you plans in the current directory, Plandex will also show you plans in nearby parent directories or subdirectories. This helps you keep track of what plans you're working on and where they are in your project hierarchy. If you want to switch to a plan in a different directory, first `cd` into that directory, then run `plandex cd` to select the plan.

<!-- ```bash
cd your-project
plandex new -n root-project-plan # cwd is 'your-project'
plandex current # 'your-project' current plan is root-project-plan
plandex load file.go # loads 'your-project/file.go'
cd some-subdirectory # cwd is now 'some-subdirectory'
plandex new -n subdir-plan1 # current plan is subdir-plan1
plandex load subfile.go # loads 'some-subdirectory/subfile.go'
plandex new -n subdir-plan2 # current plan is now subdir-plan2
plandex plans # shows subdir-plan1 and subdir-plan2 in current directory + root-project-plan in parent directory
cd ../ # cwd is now 'your-project', current plan is root-project-plan
plandex plans # shows root-project-plan in current directory + subdir-plan1 and subdir-plan2 in child directory 'some-subdirectory'
cd some-subdirectory # cwd is now 'some-subdirectory', current plan is subdir-plan2
plandex cd subdir-plan1 # cwd is still 'some-subdirectory', current plan is now subdir-plan1
``` -->

One more thing to note on directories: you can load context from parent or sibling directories if needed by using `..` in your load paths.

```bash
plandex load ../file.go # loads file.go from parent directory
plandex load ../sibling-dir/test.go # loads test.go from sibling directory
```

## Ignoring files  🙈

Plandex respects `.gitignore` and won't load any files that you're ignoring. You can also add a `.plandexignore` file with ignore patterns to any directory.

## Help  ℹ️

There are a few more commands that haven't been covered in this guide. To see all available commands:

```
plandex help
```

For help on any command:

```
plandex [command] --help
```

## Aliases  🥷

You can use the `pdx` alias anywhere you would use `plandex`. Many commands have short aliases as well.

```bash
pdx l component.ts action.ts reducer.ts # plandex load
pdx t -f task.txt # plandex tell
pdx co # plandex checkout
pdx br # plandex branches
pdx rw # plandex rewind
pdx c # plandex continue
pdx ch # plandex changes
pdx ap # plandex apply
pdx conn # plandex connect
pdx s # plandex stop
pdx pl # plandex plans
pdx cu # plandex current
pdx dp # plandex delete-plan
pdx db # plandex delete-branch
```
