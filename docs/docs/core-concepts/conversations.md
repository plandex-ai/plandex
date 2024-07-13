---
sidebar_position: 7
sidebar_label: Conversations
---

# Conversations

Each time you send a prompt to Plandex or Plandex responds, the plan's **conversation** is updated. Conversations are [version controlled](./version-control.md) and can be [branched](./branches.md).

## Conversation History

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

## Conversation Summaries

Every time the AI model replies, Plandex will summarize the conversation so far in the background and store the summary in case it's needed later. When the conversation size in tokens exceeds the model's limit, Plandex will automatically replace some number of older messages with the corresponding summary. It will summarize as many messages as necessary to keep the conversation size under the limit.

You can see the latest summary with the `summary` command.

```bash
plandex summary # show the latest conversation summary
```

As with the `convo` command, you can output the summary in plain text with no ANSI codes with the `--plain` or `-p` flag.

```bash
plandex summary --plain
```

