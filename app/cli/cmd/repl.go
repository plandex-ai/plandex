package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/lib"
	"plandex-cli/term"
	"plandex-cli/types"
	"plandex-cli/version"
	shared "plandex-shared"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/plandex-ai/go-prompt"
	pstrings "github.com/plandex-ai/go-prompt/strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start interactive Plandex REPL",
	Run:   runRepl,
}

var cliSuggestions []prompt.Suggest
var projectPaths *types.ProjectPaths
var currentPrompt *prompt.Prompt

var replConfig *shared.PlanConfig

var sessionId string

func init() {
	RootCmd.AddCommand(replCmd)

	replCmd.Flags().BoolP("chat", "c", false, "Start in chat mode")
	replCmd.Flags().BoolP("tell", "t", false, "Start in tell mode")

	AddNewPlanFlags(replCmd)

	for _, config := range term.CliCommands {
		if config.Repl {
			desc := config.Desc
			if config.Alias != "" {
				desc = fmt.Sprintf("(\\%s) %s", config.Alias, desc)
			}
			cliSuggestions = append(cliSuggestions, prompt.Suggest{Text: "\\" + config.Cmd, Description: desc})
		}
	}
}

func setReplConfig() {
	res, apiErr := api.Client.GetPlanConfig(lib.CurrentPlanId)
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plan config: %v", apiErr.Msg)
	}
	replConfig = res
}

func runRepl(cmd *cobra.Command, args []string) {
	sessionId = uuid.New().String()

	term.SetIsRepl(true)

	auth.MustResolveAuthWithOrg()
	lib.MustResolveOrCreateProject()

	term.StartSpinner("")
	lib.LoadState()

	chatFlag, err := cmd.Flags().GetBool("chat")
	if err != nil {
		term.OutputErrorAndExit("Error getting chat flag: %v", err)
	}

	tellFlag, err := cmd.Flags().GetBool("tell")
	if err != nil {
		term.OutputErrorAndExit("Error getting tell flag: %v", err)
	}

	if chatFlag && tellFlag {
		term.OutputErrorAndExit("Cannot specify both --chat and --tell flags")
	}

	if chatFlag {
		lib.CurrentReplState.Mode = lib.ReplModeChat
		lib.WriteState()
	} else if tellFlag {
		lib.CurrentReplState.Mode = lib.ReplModeTell
		lib.WriteState()
	}

	afterNew := false

	if lib.CurrentPlanId == "" {
		os.Setenv("PLANDEX_DISABLE_SUGGESTIONS", "1")
		args := []string{}

		if noAuto {
			args = append(args, "--no-auto")
		} else if basicAuto {
			args = append(args, "--basic")
		} else if plusAuto {
			args = append(args, "--plus")
		} else if semiAuto {
			args = append(args, "--semi")
		} else if fullAuto {
			args = append(args, "--full")
		}

		if ossModels {
			args = append(args, "--oss")
		} else if strongModels {
			args = append(args, "--strong")
		} else if cheapModels {
			args = append(args, "--cheap")
		} else if dailyModels {
			args = append(args, "--daily")
		} else if reasoningModels {
			args = append(args, "--reasoning")
		} else if geminiPreviewModels {
			args = append(args, "--gemini-preview")
		}

		newCmd.Run(newCmd, args)
		os.Setenv("PLANDEX_DISABLE_SUGGESTIONS", "")
		afterNew = true
	}

	if !auth.Current.IntegratedModelsMode {
		lib.MustVerifyApiKeys()
	}

	projectPaths, err = fs.GetProjectPaths(fs.Cwd)
	if err != nil {
		color.New(term.ColorHiRed).Printf("Error getting project paths: %v\n", err)
	}

	setReplConfig()

	settings, apiErr := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting settings: %v", apiErr.Msg)
	}

	var printAutoFn func()
	var printModelFn func()
	if !afterNew {
		var didUpdateConfig bool
		var updatedConfig *shared.PlanConfig
		var updatedSettings *shared.PlanSettings
		didUpdateConfig, updatedConfig, printAutoFn = resolveAutoModeSilent(replConfig)
		updatedSettings, printModelFn = resolveModelPackSilent(settings)

		if didUpdateConfig {
			loadMapIfNeeded(replConfig, updatedConfig)
			removeMapIfNeeded(replConfig, updatedConfig)

			if updatedConfig != nil {
				replConfig = updatedConfig
			}
		}

		if updatedSettings != nil {
			settings = updatedSettings
		}
	}

	replWelcome(replWelcomeParams{
		afterNew:     afterNew,
		isHelp:       false,
		printAutoFn:  printAutoFn,
		printModelFn: printModelFn,
		config:       replConfig,
		packName:     settings.ModelPack.Name,
	})

	var p *prompt.Prompt
	p = prompt.New(
		func(in string) { executor(in, p) },
		prompt.WithPrefixCallback(func() string {
			// Get last part of current working directory
			// cwd := fs.Cwd
			// dirName := filepath.Base(cwd)

			// Build prefix with directory and mode indicator
			var modeIcon string
			if lib.CurrentReplState.Mode == lib.ReplModeTell {
				modeIcon = "⚡️"
				if replConfig.AutoApply && replConfig.AutoExec {
					modeIcon += "❗️" // warning reminder for auto apply and auto exec
				}
			} else if lib.CurrentReplState.Mode == lib.ReplModeChat {
				modeIcon = "💬"
			}
			return fmt.Sprintf("%s ", modeIcon)
		}),
		prompt.WithTitle("Plandex "+version.Version),
		prompt.WithSelectedSuggestionBGColor(prompt.LightGray),
		prompt.WithSuggestionBGColor(prompt.DarkGray),
		prompt.WithCompletionOnDown(),
		prompt.WithCompleter(completer),
		prompt.WithExecuteOnEnterCallback(executeOnEnter),
		prompt.WithHistory(lib.GetHistory()),
	)
	currentPrompt = p
	p.Run()
}

