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
