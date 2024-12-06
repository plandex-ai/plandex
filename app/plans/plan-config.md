I want to add a plan config feature. It should be loosely modeled  
on the plan model settings (often referred to also as just plan settings),  
and the 'model packs' feature. I want to store a 'plan_config' JSON field on
the 'plans' table. I also want a 'settings' and a 'set' CLI command.        
  
'settings' shows the current settings (like 'models' cmd)  and 'set' allows 
updating the settings just like 'set-model'. the server-side updates can also
work similarly to models and model sets. it shouldn't be added to the file  
system or use git at all-it should just be stored in the database.          
                                                                              
it should have these properties to start:                                   
                                                                              
AutoApply bool AutoCommit bool AutoContext bool NoExec bool AutoDebug bool AutoDebugTries int         

Apart from the plan config, I also want a default user-level config with the
same properties. similar to how there's 'set-model' and 'set-model default'-
this should also have a 'set default' command. again use model settings as a
guide.

On the server side, to keep things neater, create new files for the API and DB handlers rather than including them in the existing plan settings handlers.