func getSuggestions() []prompt.Suggest {
	suggestions := []prompt.Suggest{}

	if lib.CurrentReplState.IsMulti {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\send", Description: "(\\s) Send the current prompt"},
			{Text: "\\multi", Description: "(\\m) Turn multi-line mode off"},
			{Text: "\\run", Description: "(\\r) Run a file through tell/chat based on current mode"},
			{Text: "\\quit", Description: "(\\q) Exit the REPL"},
		}...)

	}

	if lib.CurrentReplState.Mode == lib.ReplModeTell {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\chat", Description: "(\\ch) Switch to 'chat' mode to have a conversation without making changes"},
		}...)
	} else if lib.CurrentReplState.Mode == lib.ReplModeChat {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\tell", Description: "(\\t) Switch to 'tell' mode for implementation"},
		}...)
	}

	if !lib.CurrentReplState.IsMulti {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\multi", Description: "(\\m) Turn multi-line mode on"},
			{Text: "\\run", Description: "(\\r) Run a file through tell/chat based on current mode"},
			{Text: "\\quit", Description: "(\\q) Exit the REPL"},
		}...)
	}

	// Add help command suggestion
	suggestions = append(suggestions, prompt.Suggest{Text: "\\help", Description: "(\\h) REPL info and list of commands"})
	suggestions = append(suggestions, cliSuggestions...)

	for path := range projectPaths.ActivePaths {
		if path == "." {
			continue
		}

		isDir := projectPaths.ActiveDirs[path]

		if isDir {
			path += "/"
		}

		suggestions = append(suggestions, prompt.Suggest{Text: "@" + path})

		loadArgs := path
		if isDir {
			loadArgs += " -r"
		}
		suggestions = append(suggestions, prompt.Suggest{Text: "\\load " + loadArgs})

		if isDir {
			loadArgs = path
			loadArgs += " --map"
			suggestions = append(suggestions, prompt.Suggest{Text: "\\load " + loadArgs})

			loadArgs = path
			loadArgs += " --tree"
			suggestions = append(suggestions, prompt.Suggest{Text: "\\load " + loadArgs})
		}

		if filepath.Ext(path) == ".md" || filepath.Ext(path) == ".txt" {
			suggestions = append(suggestions, prompt.Suggest{Text: "\\run " + path})
		}
	}

	return suggestions
}

func executeOnEnter(p *prompt.Prompt, indentSize int) (int, bool) {
	input := p.Buffer().Text()
	cmd, _ := parseCommand(input)

	if cmd != "" {
		return 0, true
	}

	if lib.CurrentReplState.IsMulti {
		return 0, false
	}

	return 0, true
}

const cancelOpt = "Cancel"

