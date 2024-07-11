---
title: Intro
description: Plandex is an open source, terminal-based AI coding engine that helps you work on complex, real-world development tasks with LLMs.
sidebar_position: 1
sidebar_label: Intro
---

# Introduction

Plandex is an open source, terminal-based AI coding engine that helps you work on complex, real-world development tasks with LLMs.

It combines multiple agents to complete tasks that span many files and model responses. When you give Plandex a task, it continues working automatically until the task is determined to be complete.   

## Use cases

- Build new projects from scratch.
- Add features to existing projects.
- Write tests.
- Write scripts.
- Fix bugs.
- Refactor.
- Work with unfamiliar technologies.
- Ask questions about code.
- Understand a codebase. 

## What makes Plandex different?

### Version Control

Plandex gives the LLM its own version-controlled staging area/sandbox (separate from your project's git repo) where all of its proposed changes are accumulated. This allows you to:

- Iterate on your code and the LLM's plan at the same time without the changes becoming intertwined and difficult to disentangle.
- Review proposed changes across multiple files as a whole (rejecting any that aren't correct) to be sure that broken updates or hallucinations don't make it into your project files.
- Branch or rewind the LLM's plan in order to explore multiple paths or revert to the exact step where a task went off the rails.

### Context Management

Apart from version control, Plandex also helps you manage what's in the LLM's context:

- Add files or directories to context from the terminal instead of copying and pasting or clicking around in a UI. 
- Files you add to context are kept up-to-date automatically so that the LLM is always using the latest version.
- Unlike IDE-based tools that automatically and opaquely load context in the background, Plandex gives the developer precise control of what's in the LLM's context. You never have to wonder what's been loaded or whether it's up-to-date. This is crucial to getting good results and keeping a handle on costs, particularly when you want to go beyond auto-complete and work on larger tasks.

## Focus

For now, Plandex is focused on: 

1. Planning out the changes needed to complete a task.
2. Creating or updating all the necessary files to complete that task. 

It doesn't yet do automatic execution of code or automatic selection of context—both are left to the developer.

In other words, Plandex isn't (yet) shooting for full autonomy. While we do plan to move in this direction over time, we think the current level of model capabilities make Plandex's focus a sweet spot for achieving real productivity gains.

Though full autonomy is certainly an enticing prospect, in practice it often means wasting a lot of time and tokens letting the LLM spin its wheels on problems that are trivial for a developer to identify and fix. You can definitely use Plandex to help debug its own code, but for now we think it's best to make this opt-in rather than the default behavior.

## Models

Plandex uses OpenAI models by default, as so far we've found them to offer the best balance between intelligence and reliability.

That said, you can also use it with a wide range of other models, including Anthropic Claude, Google Gemini, Mixtral, Llama and many more via OpenRouter.ai, Together.ai, or any other OpenAI-compatible provider.

## Languages and Platforms

You can use Plandex to work with any language or framework that the underlying LLM has been trained on. For the largest models, this includes just about any language or framework you can think of, though output quality will tend to be best for those that are more popular and therefore better represented in the training data.

Plandex is cross-platform and easy to install. It supports Mac, Linux, FreeBSD, and Windows. It runs from a single binary with no dependencies.

## Hosting

Plandex runs on a client-server architecture. The Plandex server is open source and can be self-hosted. A cloud-hosted option is also offered for getting started as quickly as possible with minimal setup.

## Community

Join our growing community and help guide Plandex's development.

- [GitHub](https://github.com/plandex-ai/plandex) - post an [issue](https://github.com/plandex-ai/plandex/issues), start a [discussion](https://github.com/plandex-ai/plandex/discussions), or [fork and contribute.](https://github.com/plandex-ai/plandex/fork)
- [Discord](https://discord.gg/plandex-ai) - ask for help, give feedback, share your use cases, or just hang out.
- [X](https://x.com/PlandexAI) - follow for updates on new versions and other AI coding content.
- [YouTube](https://www.youtube.com/@plandex-ny5ry) - subscribe to watch various tasks and projects get completed with Plandex.
