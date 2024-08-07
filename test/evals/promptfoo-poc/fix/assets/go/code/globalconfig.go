var GlobalConfig = map[string]string{
    "path": "/default/path",
    "timeout": "30s",
}

func init() {
    // Load additional settings
    GlobalConfig["debug"] = "false"
}