func executor(in string, p *prompt.Prompt) {
	defer lib.WriteHistory(in)

	in = strings.TrimSpace(in)
	lines := strings.Split(in, "\n")
	lastLine := lines[len(lines)-1]
	lastLine = strings.TrimSpace(lastLine)

	trimmedInput := strings.TrimSpace(in)
	if trimmedInput == "" {
		return
	}
	// condense whitespace
	condensedInput := strings.Join(strings.Fields(trimmedInput), " ")

	// Handle plandex/pdx command prefix
	if strings.HasPrefix(lastLine, "plandex ") || strings.HasPrefix(lastLine, "pdx ") {
		fmt.Println()
		parts := strings.Fields(lastLine)
		if len(parts) > 1 {
			args := parts[1:] // Skip the "plandex" or "pdx" command
			_, err := lib.ExecPlandexCommandWithParams(args, lib.ExecPlandexCommandParams{
				SessionId: sessionId,
			})
			if err != nil {
				color.New(term.ColorHiRed).Printf("Error executing command: %v\n", err)
			}
		}
		fmt.Println()
		return
	}

	// Find the last \ or @ in the last line
	lastBackslashIndex := strings.LastIndex(lastLine, "\\")
	lastAtIndex := strings.LastIndex(lastLine, "@")

	var preservedBuffer string
	if len(lines) > 1 {
		preservedBuffer = strings.Join(lines[:len(lines)-1], "\n") + "\n"
	}

	suggestions, _, _ := completer(prompt.Document{Text: in})

	// Handle file references
	if lastAtIndex != -1 && lastAtIndex > lastBackslashIndex {
		paths := strings.Split(lastLine, "@")
		numPaths := len(paths)

		filteredPaths := []string{}

		for i, path := range paths {
			p := strings.TrimSpace(path)
			if i == 0 {
				// text before the @
				preservedBuffer += p + " "
				continue
			}

			if (p == "" || !projectPaths.ActivePaths[p]) && len(suggestions) > 0 && i == numPaths-1 {
				p = strings.Replace(suggestions[0].Text, "@", "", 1)
				filteredPaths = append(filteredPaths, p)
			} else if projectPaths.ActivePaths[p] {
				filteredPaths = append(filteredPaths, p)
			}
		}

		if len(filteredPaths) > 0 {
			args := []string{"load"}
			args = append(args, filteredPaths...)
			args = append(args, "-r")
			fmt.Println()
			_, err := lib.ExecPlandexCommandWithParams(args, lib.ExecPlandexCommandParams{
				SessionId: sessionId,
			})
			if err != nil {
				color.New(term.ColorHiRed).Printf("Error executing command: %v\n", err)
			}
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return
		}
	}

	// Handle commands
	if lastBackslashIndex != -1 {
		cmdString := strings.TrimSpace(lastLine[lastBackslashIndex+1:])
		if cmdString == "" {
			return
		}
		res := execWithInput(execWithInputParams{
			cmdString:          cmdString,
			in:                 condensedInput,
			lastBackslashIndex: lastBackslashIndex,
			preservedBuffer:    preservedBuffer,
			p:                  p,
			lastLine:           lastLine,
			condensedInput:     condensedInput,
			trimmedInput:       trimmedInput,
			lines:              lines,
			suggestions:        suggestions,
		})
		if res.shouldReturn {
			return
		}
		condensedInput = res.condensedInput
		trimmedInput = res.trimmedInput
	} else if len(lines) == 1 {
		// Check for likely accidental command inputs (with no backslash) and confirm with user
		var allCommands []string
		for replCmd := range lib.ReplCmdAliases {
			allCommands = append(allCommands, replCmd)
		}
		for _, config := range term.CliCommands {
			if config.Repl {
				allCommands = append(allCommands, config.Cmd)
			}
		}

		// Only suggest commands if they're close enough matches
		maybeCmds := findSimilarCommands(lastLine, allCommands)

		if len(maybeCmds) > 0 {
			res := suggestCmds(maybeCmds, getPromptOpt(lastLine))
			if res.shouldReturn {
				return
			}
			matchedCmd := res.matchedCmd
			if matchedCmd != "" {
				res := execWithInput(execWithInputParams{
					cmdString:          matchedCmd,
					in:                 condensedInput,
					lastBackslashIndex: lastBackslashIndex,
					preservedBuffer:    preservedBuffer,
					p:                  p,
					lastLine:           lastLine,
					condensedInput:     condensedInput,
					trimmedInput:       trimmedInput,
					lines:              lines,
					suggestions:        suggestions,
				})
				if res.shouldReturn {
					return
				}
				condensedInput = res.condensedInput
				trimmedInput = res.trimmedInput
			}
		}
	}

	// Handle non-command input based on mode
	if lib.CurrentReplState.Mode == lib.ReplModeTell {
		fmt.Println()
		args := []string{"tell", trimmedInput}
		var err error
		_, err = lib.ExecPlandexCommandWithParams(args, lib.ExecPlandexCommandParams{
			SessionId: sessionId,
		})
		if err != nil {
			color.New(term.ColorHiRed).Printf("Error executing tell: %v\n", err)
		}
	} else if lib.CurrentReplState.Mode == lib.ReplModeChat {
		fmt.Println()
		args := []string{"chat", trimmedInput}
		output, err := lib.ExecPlandexCommandWithParams(args, lib.ExecPlandexCommandParams{
			SessionId: sessionId,
		})
		if err != nil {
			color.New(term.ColorHiRed).Printf("Error executing chat: %v\n", err)
		}

		rx := regexp.MustCompile(`(?i)(switch|start|continue|begin|change|chang|move|proceed|go|transition)(ing)?( to | with | into )?(tell|implementation|coding|development)( mode)?`)

		if rx.MatchString(output) {
			fmt.Println()
			res, err := term.ConfirmYesNo("Switch to tell mode for implementation?")
			if err != nil {
				color.New(term.ColorHiRed).Printf("Error confirming yes/no: %v\n", err)
			}
			if res {
				lib.CurrentReplState.Mode = lib.ReplModeTell
				lib.WriteState()
				fmt.Println()
				color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" ⚡️ Tell mode is enabled ")
				fmt.Println()
				fmt.Println("Now that you're in tell mode, you can either begin the implementation based on the conversation so far, or you can send another prompt to begin the implementation with additional information or instructions.")
				fmt.Println()
				beginImplOpt := "Begin implementation"
				anotherPromptOpt := "Send another prompt"
				sel, err := term.SelectFromList("What would you like to do?", []string{beginImplOpt, anotherPromptOpt})
				if err != nil {
					color.New(term.ColorHiRed).Printf("Error selecting from list: %v\n", err)
				}
				if sel == beginImplOpt {
					fmt.Println()
					args := []string{"tell", "--from-chat"}
					_, err := lib.ExecPlandexCommandWithParams(args, lib.ExecPlandexCommandParams{
						SessionId: sessionId,
					})
					if err != nil {
						color.New(term.ColorHiRed).Printf("Error executing tell: %v\n", err)
					}
				}
			}
		}
	}
	fmt.Println()
}

