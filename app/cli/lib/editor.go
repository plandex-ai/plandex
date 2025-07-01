package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/term"
	shared "plandex-shared"
	"sort"
	"strings"
)

func MaybePromptAndOpen(path string, defaultConfig *shared.PlanConfig, planConfig *shared.PlanConfig) bool {
	var cmd string
	var args []string
	var openManually bool

	var checkConfig *shared.PlanConfig
	if planConfig != nil {
		checkConfig = planConfig
	} else if defaultConfig != nil {
		checkConfig = defaultConfig
	} else {
		term.OutputErrorAndExit("Missing config")
	}

	if checkConfig.EditorOpenManually {
		return false
	}

	if checkConfig.EditorCommand == "" {
		editorRes := SelectEditor(true)
		cmd = editorRes.Cmd
		args = editorRes.Args
		openManually = editorRes.OpenManually

		// update the default editor config
		toUpdateDefault := *defaultConfig
		toUpdateDefault.Editor = editorRes.Name
		toUpdateDefault.EditorCommand = cmd
		toUpdateDefault.EditorArgs = args
		toUpdateDefault.EditorOpenManually = openManually

		apiErr := api.Client.UpdateDefaultPlanConfig(shared.UpdateDefaultPlanConfigRequest{
			Config: &toUpdateDefault,
		})
		if apiErr != nil {
			term.OutputErrorAndExit("Error updating default config: %v", apiErr)
		}

		// also update the current plan config
		if planConfig != nil {
			toUpdate := *planConfig
			toUpdate.Editor = editorRes.Name
			toUpdate.EditorCommand = cmd
			toUpdate.EditorArgs = args
			toUpdate.EditorOpenManually = openManually
			apiErr = api.Client.UpdatePlanConfig(CurrentPlanId, shared.UpdatePlanConfigRequest{
				Config: &toUpdate,
			})
			if apiErr != nil {
				term.OutputErrorAndExit("Error updating plan config: %v", apiErr)
			}
		}
	} else {
		cmd = checkConfig.EditorCommand
		args = checkConfig.EditorArgs
	}

	if openManually {
		return false
	}

	err := exec.Command(cmd, append(args, path)...).Start()
	if err != nil {
		term.OutputErrorAndExit("Error opening template: %v", err)
	}

	return true
}

type SelectEditorResult struct {
	Name         string
	Cmd          string
	Args         []string
	OpenManually bool
}

func SelectEditor(includeOpenManuallyOpt bool) SelectEditorResult {
	editors := detectEditors()

	opts := []string{}
	for _, c := range editors {
		opts = append(opts, c.name)
	}
	const otherOpt = "Other (custom command)"
	opts = append(opts, otherOpt)

	const openManuallyOpt = "Open files manually"
	if includeOpenManuallyOpt {
		opts = append(opts, openManuallyOpt)
	}

	choice, err := term.SelectFromList("What's your preferred editor?", opts)
	if err != nil {
		term.OutputErrorAndExit("Error selecting editor: %v", err)
	}

	var name string
	var cmd string
	var args []string

	if choice == otherOpt {
		choice, err = term.GetRequiredUserStringInput("Enter the command to open the editor")
		if err != nil {
			term.OutputErrorAndExit("Error getting editor command: %v", err)
		}
		name = choice
		parts := strings.Fields(choice)
		if len(parts) == 0 {
			term.OutputErrorAndExit("Invalid editor command: %s", choice)
		}
		cmd = parts[0]
		if len(parts) > 1 {
			args = parts[1:]
		}
	} else if choice == openManuallyOpt {
		return SelectEditorResult{
			Name:         "Open manually",
			Cmd:          "",
			Args:         []string{},
			OpenManually: true,
		}
	} else {
		var candidate editorCandidate
		for _, c := range editors {
			if c.name == choice {
				candidate = c
				break
			}
		}
		name = candidate.name
		cmd = candidate.cmd
		args = candidate.args
	}

	return SelectEditorResult{
		Name: name,
		Cmd:  cmd,
		Args: args,
	}
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
