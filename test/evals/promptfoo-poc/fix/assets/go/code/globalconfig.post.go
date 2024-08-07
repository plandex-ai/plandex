var GlobalConfig = map[string]string{
    // Configuration settings
    "path": "/new/default/path",
    "timeout": "30s",
}

func init() {
    // Load additional settings
    GlobalConfig["debug"] = "false"
    GlobalConfig["path"] = "/new/default/path" // duplicated update
}