func completer(in prompt.Document) ([]prompt.Suggest, pstrings.RuneNumber, pstrings.RuneNumber) {
	// Don't show suggestions if we're navigating history
	if currentPrompt.IsNavigatingHistory() {
		return []prompt.Suggest{}, 0, 0
	}

	endIndex := in.CurrentRuneIndex()

	lines := strings.Split(in.Text, "\n")
	currentLineNum := strings.Count(in.TextBeforeCursor(), "\n")

	// Don't show suggestions if we're not on the last line
	if currentLineNum < len(lines)-1 {
		return []prompt.Suggest{}, 0, 0
	}

	lastLine := lines[len(lines)-1]
	if strings.TrimSpace(lastLine) == "" && len(lines) > 1 {
		lastLine = lines[len(lines)-2]
	}

	// Handle plandex/pdx command prefix
	if strings.HasPrefix(lastLine, "plandex ") || strings.HasPrefix(lastLine, "pdx ") {
		parts := strings.Fields(lastLine)
		var prefix string
		if len(parts) > 1 {
			prefix = parts[len(parts)-1]
		}
		startIndex := endIndex - pstrings.RuneNumber(len(prefix))

		suggestions := []prompt.Suggest{}
		for _, config := range term.CliCommands {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        config.Cmd,
				Description: config.Desc,
			})
		}

		filtered := prompt.FilterFuzzy(suggestions, prefix, true)
		return filtered, startIndex, endIndex
	}

	// Find the last valid \ or @ in the current line
	lastBackslashIndex := -1
	lastAtIndex := -1

	// Helper function to check if character at index is valid (start of line or after space)
	isValidPosition := func(str string, index int) bool {
		if index <= 0 {
			return true // Start of line
		}
		return unicode.IsSpace(rune(str[index-1])) // After whitespace
	}

	// Find last valid backslash
	for i := len(lastLine) - 1; i >= 0; i-- {
		if lastLine[i] == '\\' && isValidPosition(lastLine, i) {
			lastBackslashIndex = i
			break
		}
	}

	// Find last valid @
	for i := len(lastLine) - 1; i >= 0; i-- {
		if lastLine[i] == '@' && isValidPosition(lastLine, i) {
			lastAtIndex = i
			break
		}
	}

	var w string
	var startIndex pstrings.RuneNumber

	if lastBackslashIndex == -1 && lastAtIndex == -1 {
		return []prompt.Suggest{}, 0, 0
	}

	// Use the rightmost special character
	if lastBackslashIndex > lastAtIndex {
		// Get everything after the last backslash
		w = lastLine[lastBackslashIndex:]
		startIndex = endIndex - pstrings.RuneNumber(len(w))
	} else if lastAtIndex != -1 {
		// Get everything after the last @
		w = lastLine[lastAtIndex:]
		startIndex = endIndex - pstrings.RuneNumber(len(w))
	}

	// Verify this is at the end of the line (allowing for trailing spaces)
	if !strings.HasSuffix(strings.TrimSpace(lastLine), strings.TrimSpace(w)) {
		return []prompt.Suggest{}, 0, 0
	}

	wTrimmed := strings.TrimSpace(strings.TrimPrefix(w, "\\"))
	parts := strings.Split(wTrimmed, " ")
	wCmd := parts[0]

	// For commands, verify it starts with an actual command
	if strings.HasPrefix(w, "\\") {
		isValidCommand := false
		for _, config := range term.CliCommands {
			if !config.Repl {
				continue
			}
			if strings.HasPrefix(config.Cmd, wCmd) ||
				(config.Alias != "" && strings.HasPrefix(config.Alias, wCmd)) {
				isValidCommand = true
				break
			}
		}
		// Also check built-in REPL commands
		if strings.HasPrefix("quit", wCmd) ||
			strings.HasPrefix("multi", wCmd) ||
			strings.HasPrefix("tell", wCmd) ||
			strings.HasPrefix("chat", wCmd) ||
			strings.HasPrefix("send", wCmd) ||
			strings.HasPrefix("run", wCmd) {
			isValidCommand = true
		}
		if !isValidCommand && wCmd != "" {
			return []prompt.Suggest{}, 0, 0
		}
	}

	fuzzySuggestions := prompt.FilterFuzzy(getSuggestions(), w, true)
	prefixMatches := prompt.FilterHasPrefix(getSuggestions(), w, true)

	runFilteredFuzzy := []prompt.Suggest{}
	runFilteredPrefixMatches := []prompt.Suggest{}
	for _, s := range fuzzySuggestions {
		if strings.HasPrefix(s.Text, "\\run ") {
			if wCmd == "run" {
				runFilteredFuzzy = append(runFilteredFuzzy, s)
			}
		} else {
			runFilteredFuzzy = append(runFilteredFuzzy, s)
		}
	}
	for _, s := range prefixMatches {
		if strings.HasPrefix(s.Text, "\\run ") {
			if wCmd == "run" {
				runFilteredPrefixMatches = append(runFilteredPrefixMatches, s)
			}
		} else {
			runFilteredPrefixMatches = append(runFilteredPrefixMatches, s)
		}
	}
	fuzzySuggestions = runFilteredFuzzy
	prefixMatches = runFilteredPrefixMatches

	loadFilteredFuzzy := []prompt.Suggest{}
	loadFilteredPrefixMatches := []prompt.Suggest{}
	for _, s := range fuzzySuggestions {
		if strings.HasPrefix(s.Text, "\\load ") {
			if wCmd == "load" {
				loadFilteredFuzzy = append(loadFilteredFuzzy, s)
			}
		} else {
			loadFilteredFuzzy = append(loadFilteredFuzzy, s)
		}
	}
	for _, s := range prefixMatches {
		if strings.HasPrefix(s.Text, "\\load ") {
			if wCmd == "load" {
				loadFilteredPrefixMatches = append(loadFilteredPrefixMatches, s)
			}
		} else {
			loadFilteredPrefixMatches = append(loadFilteredPrefixMatches, s)
		}
	}
	fuzzySuggestions = loadFilteredFuzzy
	prefixMatches = loadFilteredPrefixMatches

	if strings.TrimSpace(w) != "\\" {
		sort.Slice(prefixMatches, func(i, j int) bool {
			iTxt := prefixMatches[i].Text
			jTxt := prefixMatches[j].Text
			if iTxt == "\\chat" || iTxt == "\\tell" || iTxt == "\\multi" || iTxt == "\\quit" || iTxt == "\\send" || iTxt == "\\run" {
				return true
			}
			if jTxt == "\\chat" || jTxt == "\\tell" || jTxt == "\\multi" || jTxt == "\\quit" || jTxt == "\\send" || jTxt == "\\run" {
				return false
			}
			return prefixMatches[i].Text < prefixMatches[j].Text
		})
	}

	if len(prefixMatches) > 0 {
		// Remove prefix matches from fuzzy results to avoid duplicates
		prefixMatchSet := make(map[string]bool)
		for _, s := range prefixMatches {
			prefixMatchSet[s.Text] = true
		}

		nonPrefixFuzzy := make([]prompt.Suggest, 0)
		for _, s := range fuzzySuggestions {
			if !prefixMatchSet[s.Text] {
				nonPrefixFuzzy = append(nonPrefixFuzzy, s)
			}
		}

		fuzzySuggestions = append(prefixMatches, nonPrefixFuzzy...)
	}

	var aliasMatch string

	if lib.ReplCmdAliases[wTrimmed] != "" {
		aliasMatch = "\\" + lib.ReplCmdAliases[wTrimmed]
	} else {
		for _, s := range term.CliCommands {
			if s.Alias == wTrimmed {
				aliasMatch = "\\" + s.Cmd
				break
			}
		}
	}

	if aliasMatch != "" {
		// put the suggestion with the alias match at the beginning
		var matched prompt.Suggest
		found := false
		for _, s := range fuzzySuggestions {
			if s.Text == aliasMatch {
				matched = s
				found = true
				break
			}
		}
		if found {
			newSuggestions := []prompt.Suggest{}
			newSuggestions = append(newSuggestions, matched)
			for _, s := range fuzzySuggestions {
				if s.Text != aliasMatch {
					newSuggestions = append(newSuggestions, s)
				}
			}
			fuzzySuggestions = newSuggestions
		}
	}

	return fuzzySuggestions, startIndex, endIndex
}

