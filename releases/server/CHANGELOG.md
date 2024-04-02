## Version 0.8.1
- Fixes for two potential server crashes
- Fix for server git repo remaining in locked state after a crash, which caused various issues
- Fix for server git user and email not being set in some environments (https://github.com/plandex-ai/plandex/issues/8)
- Fix for 'replacements failed' error that was popping up in some circumstances
- Fix for build issue that could cause large updates to fail, take too long, or use too many tokens in some circumstances
- Clean up extraneous logging
- Prompt update to prevent ouputting files at absolute paths (like '/etc/config.txt')
- Prompt update to prevent sometimes using file block format for explanations, causing explanations to be outputted as files
- Prompt update to prevent stopping before the the plan is really finished 
- Increase maximum number of auto-continuations to 50 (from 30)

## Version 0.8.0
- User management improvements and fixes
- Backend support for `plandex invite`, `plandex users`, and `plandex revoke` commands
- Improvements to copy for email verification emails
- Fix for org creation when creating a new account
- Send an email to invited user when they are invited to an org
- Add timeout when forwarding requests from one instance to another within a cluster

## Version 0.7.1
- Fix for SMTP email issue
- Add '/version' endpoint to server

## Version 0.7.0
Initial release
