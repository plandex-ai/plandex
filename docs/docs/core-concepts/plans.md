---
sidebar_position: 1
sidebar_label: Plans
---

# Plans

A **plan** in Plandex is similar to a conversation in ChatGPT. It might only include a single prompt and model response that executes one small task, or it could represent a long back and forth with the model that generates dozens of files and builds a whole feature or an entire app.

A plan includes: 

- Any context that you've loaded. 
- Your conversation with the model. 
- Any proposed changes that have been accumulated during the course of the conversation.

Plans support [version control](./version-control.md) and [branches](./branches.md).

## Creating a New Plan

First `cd` into your **project's directory.** Make a new directory first with `mkdir your-project-dir` if you're starting on a new project.

```bash
cd your-project-dir
```

Then **start your first plan** with `plandex new`.

```bash
plandex new
```

## Plan Names and Drafts

When you create a plan, Plandex will automatically name your plan after you send the first prompt, but you can also give it a name up front.

```bash
plandex new -n foo-adapters-component
```

If you don't give your plan a name up front, it will be named `draft` until you send an initial prompt. To keep things tidy, you can only have one active plan named `draft`. If you create a new draft plan, any existing draft plan will be removed.

## Listing Plans

When you have multiple plans, you can list them with the `plans` command.

```bash
plandex plans
```

## The Current Plan

It's important to know what the **current plan** is for any given directory, since most Plandex commands are executed against that plan.

To check the current plan:

```bash
plandex current
```

You can change the current plan with the `cd` command:

```
plandex cd # select from a list of plans
plandex cd some-other-plan # cd to a plan by name
plandex cd 2 # cd to a plan by number in the `plandex plans` list
```

## Deleting Plans

You can delete a plan with the `delete-plan` command:

```bash
plandex delete-plan # select from a list of plans to delete
plandex delete-plan some-plan # delete a plan by name
plandex delete-plan 4 # delete a plan by number in the `plandex plans` list
```

## Archiving Plans

You can archive plans you want to keep around but aren't currently working on with the `archive` command. You can see archived plans in the current directory with `plans --archived`. You can unarchive a plan with the `unarchive` command.

```bash
plandex archive # select from a list of plans to archive
plandex archive some-plan # archive a plan by name
plandex archive 2 # archive a plan by number in the `plandex plans` list

plandex unarchive # select from a list of archived plans to unarchive
plandex unarchive some-plan # unarchive a plan by name
plandex unarchive 2 # unarchive a plan by number in the `plandex plans --archived` list
```

## .plandex Directory

When you run `plandex new` for the first time in any directory, Plandex will create a `.plandex` directory there for light project-level config.  

If multiple people are using Plandex with the same project, you should either:

- **Commit** the `.plandex` directory and get everyone into the same [org](./orgs.md) in Plandex.
- Put `.plandex/` in `.gitignore`

## Project Directories

So far, we've assumed you're running `plandex new` to create plans in your project's root directory. While that is the most common use case, it can be useful to create plans in subdirectories of your project too. That's because context file paths in Plandex are specified relative to the directory where the plan was created. So if you're working on a plan for just one part of your project, you might want to create the plan in a subdirectory in order to shorten paths when loading context or referencing files in your prompts. This can also help with plan organization if you have a lot of plans.

When you run `plandex plans`, in addition to showing you plans in the current directory, Plandex will also show you plans in nearby parent directories or subdirectories. This helps you keep track of what plans you're working on and where they are in your project hierarchy. If you want to switch to a plan in a different directory, first `cd` into that directory, then run `plandex cd` to select the plan.