type replWelcomeParams struct {
	afterNew     bool
	isHelp       bool
	printAutoFn  func()
	printModelFn func()
	packName     string
	config       *shared.PlanConfig
}

func replWelcome(params replWelcomeParams) {
	// print REPL welcome message and basic info
	// have to make these requests serially in case re-authentication is needed

	afterNew := params.afterNew
	isHelp := params.isHelp
	printAutoFn := params.printAutoFn
	printModelFn := params.printModelFn
	packName := params.packName

	plan, apiErr := api.Client.GetPlan(lib.CurrentPlanId)
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plan: %v", apiErr.Msg)
	}

	config := params.config
	if config == nil {
		config, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
		if apiErr != nil {
			term.OutputErrorAndExit("Error getting plan config: %v", apiErr.Msg)
		}
	}

	currentBranchesByPlanId, err := api.Client.GetCurrentBranchByPlanId(lib.CurrentProjectId, shared.GetCurrentBranchByPlanIdRequest{
		CurrentBranchByPlanId: map[string]string{
			lib.CurrentPlanId: lib.CurrentBranch,
		},
	})

	if err != nil {
		term.OutputErrorAndExit("Error getting current branches: %v", err)
	}

	term.StopSpinner()

	if !afterNew {
		fmt.Println()
	}

	color.New(color.FgHiWhite, color.BgBlue, color.Bold).Print(" 👋 Welcome to Plandex ")

	versionStr := version.Version
	if versionStr != "development" {
		color.New(color.FgHiWhite, color.BgHiBlack).Printf(" v%s ", versionStr)
	}

	fmt.Println()
	fmt.Println()

	fmt.Println(lib.GetCurrentPlanTable(plan, currentBranchesByPlanId, nil))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)

	var contextMode string
	if config.AutoLoadContext {
		contextMode = "auto"
	} else {
		contextMode = "manual"
	}

	filesStr := "%s for loading files into context"
	if contextMode == "auto" {
		filesStr += " manually (optional)"
	}
	filesStr += "\n"

	color.New(color.FgHiWhite).Printf("%s for commands\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\"))
	color.New(color.FgHiWhite).Printf(filesStr, color.New(term.ColorHiCyan, color.Bold).Sprint("@"))
	color.New(color.FgHiWhite).Printf("%s (\\h) for help\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\help"))
	color.New(color.FgHiWhite).Printf("%s (\\q) to exit\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\quit"))

	fmt.Println()

	if printAutoFn != nil {
		printAutoFn()
	} else {
		printAutoModeTable(config)
	}

	color.New(color.FgHiWhite).Printf("%s to change auto mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\set-auto"))
	color.New(color.FgHiWhite).Printf("%s to see config\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\config"))
	color.New(color.FgHiWhite).Printf("%s to customize config\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\set-config"))
	fmt.Println()

	if printModelFn != nil {
		printModelFn()
	} else {
		printModelPackTable(packName)
	}
	color.New(color.FgHiWhite).Printf("%s to see model details\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\models"))
	color.New(color.FgHiWhite).Printf("%s to change models\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\set-model"))

	showReplMode()
	showMultiLineMode()
	fmt.Println()

	if !isHelp {
		if lib.CurrentReplState.Mode == lib.ReplModeTell {
			color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" Describe a coding task 👇 ")
		} else {
			color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" Ask a question or chat 👇 ")
		}

		fmt.Println()
	}
}

