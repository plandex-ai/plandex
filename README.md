<h1 align="center">
 <a href="https://plandex.ai">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="images/plandex-logo-dark.png"/>
    <source media="(prefers-color-scheme: light)" srcset="images/plandex-logo-light.png"/>
    <img width="400" src="images/plandex-logo-dark-bg.png"/>
 </a>
 <br />
</h1>
<br />

<div align="center">

<p align="center">
  <!-- Call to Action Links -->
  <a href="#install">
    <b>30-Second Install</b>
  </a>
   · 
  <!-- <a href="https://plandex.ai">
    <b>Website</b>
  </a>
   ·  -->
  <a href="https://docs.plandex.ai/">
    <b>Docs</b>
  </a>
   · 
  <a href="#more-examples-">
    <b>Examples</b>
  </a>
   · 
  <a href="https://docs.plandex.ai/hosting/self-hosting">
    <b>Self-Hosting</b>
  </a>
   <!-- · 
  <a href="./guides/DEVELOPMENT.md">
    <b>Development</b>
  </a> -->
  <!--  · 
  <a href="https://discord.gg/plandex-ai">
    <b>Discord</b>
  </a>  
   · 
  <a href="#weekly-office-hours-">
    <b>Office Hours</b>
  </a>  
  -->
</p>

<br>

