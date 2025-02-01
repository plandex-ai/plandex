package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/lib"
	"plandex/term"
	"plandex/types"
	"plandex/version"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"

	"github.com/plandex-ai/go-prompt"
	pstrings "github.com/plandex-ai/go-prompt/strings"
)

var cliSuggestions []prompt.Suggest
var projectPaths *types.ProjectPaths

type replMode string

const (
	replModeTell replMode = "tell"
	replModeChat replMode = "chat"
)

type ReplState struct {
	mode    replMode
	isMulti bool
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start interactive Plandex REPL",
	Run:   runRepl,
}

var replState ReplState = ReplState{
	mode:    replModeTell,
	isMulti: false,
}

func init() {
	RootCmd.AddCommand(replCmd)

	replCmd.Flags().BoolP("chat", "c", false, "Start in chat mode")
	replCmd.Flags().BoolP("tell", "t", false, "Start in tell mode")

	for _, config := range term.CliCommands {
		if config.Repl {
			cliSuggestions = append(cliSuggestions, prompt.Suggest{Text: "\\" + config.Cmd, Description: config.Desc})
		}
	}
}

func runRepl(cmd *cobra.Command, args []string) {
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
		replState.mode = replModeChat
	} else if tellFlag {
		replState.mode = replModeTell
	}

	term.StartSpinner("")

	term.SetIsRepl(true)

	auth.MustResolveAuthWithOrg()
	lib.MustResolveOrCreateProject()

	if lib.CurrentPlanId == "" {
		newCmd.Run(newCmd, []string{})
	}

	projectPaths, err = fs.GetProjectPaths(fs.Cwd)
	if err != nil {
		color.New(term.ColorHiRed).Printf("Error getting project paths: %v\n", err)
	}

	replWelcome(false)

	var p *prompt.Prompt
	p = prompt.New(
		func(in string) { executor(in, p) },
		prompt.WithPrefix("👉 "),
		prompt.WithTitle("Plandex "+version.Version),
		prompt.WithSelectedSuggestionBGColor(prompt.LightGray),
		prompt.WithSuggestionBGColor(prompt.DarkGray),
		prompt.WithCompletionOnDown(),
		prompt.WithCompleter(completer),
		prompt.WithExecuteOnEnterCallback(executeOnEnter),
	)
	p.Run()
}

func getSuggestions() []prompt.Suggest {
	suggestions := []prompt.Suggest{}

	if replState.isMulti {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\send", Description: "Send the current prompt"},
			{Text: "\\multi", Description: "Turn multi-line mode off"},
			{Text: "\\quit", Description: "Exit the REPL"},
		}...)

	}

	if replState.mode == replModeTell {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\chat", Description: "Switch to 'chat' mode to have a conversation without making changes"},
		}...)
	} else if replState.mode == replModeChat {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\tell", Description: "Switch to 'tell' mode for implementation"},
		}...)
	}

	if !replState.isMulti {
		suggestions = append(suggestions, []prompt.Suggest{
			{Text: "\\multi", Description: "Turn multi-line mode on"},
			{Text: "\\quit", Description: "Exit the REPL"},
		}...)
	}

	// Add help command suggestion
	suggestions = append(suggestions, prompt.Suggest{Text: "\\help", Description: "REPL info and list of commands"})
	suggestions = append(suggestions, cliSuggestions...)

	for path := range projectPaths.ActivePaths {
		if path == "." {
			continue
		}
		suggestions = append(suggestions, prompt.Suggest{Text: "@" + path})
	}

	return suggestions
}

