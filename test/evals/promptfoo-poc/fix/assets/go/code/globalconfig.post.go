pdx-1: var GlobalConfig = map[string]string{
pdx-2:     // Configuration settings
pdx-3:     "path": "/new/default/path",
pdx-4:     "timeout": "30s",
pdx-5: }
pdx-6: 
pdx-7: func init() {
pdx-8:     // GlobalConfig holds configuration settings.
pdx-9:     GlobalConfig["debug"] = "false"
pdx-10:     GlobalConfig["path"] = "/new/default/path" // duplicated update
pdx-11: }