[![Discord Follow](https://dcbadge.vercel.app/api/server/plandex-ai?style=flat)](https://discord.gg/plandex-ai)
[![GitHub Repo stars](https://img.shields.io/github/stars/plandex-ai/plandex?style=social)](https://github.com/plandex-ai/plandex)
[![Twitter Follow](https://img.shields.io/twitter/follow/PlandexAI?style=social)](https://twitter.com/PlandexAI)

</div>

<p align="center">
  <!-- Badges -->
<a href="https://github.com/plandex-ai/plandex/pulls"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome" /></a> <a href="https://github.com/plandex-ai/plandex/releases?q=cli"><img src="https://img.shields.io/github/v/release/plandex-ai/plandex?filter=cli*" alt="Release" /></a>
<a href="https://github.com/plandex-ai/plandex/releases?q=server"><img src="https://img.shields.io/github/v/release/plandex-ai/plandex?filter=server*" alt="Release" /></a>

  <!-- <a href="https://github.com/your_username/your_project/issues">
    <img src="https://img.shields.io/github/issues-closed/your_username/your_project.svg" alt="Issues Closed" />
  </a> -->

</p>

<br />

<div align="center">
<a href="https://trendshift.io/repositories/8994" target="_blank"><img src="https://trendshift.io/api/badge/repositories/8994" alt="plandex-ai%2Fplandex | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>
</div>

<br>

<h3 align="center">AI driven development in your terminal.<br/>Build entire features and apps with a robust workflow.</h3>

<br/>
<br/>

<!-- Vimeo link is nicer on mobile than embedded video... downside is it navigates to vimeo in same tab (no way to add target=_blank) -->
<!-- https://github.com/plandex-ai/plandex/assets/545350/c2ee3bcd-1512-493f-bdd5-e3a4ca534a36 -->

<a href="https://player.vimeo.com/video/926634577">
  <img src="images/plandex-intro-vimeo.png" alt="Plandex intro video" width="100%"/>
</a>

<br/>
<br/>

## More examples  🎥

<h4>👉  <a href="https://www.youtube.com/watch?v=0ULjQx25S_Y">Building Pong in C/OpenGL with GPT-4o and Plandex</a></h4>

<h4>👉  <a href="https://www.youtube.com/watch?v=rnlepfh7TN4">Fixing a tricky real-world bug in 5 minutes with Claude Opus 3 and Plandex</a></h4>

<br/>

## Learn more about Plandex  🧐

- [Overview](#overview-)
- [Install](#install)
- [Get started](#get-started-)
- [Docs](https://docs.plandex.ai/)
- [Build complex software](#build-complex-software-with-llms-)
- [Why Plandex?](#why-plandex-)
- [Roadmap](#roadmap-%EF%B8%8F)
- [Discussion and discord](#discussion-and-discord-)
- [Contributors](#contributors-)
<br/>

## Overview  📚

<p>Churn through your backlog, work with unfamiliar technologies, get unstuck, and <strong>spend less time on the boring stuff.</strong></p>

<p>Plandex is a <strong>reliable and developer-friendly</strong> AI coding agent in your terminal. It can plan out and complete <strong>large tasks</strong> that span many files and steps.</p>
 
<p>Designed for <strong>real-world use-cases</strong>, Plandex can help you build a new app quickly, add new features to an existing codebase, write tests and scripts, understand code, and fix bugs. </p>

<br/>

## Install  📥

```bash
curl -sL https://plandex.ai/install.sh | bash
```

**Note:** Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/install). Plandex only works correctly on Windows in the WSL shell. It doesn't work in the Windows CMD prompt or PowerShell.

[More installation options.](https://docs.plandex.ai/install)

<br/>

## Get started  🚀

Plandex uses OpenAI by default. If you don't have an OpenAI account, first [sign up here.](https://platform.openai.com/signup)

Then [generate an API key here](https://platform.openai.com/account/api-keys) and `export` it.

```bash
export OPENAI_API_KEY=...
```


Now `cd` into your **project's directory.** Make a new directory first with `mkdir your-project-dir` if you're starting on a new project.

```bash
cd your-project-dir
```


Then **start your first plan** with `plandex new`.

```bash
plandex new
```


Load any relevant files, directories, directory layouts, urls, or images **into the LLM's context** with `plandex load`.

```bash
plandex load some-file.ts another-file.ts
plandex load src/components -r # load a whole directory
plandex load src --tree # load a directory layout (file names only)
plandex load src/**/*.ts # load files matching a glob pattern
plandex load https://raw.githubusercontent.com/plandex-ai/plandex/main/README.md # load the text content of a url
plandex load images/mockup.png # load an image
```


Now **send your prompt.** You can pass it in as a file:

```bash
plandex tell -f prompt.txt
```


Write it in vim:

```bash
plandex tell # tell with no arguments opens vim so you can write your prompt there
```


Or pass it inline (use enter for line breaks):

```bash
plandex tell "add a new line chart showing the number of foobars over time to components/charts.tsx"
```

Plandex will make a plan for your task and then implement that plan in code. **The changes won't yet be applied to your project files.** Instead, they'll accumulate in Plandex's sandbox.

To learn about reviewing changes, iterating on the plan, and applying changes to your project, **[continue with the full quickstart.](https://docs.plandex.ai/quick-start#review-the-changes)**

<br/>

## Docs  🛠️

### [👉  Full documentation.](https://docs.plandex.ai/)


<br/>

## Build complex software with LLMs  🌟

⚡️  Changes are accumulated in a protected sandbox so that you can review them before automatically applying them to your project files. Built-in version control allows you to easily go backwards and try a different approach. Branches allow you to try multiple approaches and compare the results.

📑  Manage context efficiently in the terminal. Easily add files or entire directories to context, and keep them updated automatically as you work so that models always have the latest state of your project.

🧠  By default, Plandex relies on the OpenAI API and requires an `OPENAI_API_KEY` environment variable. You can also use it with a wide range of other models, including Anthropic Claude, Google Gemini, Mixtral, Llama and many more via OpenRouter.ai, Together.ai, or any other OpenAI-compatible provider.

✅  Plandex supports Mac, Linux, FreeBSD, and Windows. It runs from a single binary with no dependencies.

<br/>

## Why Plandex?  🤔

🏗️  Go beyond autocomplete to build complex functionality with AI.<br>
🚫  Stop the mouse-centered, copy-pasting madness of coding with ChatGPT.<br>
⚡️  Ensure the model always has the latest versions of files in context.<br>
🪙  Retain granular control over what's in the model's context and how many tokens you're using.<br>
⏪  Rewind, iterate, and retry as needed until you get your prompt just right.<br>
🌱  Explore multiple approaches with branches.<br>
🔀  Run tasks in the background or work on multiple tasks in parallel.<br>
🎛️  Try different models and temperatures, then compare results.<br>

<br/>

<<<<<<< HEAD
## Plandex Cloud  ☁️

Plandex Cloud is the easiest and most reliable way to use Plandex. You'll be prompted to start an anonymous trial (no email required) when you create your first plan with `plandex new`. Trial accounts are limited to 10 plans and 10 AI model replies per plan. You can upgrade to an unlimited account with your name and email.

Plandex Cloud accounts are free for now. In the future, they will cost somewhere in the $10-20 per month range.

<br/>

## Self-hosting  🏠

Self-contained script for easy local mode and self-hosting:

```bash
git clone https://github.com/plandex-ai/plandex.git
cd plandex/app
./start_local.sh
```

Requires git, docker, and docker-compose.

[Read more about self-hosting Plandex here.](./guides/HOSTING.md)

<br/>

## Limitations and guidance ⚠️

#### **Note  →** while the caveats below still apply to some extent, Plandex's [1.0.0 release](https://github.com/plandex-ai/plandex/releases/tag/server%2Fv1.0.0) that provides gpt-4o support and automatic error-correction is a major step forward in reliability and accuracy, with over 90% reduction in syntax errors and significantly stronger planning capabilities.

- Plandex can provide a significant boost to your productivity, but as with any other AI tool, you shouldn't expect perfect results. Always review a plan before applying changes, especially if security is involved. Plandex is designed to get you 90-95% of the way there rather than 100%.

- Due to the reasoning limitations of LLMs, automatically applied file updates also aren't perfect. While these were significantly improved in the 1.0.0 release, mistakes and errors are still possible. Use the `plandex changes` command to review pending updates in a TUI, or `plandex diffs` to review them in git diff format. If a file update has mistakes, make those changes yourself with copy-and-paste and reject the file in the changes TUI.

- The more direction and detail you provide, the better the results will be. Working with Plandex often involves giving it a prompt, seeing that the results are a bit off, then using `plandex rewind` to go back and iterate on the prompt or add context before trying again. Branches are also useful for experimentation.

- If you want to go step-by-step rather than having Plandex attempt do everything at once, use `plandex tell` and `plandex continue` with the `--stop / -s` flag, which will prevent it from automatically continuing for multiple responses. Use `plandex continue` to proceed with the plan once you're ready.

- While it can be tempting to just dump your entire project into context if it fits under the token limit, and that can work just fine, you will tend to see better results (and pay less) by being more selective about what's loaded into context.

<br/>

## Security  🔐

Plandex Cloud follows best practices for network and data security. And whether cloud or self-hosted, Plandex protects model provider API keys (like your OpenAI API key). [Read more here.](./guides/SECURITY.md)

<br/>

## Privacy and data retention  🛡️

[Read about Plandex Cloud's privacy and data retention policies here.](./guides/PRIVACY.md)

<br/>

=======
>>>>>>> 1e668b3 (finish up docs, update readme for new docs, remove guides dir in favor of docs)
## Roadmap  🗺️

🧠  Support for open source models, Google Gemini, and Anthropic Claude in addition to OpenAI  ✅ released<br>
🖼️  Support for multi-modal models—add images and screenshots to context ✅ released<br>
🤝  Plan sharing and team collaboration<br>
🖥️  VSCode and JetBrains extensions<br>
📦  Community plugins and modules<br>
🔌  Github integration<br>
🌐  Web dashboard and GUI<br>
🔐  SOC2 compliance<br>
🛩️  Fine-tuned models<br>

This list will grow and be prioritized based on your feedback.

<br/>

## Discussion and discord  💬

Speaking of feedback, feel free to give yours, ask questions, report a bug, or just hang out:

- [Discord](https://discord.gg/plandex-ai)
- [Discussions](https://github.com/plandex-ai/plandex/discussions)
- [Issues](https://github.com/plandex-ai/plandex/issues)

<br/>

## Contributors  👥

⭐️  Please star, fork, explore, and contribute to Plandex. There's a lot of work to do and so much that can be improved.

Work on tests, evals, prompts, and bug fixes is especially appreciated.

[Here's an overview on setting up a development environment.](https://docs.plandex.ai/development)