func replHelp() {
	replWelcome(replWelcomeParams{
		afterNew: false,
		isHelp:   true,
	})
	term.PrintHelpAllCommands()
}

func showReplMode() {
	fmt.Println()
	if lib.CurrentReplState.Mode == lib.ReplModeTell {
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" ⚡️ Tell mode is enabled ")
		color.New(color.FgHiWhite).Printf("%s (\\ch) switch to chat mode to chat without writing code or making changes\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\chat"))
	} else if lib.CurrentReplState.Mode == lib.ReplModeChat {
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 💬 Chat mode is enabled ")
		color.New(color.FgHiWhite).Printf("%s (\\t) switch to tell mode to start writing code and implementing tasks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\tell"))
	}
	fmt.Println()
}

func showMultiLineMode() {
	if lib.CurrentReplState.IsMulti {
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 🔢 Multi-line mode is enabled ")
		fmt.Printf("%s to exit multi-line mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		fmt.Printf("%s for line breaks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\send"))
	} else {
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 1️⃣  Multi-line mode is disabled ")
		fmt.Printf("%s for multi-line editing mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
	}
}

func parseCommand(in string) (string, string) {
	in = strings.TrimSpace(in)
	lines := strings.Split(in, "\n")
	lastLine := lines[len(lines)-1]
	lastLine = strings.TrimSpace(lastLine)

	input := strings.TrimSpace(in)
	if input == "" {
		return "", ""
	}

	// Handle plandex/pdx command prefix
	if strings.HasPrefix(lastLine, "plandex ") || strings.HasPrefix(lastLine, "pdx ") {
		return lastLine, lastLine
	}

	// Find the last \ or @ in the last line
	lastBackslashIndex := strings.LastIndex(lastLine, "\\")
	lastAtIndex := strings.LastIndex(lastLine, "@")

	suggestions, _, _ := completer(prompt.Document{Text: in})

	// Handle file references
	if lastAtIndex != -1 && lastAtIndex > lastBackslashIndex {
		paths := strings.Split(lastLine, "@")
		split2 := strings.SplitN(lastLine, "@", 2)
		numPaths := len(paths)

		filteredPaths := []string{}

		for i, path := range paths {
			p := strings.TrimSpace(path)
			if i == 0 {
				// text before the @
				continue
			}

			if (p == "" || !projectPaths.ActivePaths[p]) && len(suggestions) > 0 && i == numPaths-1 {
				p = strings.Replace(suggestions[0].Text, "@", "", 1)
				filteredPaths = append(filteredPaths, p)
			} else if projectPaths.ActivePaths[p] {
				filteredPaths = append(filteredPaths, p)
			}
		}

		if len(filteredPaths) > 0 {
			res := ""
			for _, p := range filteredPaths {
				res += "@" + p + " "
			}
			return res, split2[1]
		}
	}

	// Handle commands
	if lastBackslashIndex != -1 {
		cmdString := strings.TrimSpace(lastLine[lastBackslashIndex+1:])
		if cmdString == "" {
			return "", ""
		}

		// Split into command and args
		parts := strings.Fields(cmdString)
		cmd := parts[0]
		args := parts[1:]

		// Handle built-in REPL commands
		switch cmd {
		case "quit", lib.ReplCmdAliases["quit"]:
			return "\\quit", "\\" + cmdString

		case "help", lib.ReplCmdAliases["help"]:
			return "\\help", "\\" + cmdString

		case "multi", lib.ReplCmdAliases["multi"]:
			return "\\multi", "\\" + cmdString

		case "send", lib.ReplCmdAliases["send"]:
			return "\\send", "\\" + cmdString

		case "tell", lib.ReplCmdAliases["tell"]:
			return "\\tell", "\\" + cmdString

		case "chat", lib.ReplCmdAliases["chat"]:
			return "\\chat", "\\" + cmdString

		case "run", lib.ReplCmdAliases["run"]:
			return "\\run", "\\" + cmdString

		default:
			// Check CLI commands
			var matchedCmd string

			for _, config := range term.CliCommands {

				if (cmd == config.Cmd || (config.Alias != "" && cmd == config.Alias)) && config.Repl {
					matchedCmd = config.Cmd
					break
				}
			}

			if matchedCmd == "" {
				for _, config := range term.CliCommands {
					if strings.HasPrefix(config.Cmd, cmd) && config.Repl {
						matchedCmd = config.Cmd
						break
					}
				}
			}

			if matchedCmd != "" {
				res := matchedCmd
				if len(args) > 0 {
					res += " " + strings.Join(args, " ")
				}
				return res, "\\" + cmdString
			}
		}
	}

	return "", ""
}

func isFileInProjectPaths(filePath string) bool {
	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	// Check if file is within any project path
	for path := range projectPaths.ActivePaths {
		projectAbs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, projectAbs) {
			return true
		}
	}
	return false
}

func handleRunCommand(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("run command requires exactly one file path argument")
	}

	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Build command based on current mode
	var cmdArgs []string
	if lib.CurrentReplState.Mode == lib.ReplModeTell {
		cmdArgs = []string{"tell", "-f", filePath}
	} else {
		cmdArgs = []string{"chat", "-f", filePath}
	}

	// Execute the command
	_, err := lib.ExecPlandexCommand(cmdArgs)
	if err != nil {
		return fmt.Errorf("error executing command: %v", err)
	}

	return nil
}

