package types

import ignore "github.com/sabhiram/go-gitignore"

type ProjectPaths struct {
	ActivePaths    map[string]bool
	AllPaths       map[string]bool
	ActiveDirs     map[string]bool
	AllDirs        map[string]bool
	PlandexIgnored *ignore.GitIgnore
	IgnoredPaths   map[string]string
}
