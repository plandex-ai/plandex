package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"plandex-cli/term"
	"sort"
	"strings"
)

func MaybePromptAndOpen(path string) bool {
	editors := detectEditors()
	if len(editors) == 0 {
		// just exit if there are no editors available
		return false
	}
	opts := []string{}
	for _, c := range editors {
		opts = append(opts, "Open with "+c.name)
	}

	const openManually = "Open manually"
	opts = append(opts, openManually)

	choice, err := term.SelectFromList("Open the file now?", opts)
	if err != nil {
		term.OutputErrorAndExit("Error selecting editor: %v", err)
	}

	if choice == openManually {
		return false
	}

	var idx int
	for i, c := range opts {
		if c == choice {
			idx = i
			break
		}
	}

	if idx < len(editors) {
		sel := editors[idx]
		err = exec.Command(sel.cmd, append(sel.args, path)...).Start()
		if err != nil {
			term.OutputErrorAndExit("Error opening template: %v", err)
		}
		return true
	}

	return false
}

type editorCandidate struct {
	name        string
	cmd         string
	args        []string
	isJetBrains bool
}

const maxEditorOpts = 5

func detectEditors() []editorCandidate {
	guess := []editorCandidate{
		// Popular non-JetBrains launchers
		{"VS Code", "code", nil, false},
		{"Cursor", "cursor", nil, false},
		{"Zed", "zed", nil, false},
		{"Neovim", "nvim", nil, false},

		// JetBrains IDE-specific launchers
		{"IntelliJ IDEA", "idea", nil, true},
		{"GoLand", "goland", nil, true},
		{"PyCharm", "pycharm", nil, true},
		{"CLion", "clion", nil, true},
		{"WebStorm", "webstorm", nil, true},
		{"PhpStorm", "phpstorm", nil, true},
		{"DataGrip", "datagrip", nil, true},
		{"RubyMine", "rubymine", nil, true},
		{"Rider", "rider", nil, true},
		{"DataSpell", "dataspell", nil, true},

		// JetBrains universal CLI (2023.2+)
		{"JetBrains (jb)", "jb", []string{"open"}, true},

		{"Vim", "vim", nil, false},
		{"Nano", "nano", nil, false},
		{"Helix", "hx", nil, false},
		{"Micro", "micro", nil, false},
		{"Sublime Text", "subl", nil, false},
		{"TextMate", "mate", nil, false},
		{"Kakoune", "kak", nil, false},
		{"Emacs", "emacs", nil, false},
		{"Kate", "kate", nil, false},
	}
	pref := map[string]bool{}
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			// keep only the binary name, drop path/flags
			cmd := filepath.Base(strings.Fields(v)[0])
			pref[cmd] = true
		}
	}

	_, err := exec.LookPath("jb") // true if universal launcher exists
	jbOnPath := err == nil

	var found []editorCandidate
	for _, c := range guess {
		if _, err := exec.LookPath(c.cmd); err != nil {
			continue // not on PATH
		}

		// If jb is present, drop per-IDE launchers *unless* this exact cmd
		// is marked preferred by VISUAL/EDITOR.
		if jbOnPath && c.isJetBrains && !pref[c.cmd] {
			continue
		}
		found = append(found, c)
	}

	for cmd := range pref {
		if _, err := exec.LookPath(cmd); err == nil {
			already := false
			for _, c := range found {
				if c.cmd == cmd {
					already = true
					break
				}
			}
			if !already {
				found = append(found, editorCandidate{name: cmd, cmd: cmd})
			}
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		pi, pj := pref[found[i].cmd], pref[found[j].cmd]
		if pi == pj {
			return false // keep original order
		}
		return pi // true → i comes before j
	})
	if len(found) > maxEditorOpts {
		found = found[:maxEditorOpts]
	}

	return found
}