func getPromptOpt(cmd string) string {
	asPrompt := cmd
	if len(asPrompt) > 20 {
		asPrompt = asPrompt[:20] + "..."
	}
	return fmt.Sprintf("Send '%s' as a prompt to the AI model", asPrompt)
}

type suggestCmdsResult struct {
	shouldReturn bool
	matchedCmd   string
}

func suggestCmds(cmds []string, promptOpt string) suggestCmdsResult {
	var matchedCmd string

	fmt.Println()
	opts := []string{}

	for _, match := range cmds {
		opts = append(opts, "\\"+match)
	}
	opts = append(opts, cancelOpt, promptOpt)
	sel, err := term.SelectFromList("🤔 Did you mean to type one of these commands?", opts)
	if err != nil {
		color.New(term.ColorHiRed).Printf("Error selecting from list: %v\n", err)
	}
	if sel == cancelOpt {
		return suggestCmdsResult{shouldReturn: true}
	} else if sel != promptOpt {
		matchedCmd = strings.Replace(sel, "\\", "", 1)
	}

	return suggestCmdsResult{matchedCmd: matchedCmd}
}

type execWithInputParams struct {
	cmdString          string
	in                 string
	lastBackslashIndex int
	preservedBuffer    string
	p                  *prompt.Prompt
	lastLine           string
	condensedInput     string
	trimmedInput       string
	lines              []string
	suggestions        []prompt.Suggest
}

type execWithInputResult struct {
	shouldReturn   bool
	condensedInput string
	trimmedInput   string
}

