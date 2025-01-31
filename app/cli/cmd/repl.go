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
	term.StartSpinner("")

	term.SetIsRepl(true)

	auth.MustResolveAuthWithOrg()
	lib.MustResolveOrCreateProject()

	if lib.CurrentPlanId == "" {
		newCmd.Run(newCmd, []string{})
	}

	var err error
	projectPaths, err = fs.GetProjectPaths(fs.Cwd)
	if err != nil {
		color.New(term.ColorHiRed).Printf("Error getting project paths: %v\n", err)
	}

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
	color.New(color.FgHiWhite, color.BgHiBlack).Printf(" v%s ", version.Version)
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
		contextMode = "Auto-load"
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
	if replState.isMulti {
		color.New(color.FgHiWhite).Printf("%s to exit multi-line mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\send"))
		fmt.Printf("%s for line breaks\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
	} else {
		color.New(color.FgHiWhite).Printf("%s for multi-line editing mode\n", color.New(term.ColorHiCyan, color.Bold).Sprint("\\multi"))
		fmt.Printf("%s to send prompt\n", color.New(term.ColorHiCyan, color.Bold).Sprint("enter"))
	}

	fmt.Println()

	if replState.mode == replModeTell {
		color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" 💻 Describe a coding task ")
	} else {
		color.New(color.FgHiWhite, color.BgBlue, color.Bold).Println(" 💬 Ask a question or chat ")
	}

	fmt.Println()

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
	// fmt.Println(in)

	// this ensures it works with multi-line input
	in = strings.TrimSpace(in)
	lines := strings.Split(in, "\n")
	lastLine := lines[len(lines)-1]
	lastLine = strings.TrimSpace(lastLine)

	if lastLine == "" {
		return
	}

	lastLineWords := strings.Fields(lastLine)
	lastLineLastWord := ""
	if len(lastLineWords) > 0 {
		lastLineLastWord = lastLineWords[len(lastLineWords)-1]
	}

	var preservedBuffer string
	if len(lines) > 1 {
		// Exclude the command line from preserved buffer
		preservedBuffer = strings.Join(lines[:len(lines)-1], "\n") + "\n"
	}

	isFile := strings.HasPrefix(lastLineLastWord, "@")
	if isFile {
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

	isSend := false
	isCommand := false

	cmd := strings.TrimPrefix(lastLineLastWord, "\\")
	wordSplit := strings.Split(lastLine, lastLineLastWord)
	var beforeCmd string
	if len(wordSplit) > 0 {
		beforeCmd = wordSplit[0]
	}

	if cmd == "quit" || strings.HasPrefix("quit", cmd) {
		os.Exit(0)
	}

	if cmd == "multi" || strings.HasPrefix("multi", cmd) {
		if beforeCmd != "" {
			preservedBuffer += beforeCmd
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
	}

	if cmd == "send" || strings.HasPrefix("send", cmd) {
		isSend = true
	}

	if cmd == "tell" || strings.HasPrefix("tell", cmd) {
		if beforeCmd != "" {
			preservedBuffer += beforeCmd
		}
		replState.mode = replModeTell
		fmt.Println()
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 💻 Tell mode is enabled ")
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return
	}

	if cmd == "chat" || strings.HasPrefix("chat", cmd) {
		if beforeCmd != "" {
			preservedBuffer += beforeCmd
		}
		replState.mode = replModeChat
		fmt.Println()
		color.New(color.BgMagenta, color.FgHiWhite, color.Bold).Println(" 💬 Chat mode is enabled ")
		fmt.Println()
		if preservedBuffer != "" {
			p.InsertTextMoveCursor(preservedBuffer, true)
		}
		return
	}

	if !isSend {
		if strings.HasPrefix(lastLineLastWord, "\\") && lastLineLastWord != "\\" {
			isCommand = true
		}
	}

	if isCommand {
		// Split the command and arguments
		args := strings.Fields(cmd)
		if len(args) > 0 {
			cmd := args[0]
			rest := args[1:]

			// Find the command in CliCommands
			for _, config := range term.CliCommands {
				if cmd == config.Cmd || (config.Alias != "" && cmd == config.Alias) || strings.HasPrefix(config.Cmd, cmd) {
					if !config.Repl {
						continue
					}
					if beforeCmd != "" {
						preservedBuffer += beforeCmd
					}

					fmt.Println()

					cmd = config.Cmd

					// Execute the command with arguments
					execArgs := []string{cmd}
					execArgs = append(execArgs, rest...)
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
			color.New(term.ColorHiRed).Printf("🤷‍♂️ Unknown command: %s\n", args[0])
			fmt.Println()
			if preservedBuffer != "" {
				p.InsertTextMoveCursor(preservedBuffer, true)
			}
			return
		}
	}

	var input string
	if len(lines) > 0 {
		input := strings.Join(lines, "\n")
		if isSend {
			input = strings.TrimSuffix(input, "\\send")
			input = strings.TrimSpace(input)
		}
	}
	if input == "" {
		if isSend {
			fmt.Println()
			fmt.Println("🤷‍♂️ No prompt to send")
			fmt.Println()
			return
		}

		// write a newline to the buffer
		p.InsertTextMoveCursor("\n", true)
		return
	}

	// Handle non-command input based on mode
	if replState.mode == replModeTell {
		// Process the tell command
		args := []string{"tell", input}
		err := execPlandexCommand(args)
		if err != nil {
			color.New(term.ColorHiRed).Printf("Error executing tell: %v\n", err)
		}
	} else if replState.mode == replModeChat {
		// Process the chat command
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
	w := in.GetWordBeforeCursor()

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

	// For both \ and @ symbols, only show suggestions if they're at the start or end of the line
	if strings.HasPrefix(w, "\\") || strings.HasPrefix(w, "@") {
		trimmedLastLine := strings.TrimSpace(lastLine)
		// Don't show suggestions if the symbol is in the middle
		// (not at start and not at end)
		if !strings.HasPrefix(trimmedLastLine, w) && !strings.HasSuffix(trimmedLastLine, w) {
			latestFilteredSuggestions = []prompt.Suggest{}
			return []prompt.Suggest{}, 0, 0
		}
	}

	if !strings.HasSuffix(strings.TrimSpace(in.Text), strings.TrimSpace(w)) {
		latestFilteredSuggestions = []prompt.Suggest{}
		return []prompt.Suggest{}, 0, 0
	}

	lastLineWords := strings.Fields(lastLine)
	var lastLineLastWord string
	if len(lastLineWords) > 0 {
		lastLineLastWord = lastLineWords[len(lastLineWords)-1]
	}

	// Add special handling for multi-line mode
	if replState.isMulti {
		// If w is empty but lastLine/lastWord has content and starts with a prefix we care about
		if strings.TrimSpace(w) == "" &&
			(strings.HasPrefix(strings.TrimSpace(lastLineLastWord), "\\") ||
				strings.HasPrefix(strings.TrimSpace(lastLineLastWord), "@")) {
			w = strings.TrimSpace(lastLineLastWord)
		}
	}

	if w == "" || !(strings.HasPrefix(w, "\\") || strings.HasPrefix(w, "@")) {
		latestFilteredSuggestions = []prompt.Suggest{}
		return []prompt.Suggest{}, 0, 0
	}

	startIndex := endIndex - pstrings.RuneCountInString(w)

	fuzzySuggestions := prompt.FilterFuzzy(getSuggestions(), w, true)

	// Sort suggestions to put prefix matches first

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