func executeOnEnter(p *prompt.Prompt, indentSize int) (int, bool) {
	if replState.isMulti {
		input := p.Buffer().Text()
		input = strings.TrimSpace(input)
		lines := strings.Split(input, "\n")
		lastLine := lines[len(lines)-1]
		lastLine = strings.TrimSpace(lastLine)
		lastLineWords := strings.Fields(lastLine)
		lastLineLastWord := ""
		if len(lastLineWords) > 0 {
			lastLineLastWord = lastLineWords[len(lastLineWords)-1]
		}

		if strings.HasPrefix(lastLineLastWord, "\\") ||
			strings.HasPrefix(lastLineLastWord, "@") { // @file
			return 0, true
		}

		return 0, false
	}

	return 0, true
}

func executor(in string, p *prompt.Prompt) {
	in = strings.TrimSpace(in)
	lines := strings.Split(in, "\n")
	lastLine := lines[len(lines)-1]
	lastLine = strings.TrimSpace(lastLine)

	input := strings.TrimSpace(in)
	if input == "" {
		return
	}

	// Find the last \ or @ in the last line
	lastBackslashIndex := strings.LastIndex(lastLine, "\\")
	lastAtIndex := strings.LastIndex(lastLine, "@")

	var preservedBuffer string
	if len(lines) > 1 {
		preservedBuffer = strings.Join(lines[:len(lines)-1], "\n") + "\n"
	}

	// Handle file references
	if lastAtIndex != -1 && lastAtIndex > lastBackslashIndex {
		paths := strings.Split(lastLine, "@")
		numPaths := len(paths)
		filteredPaths := []string{}
		for i, path := range paths {
			p := strings.TrimSpace(path)
			if p == "" {
				continue
			} else if i == 0 {
				preservedBuffer += p + " "
			}

			if !projectPaths.ActivePaths[p] && len(latestFilteredSuggestions) > 0 && i == numPaths-1 {
				p = strings.Replace(latestFilteredSuggestions[0].Text, "@", "", 1)
				filteredPaths = append(filteredPaths, p)
			} else if projectPaths.ActivePaths[p] {
				filteredPaths = append(filteredPaths, p)
			}
		}

		args := []string{"load"}
		args = append(args, filteredPaths...)
		args = append(args, "-r")
		execPlandexCommand(args)
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return
	}

	// Handle commands
	if lastBackslashIndex != -1 {
		cmdString := strings.TrimSpace(lastLine[lastBackslashIndex+1:])
		if cmdString == "" {
			return
		}

		// Split into command and args
		parts := strings.Fields(cmdString)
		cmd := parts[0]
		args := parts[1:]

		// Handle built-in REPL commands
		switch {
		case cmd == "quit" || strings.HasPrefix("quit", cmd):
			os.Exit(0)

		case cmd == "help" || strings.HasPrefix("help", cmd):
			if lastBackslashIndex > 0 {
				preservedBuffer += lastLine[:lastBackslashIndex]
			}
			replHelp()
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return

		case cmd == "multi" || strings.HasPrefix("multi", cmd):
			if lastBackslashIndex > 0 {
				preservedBuffer += lastLine[:lastBackslashIndex]
			}
			fmt.Println()
			if replState.isMulti {
				replState.isMulti = false
				color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 🙅‍♂️ Multi-line mode is disabled ")
				fmt.Printf("%s for multi-line editing mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
				fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
			} else {
				replState.isMulti = true
				color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 📖 Multi-line mode is enabled ")
				fmt.Printf("%s to exit multi-line mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
				fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\send"))
				fmt.Printf("%s for line breaks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
			}
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return

		case cmd == "send" || strings.HasPrefix("send", cmd):
			input = strings.TrimSuffix(input, "\\send")
			input = strings.TrimSpace(input)
			if input == "" {
				fmt.Println()
				fmt.Println("🤷‍♂️ No prompt to send")
				fmt.Println()
				return
			}

		case cmd == "tell" || strings.HasPrefix("tell", cmd):
			if lastBackslashIndex > 0 {
				preservedBuffer += lastLine[:lastBackslashIndex]
			}
			replState.mode = replModeTell
			fmt.Println()
			color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 💻 Tell mode is enabled ")
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return

		case cmd == "chat" || strings.HasPrefix("chat", cmd):
			if lastBackslashIndex > 0 {
				preservedBuffer += lastLine[:lastBackslashIndex]
			}
			replState.mode = replModeChat
			fmt.Println()
			color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 💬 Chat mode is enabled ")
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return

		default:
			// Check CLI commands
			for _, config := range term.CliCommands {
				if (cmd == config.Cmd || (config.Alias != "" && cmd == config.Alias) || strings.HasPrefix(config.Cmd, cmd)) && config.Repl {
					if lastBackslashIndex > 0 {
						preservedBuffer += lastLine[:lastBackslashIndex]
					}
					fmt.Println()
					execArgs := []string{config.Cmd}
					execArgs = append(execArgs, args...)
					err := execPlandexCommand(execArgs)
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
		}
	}

	// Handle non-command input based on mode
	if replState.mode == replModeTell {
		args := []string{"tell", input}
		err := execPlandexCommand(args)
		if err != nil {
			color.New(term.ColorHiRed).Printf("Error executing tell: %v\n", err)
		}
	} else if replState.mode == replModeChat {
		args := []string{"chat", input}
		err := execPlandexCommand(args)
		if err != nil {
			color.New(term.ColorHiRed).Printf("Error executing chat: %v\n", err)
		}
	}

	fmt.Println()
}

var latestFilteredSuggestions []prompt.Suggest

func completer(in prompt.Document) ([]prompt.Suggest, pstrings.RuneNumber, pstrings.RuneNumber) {
	endIndex := in.CurrentRuneIndex()

	lines := strings.Split(in.Text, "\n")
	currentLineNum := strings.Count(in.TextBeforeCursor(), "\n")

	// Don't show suggestions if we're not on the last line
	if currentLineNum < len(lines)-1 {
		latestFilteredSuggestions = []prompt.Suggest{}
		return []prompt.Suggest{}, 0, 0
	}

	lastLine := lines[len(lines)-1]
	if strings.TrimSpace(lastLine) == "" && len(lines) > 1 {
		lastLine = lines[len(lines)-2]
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
		latestFilteredSuggestions = []prompt.Suggest{}
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
		latestFilteredSuggestions = []prompt.Suggest{}
		return []prompt.Suggest{}, 0, 0
	}

	// For commands, verify it starts with an actual command
	if strings.HasPrefix(w, "\\") {
		isValidCommand := false
		wTrimmed := strings.TrimSpace(strings.TrimPrefix(w, "\\"))
		for _, config := range term.CliCommands {
			if strings.HasPrefix(config.Cmd, wTrimmed) ||
				(config.Alias != "" && strings.HasPrefix(config.Alias, wTrimmed)) {
				isValidCommand = true
				break
			}
		}
		// Also check built-in REPL commands
		if strings.HasPrefix("quit", wTrimmed) ||
			strings.HasPrefix("multi", wTrimmed) ||
			strings.HasPrefix("tell", wTrimmed) ||
			strings.HasPrefix("chat", wTrimmed) ||
			strings.HasPrefix("send", wTrimmed) {
			isValidCommand = true
		}
		if !isValidCommand && wTrimmed != "" {
			latestFilteredSuggestions = []prompt.Suggest{}
			return []prompt.Suggest{}, 0, 0
		}
	}

	fuzzySuggestions := prompt.FilterFuzzy(getSuggestions(), w, true)

	// Rest of the existing sorting logic
	prefixMatches := prompt.FilterHasPrefix(getSuggestions(), w, true)

	if strings.TrimSpace(w) != "\\" {
		sort.Slice(prefixMatches, func(i, j int) bool {
			iTxt := prefixMatches[i].Text
			jTxt := prefixMatches[j].Text
			if iTxt == "\\chat" || iTxt == "\\tell" || iTxt == "\\multi" || iTxt == "\\quit" || iTxt == "\\send" {
				return true
			}
			if jTxt == "\\chat" || jTxt == "\\tell" || jTxt == "\\multi" || jTxt == "\\quit" || jTxt == "\\send" {
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

	latestFilteredSuggestions = fuzzySuggestions
	return fuzzySuggestions, startIndex, endIndex
}

// execPlandexCommand spawns the same binary, wiring std streams directly so you
// don't have to capture output. Any os.Exit calls in the child won't kill your REPL.
func execPlandexCommand(args []string) error {
	// Subprocess runs the same binary (os.Args[0]) with your desired args.
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "PLANDEX_REPL=1")

	// Connect the child's standard streams to the parent.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run it. If the child calls os.Exit(1), only that child is killed,
	// and Run() will return an error here. Your REPL stays alive.
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		// If it's *not* an exit error, it might be a startup error (like file not found).
		return err
	}
	return nil
}

func replWelcome(isHelp bool) {
	// print REPL welcome message and basic info
	errCh := make(chan error, 2)
	var plan *shared.Plan
	var config *shared.PlanConfig

	go func() {
		var apiErr *shared.ApiError
		plan, apiErr = api.Client.GetPlan(lib.CurrentPlanId)
		if apiErr != nil {
			errCh <- fmt.Errorf("Error getting plan: %v", apiErr.Msg)
		}
		errCh <- nil
	}()

	go func() {
		var apiErr *shared.ApiError
		config, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
		if apiErr != nil {
			errCh <- fmt.Errorf("Error getting plan config: %v", apiErr.Msg)
		}
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			term.OutputErrorAndExit("%v", err)
		}
	}

	term.StopSpinner()

	fmt.Println()
	color.New(color.FgHiWhite, color.BgBlue, color.Bold).Print(" 👋 Welcome to the Plandex REPL ")

	versionStr := version.Version
	if versionStr != "development" {
		color.New(color.FgHiWhite, color.BgHiBlack).Printf(" v%s ", versionStr)
	}

	fmt.Println()
	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)

	table.SetHeader([]string{"Current Plan", "Branch", "REPL Mode", "Auto Mode", "Context"})

	var replModeIcon string
	if replState.mode == replModeTell {
		replModeIcon = "💻"
	} else if replState.mode == replModeChat {
		replModeIcon = "💬"
	}

	var contextMode string
	if config.AutoLoadContext {
		contextMode = "Auto"
	} else {
		contextMode = "Manual"
	}

	table.Append([]string{
		color.New(term.ColorHiGreen, color.Bold).Sprint(plan.Name),
		lib.CurrentBranch,
		strings.Title(string(replState.mode)) + " " + replModeIcon,
		shared.AutoModeLabels[config.AutoMode],
		contextMode,
	})

	table.Render()

	fmt.Println()

	color.New(color.FgHiWhite).Printf("%s for commands\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\"))
	color.New(color.FgHiWhite).Printf("%s for loading files into context\n", color.New(term.ColorHiCyan, color.Bold).Sprint("@"))

	if replState.mode == replModeTell {
		color.New(color.FgHiWhite).Printf("%s for chat mode → chat without writing code\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\chat"))
	} else {
		color.New(color.FgHiWhite).Printf("%s for tell mode → implement tasks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\tell"))
	}

	helpFn := func() {
		color.New(color.FgHiWhite).Printf("%s for help\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\help"))
	}
	if replState.isMulti {
		color.New(color.FgHiWhite).Printf("%s to exit multi-line mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\send"))
		helpFn()
		fmt.Printf("%s for line breaks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
	} else {
		color.New(color.FgHiWhite).Printf("%s for multi-line editing mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		helpFn()
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
	}

	fmt.Println()

	if !isHelp {
		if replState.mode == replModeTell {
			color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" 💻 Describe a coding task ")
		} else {
			color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" 💬 Ask a question or chat ")
		}

		fmt.Println()
	}
}

func replHelp() {
	replWelcome(true)
	term.PrintHelpAllCommands()
}