func execWithInput(params execWithInputParams) execWithInputResult {
	cmdString := params.cmdString
	in := params.in
	lastBackslashIndex := params.lastBackslashIndex
	preservedBuffer := params.preservedBuffer
	lastLine := params.lastLine
	p := params.p
	condensedInput := params.condensedInput
	trimmedInput := params.trimmedInput
	lines := params.lines
	suggestions := params.suggestions

	// Split into command and args
	parts := strings.Fields(cmdString)
	cmd := parts[0]
	args := parts[1:]

	var fuzzyNEQCheckCmds []string
	for replCmd := range lib.ReplCmdAliases {
		if replCmd != cmd {
			fuzzyNEQCheckCmds = append(fuzzyNEQCheckCmds, replCmd)
		}
	}
	for _, config := range term.CliCommands {
		if !config.Repl {
			continue
		}
		if config.Cmd != cmd {
			fuzzyNEQCheckCmds = append(fuzzyNEQCheckCmds, config.Cmd)
		}
	}

	fuzzyNEQMatches := findSimilarCommands(cmd, fuzzyNEQCheckCmds)

	// Handle built-in REPL commands
	switch {
	case cmd == "quit" || cmd == lib.ReplCmdAliases["quit"]:
		lib.WriteHistory(in)
		os.Exit(0)

	case cmd == "help" || cmd == lib.ReplCmdAliases["help"]:
		if lastBackslashIndex > 0 {
			preservedBuffer += lastLine[:lastBackslashIndex]
		}
		replHelp()
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return execWithInputResult{shouldReturn: true}

	case cmd == "multi" || cmd == lib.ReplCmdAliases["multi"]:
		if lastBackslashIndex > 0 {
			preservedBuffer += lastLine[:lastBackslashIndex]
		}
		fmt.Println()
		lib.CurrentReplState.IsMulti = !lib.CurrentReplState.IsMulti
		showMultiLineMode()
		lib.WriteState()
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return execWithInputResult{shouldReturn: true}

	case cmd == "send" || cmd == lib.ReplCmdAliases["send"]:
		condensedSplit := strings.Split(condensedInput, "\\s")
		condensedInput = strings.TrimSpace(condensedSplit[0])
		condensedInput = strings.TrimSpace(condensedInput)

		trimmedSplit := strings.Split(trimmedInput, "\\s")
		trimmedInput = strings.TrimSpace(trimmedSplit[0])
		trimmedInput = strings.TrimSpace(trimmedInput)

		if condensedInput == "" {
			fmt.Println()
			fmt.Println("🤷‍♂️ No prompt to send")
			fmt.Println()
			return execWithInputResult{shouldReturn: true}
		}

	case cmd == "tell" || cmd == lib.ReplCmdAliases["tell"]:
		if lastBackslashIndex > 0 {
			preservedBuffer += lastLine[:lastBackslashIndex]
		}
		lib.CurrentReplState.Mode = lib.ReplModeTell
		lib.WriteState()
		showReplMode()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return execWithInputResult{shouldReturn: true}

	case cmd == "chat" || cmd == lib.ReplCmdAliases["chat"]:
		if lastBackslashIndex > 0 {
			preservedBuffer += lastLine[:lastBackslashIndex]
		}
		lib.CurrentReplState.Mode = lib.ReplModeChat
		lib.WriteState()
		showReplMode()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return execWithInputResult{shouldReturn: true}

	case cmd == "run" || cmd == lib.ReplCmdAliases["run"]:
		if lastBackslashIndex > 0 {
			preservedBuffer += lastLine[:lastBackslashIndex]
		}
		fmt.Println()
		if err := handleRunCommand(args); err != nil {
			color.New(term.ColorHiRed).Printf("Run command failed: %v\n", err)
		}
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return execWithInputResult{shouldReturn: true}

	default:
		// Check CLI commands
		var matchedCmd string

		for _, config := range term.CliCommands {
			if (cmd == config.Cmd || (config.Alias != "" && cmd == config.Alias)) && config.Repl {
				matchedCmd = config.Cmd
				break
			}
		}

		if matchedCmd == "" && len(suggestions) > 0 {
			matchedCmd = strings.Replace(suggestions[0].Text, "\\", "", 1)
			return execWithInput(execWithInputParams{
				cmdString:          matchedCmd,
				in:                 condensedInput,
				lastBackslashIndex: lastBackslashIndex,
				preservedBuffer:    preservedBuffer,
				p:                  p,
				lastLine:           lastLine,
				condensedInput:     condensedInput,
				trimmedInput:       trimmedInput,
				lines:              lines,
				suggestions:        suggestions,
			})
		}

		if matchedCmd == "" {

			promptOpt := getPromptOpt(cmd)
			if len(fuzzyNEQMatches) > 0 {
				res := suggestCmds(fuzzyNEQMatches, promptOpt)
				if res.shouldReturn {
					return execWithInputResult{shouldReturn: true}
				}
				matchedCmd = res.matchedCmd
				return execWithInput(execWithInputParams{
					cmdString:          matchedCmd,
					in:                 condensedInput,
					lastBackslashIndex: lastBackslashIndex,
					preservedBuffer:    preservedBuffer,
					p:                  p,
					lastLine:           lastLine,
					condensedInput:     condensedInput,
					trimmedInput:       trimmedInput,
					lines:              lines,
					suggestions:        suggestions,
				})
			} else if len(lines) == 1 && strings.HasPrefix(trimmedInput, "\\") {
				showCmdsOpt := "Show available commands"
				opts := []string{cancelOpt, showCmdsOpt, promptOpt}
				sel, err := term.SelectFromList("🤔 Couldn't find a matching command. What do you want to do?", opts)
				if err != nil {
					color.New(term.ColorHiRed).Printf("Error selecting from list: %v\n", err)
				}
				if sel == cancelOpt {
					return execWithInputResult{shouldReturn: true}
				} else if sel == showCmdsOpt {
					replHelp()
					fmt.Println()
					return execWithInputResult{shouldReturn: true}
				}
			}
		}

		if matchedCmd != "" {
			// fmt.Println("> plandex " + config.Cmd)
			if lastBackslashIndex > 0 {
				preservedBuffer += lastLine[:lastBackslashIndex]
			}
			fmt.Println()
			execArgs := []string{matchedCmd}
			if matchedCmd == "continue" && chatOnly {
				execArgs = append(execArgs, "--chat")
			}
			execArgs = append(execArgs, args...)
			_, err := lib.ExecPlandexCommandWithParams(execArgs, lib.ExecPlandexCommandParams{
				SessionId: sessionId,
			})
			if err != nil {
				color.New(term.ColorHiRed).Printf("Error executing command: %v\n", err)
			}
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}

			if strings.HasPrefix(matchedCmd, "set-auto") || strings.HasPrefix(matchedCmd, "set-config") {
				term.StartSpinner("")
				setReplConfig()
				term.StopSpinner()
			}
			return execWithInputResult{shouldReturn: true}
		}
	}

	return execWithInputResult{
		condensedInput: condensedInput,
		trimmedInput:   trimmedInput,
	}
}

func findSimilarCommands(input string, commands []string) []string {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)
	input = strings.Trim(input, "/")

	// Get ranked matches
	ranks := fuzzy.RankFind(input, commands)

	// Filter strictly by distance
	var filtered []string
	for _, rank := range ranks {
		// include if either is a substring of the other
		if strings.Contains(rank.Target, input) || strings.Contains(input, rank.Target) {
			filtered = append(filtered, rank.Target)
			continue
		}

		// Normalize threshold based on command length
		maxLen := len(input)
		if len(rank.Target) > maxLen {
			maxLen = len(rank.Target)
		}

		threshold := 4 // Base threshold
		if maxLen < 5 {
			threshold = 1 // Stricter for very short commands
		}

		if rank.Distance <= threshold {
			filtered = append(filtered, rank.Target)
		}
	}

	return filtered
}
