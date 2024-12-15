I want to add a plan config feature. It should be loosely modeled  
on the plan model settings (often referred to also as just plan settings),  
and the 'model packs' feature. I want to store a 'plan_config' JSON field on
the 'plans' table. I also want a 'settings' and a 'set' CLI command.        
  
'settings' shows the current config (like 'models' cmd)  and 'set' allows 
updating the config just like 'set-model'. the server-side updates can also
work similarly to models and model sets. it shouldn't be added to the file  
system or use git at all-it should just be stored in the database.  These are *new* commands separate from the 'models' and 'set-model' commands and should have their own files.

For prompting and user input in the 'set' command, use the same approach as the 'set-model' command. Don't introduce any new dependencies.                                                                              
it should have these properties to start:                                   
                                                                              
AutoApply bool AutoCommit bool AutoContext bool NoExec bool AutoDebug bool AutoDebugTries int         

Apart from the plan config, I also want a default user-level config with the
same properties. similar to how there's 'set-model' and 'set-model default'-
this should also have a 'set default' command. again use model settings as a
guide.

On the server side, to keep things neater, create new files for the API and DB handlers rather than including them in the existing plan settings handlers.

On the client side, update the API interface and implementation for the new api calls.

Also update the CLI 'tell', 'continue', 'build', and 'chat' commands to use config settings by default (they can be overridden with the flags that already exist).

Server-side, it needs api handlers, db handlers, and db up/down migration.

Also add request/response types.

For the 'new' command, show the default settings to the user after the plan is created (in a nicely formatted way, similar to the 'settings' command).

Update the CLI help output accordingly. Also add the appropriate command suggestions to the 'settings' and 'set' commands. Also add suggestions to the 'new' command to demonstrate the new config settings commands.