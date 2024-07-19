pdx-1: var GlobalConfig = map[string]string{
pdx-2:     "path": "/default/path",
pdx-3:     "timeout": "30s",
pdx-4: }
pdx-5: 
pdx-6: func init() {
pdx-7:     // Load additional settings
pdx-8:     GlobalConfig["debug"] = "false"
pdx-9: }
