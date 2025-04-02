---
sidebar_position: 4
sidebar_label: REPL
---

# REPL

The Plandex REPL is a developer-friendly chat interface. It's the easiest way to use Plandex.

## Start

To start the REPL, just run `plandex` or `pdx` in any project directory (or sub-directory).

If you don't have a current plan in the directory, the REPL will create a new one. Otherwise, it will use the current plan.

## Commands

All Plandex CLI commands are available in the REPL. Just type a backslash (`\`) followed by the command.

The REPL also has a few special commands of its own:

- `\quit` or `\q` to quit the REPL
- `\help` or `\h` for help
- `@` plus a relative file path for loading files into context (note that if you're using auto-context mode, loading files yourself is optional)
- `\run` or `\r` plus a relative file path for using a file as a prompt
- `\chat` or `\ch` to switch to chat mode and have a conversation without making changes
- `\tell` or `\t` to switch to tell mode and implement tasks
- `\multi` or `\m` to switch to multi-line mode
- `\send` or `\s` to send the current prompt to Plandex (for sending a prompt in multi-line mode, since enter gives you a newline)

## REPL Flags

The REPL has a few convenient flags you can use to start it with different modes, autonomy settings, and model packs. You can pass any of these to `plandex` or `pdx` when starting the REPL.

```
  Mode
    --chat, -c     Start in chat mode (for conversation without making changes)
    --tell, -t     Start in tell mode (for implementation)

  Autonomy
    --no-auto      None → step-by-step, no automation
    --basic        Basic → auto-continue plans, no other automation
    --plus         Plus → auto-update context, smart context, auto-commit changes
    --semi         Semi-Auto → auto-load context
    --full         Full-Auto → auto-apply, auto-exec, auto-debug

  Models
    --daily        Daily driver pack (default models, balanced capability, cost, and speed)
    --strong       Strong pack (more capable models, higher cost and slower)
    --cheap        Cheap pack (less capable models, lower cost and faster)
    --oss          Open source pack (open source models)
    --gemini-exp   Gemini experimental pack (Gemini 2.5 Pro Experimental for planning and coding, default models for other roles)
```
