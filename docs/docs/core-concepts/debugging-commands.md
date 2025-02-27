---
sidebar_position: 9
sidebar_label: Debug Terminal Commands
---

# Debug Terminal Commands

As of version 2.0.0, Plandex includes a powerful new `plandex debug` command that can repeatedly run any terminal command, continually making fixes based on the command's output until it succeeds.

## Using `plandex debug`

To use `plandex debug`, simply run it with the command you want to debug:

```bash
plandex debug 'npm test'
```

This is a shortcut for:

1. Running a shell command and checking whether it succeeds or fails.
2. If it fails, send the exit code and command output to `plandex tell`.
3. Run `plandex apply` to apply the suggested fixes.
4. Repeat until the command succeeds.

## Number of Tries

By default, `plandex debug` will run the command up to 5 times before giving up. You can change this by providing a different number of tries as the first argument:

```bash
plandex debug 10 'npm test'
```

## Commands That Succeed

If a command succeeds on the first try, `plandex debug` will exit immediately without making any model calls, so you can use it for commands that may or may not succeed on the first try.

```bash
plandex debug "echo 'ok'" # succeeds and immediately exits
```

## Be Careful!!

`plandex debug` is a powerful tool, but it should be used with care. Because it applies changes automatically and repeatedly without a review step, it can quickly make a large number of changes to your project. Before using it, it's a good idea to make sure you have a clean git state so that you can easily revert the changes if something goes wrong.

You should also be careful when using `plandex debug` with commands that may have side effects. Always test commands manually first to make sure they work as expected.

If possible, try to make the commands you give to `plandex debug` *idempotent*, meaning that they're safe to run multiple times. For example, if you have a deploy script, you'd want to be sure that it cleans up after itself if it fails halfway through so that you don't end up with a partially deployed system.

## Alternative: Piping Into `plandex tell`

For a less automated approach that can accomplish the same thing, you can run your command and then pipe its output into `plandex tell`:

```bash
npm test | plandex tell 'npm test output'
```

This will work similarly to `plandex debug`, but without the automatic retries and changes. You can review the changes and then run `plandex apply` if you're happy with them.
