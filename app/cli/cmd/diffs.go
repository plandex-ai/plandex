package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"plandex-cli/ui"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// var plainTextOutput bool
var showDiffUi bool = true
var diffUiSideBySide = true
var diffUiLineByLine bool
var diffUiGit bool

var diffsCmd = &cobra.Command{
	Use:     "diff",
	Aliases: []string{"diffs"},
	Short:   "Review pending changes",
	Run:     diffs,
}

func init() {
	RootCmd.AddCommand(diffsCmd)

	diffsCmd.Flags().BoolVarP(&plainTextOutput, "plain", "p", false, "Output diffs in plain text with no ANSI codes")
	diffsCmd.Flags().BoolVar(&showDiffUi, "ui", false, "Show diffs in a browser UI")
	diffsCmd.Flags().BoolVar(&diffUiGit, "git", false, "Show diffs in git diff format")
	diffsCmd.Flags().BoolVarP(&diffUiSideBySide, "side", "s", true, "Show diffs UI in side-by-side view")
	diffsCmd.Flags().BoolVarP(&diffUiLineByLine, "line", "l", false, "Show diffs UI in line-by-line view")
}

func diffs(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MaybeResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput || showDiffUi)
	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	if diffUiGit {
		showDiffUi = false
	}

	if showDiffUi {
		getNewListener := func() net.Listener {
			outputFormat := "line-by-line"
			if diffUiSideBySide {
				outputFormat = "side-by-side"
			} else if diffUiLineByLine {
				outputFormat = "line-by-line"
			}

			// Properly escape the diff content for JavaScript
			diffJSON, err := json.Marshal(diffs)
			if err != nil {
				term.OutputErrorAndExit("Error encoding diff content: %v", err)
			}

			// Create template data
			data := struct {
				DiffContent  template.JS
				OutputFormat string
			}{
				DiffContent:  template.JS(diffJSON),
				OutputFormat: outputFormat,
			}

			// Parse and execute the template
			tmpl, err := template.New("diff").Parse(htmlTemplate)
			if err != nil {
				term.OutputErrorAndExit("Error parsing template: %v", err)
			}

			// Use :0 to let the OS pick an available port
			listener, err := net.Listen("tcp", ":0")
			if err != nil {
				term.OutputErrorAndExit("Error starting server: %v", err)
			}

			// Get the actual port chosen
			port := listener.Addr().(*net.TCPAddr).Port

			// Start web server
			go func() {
				http.HandleFunc("/"+outputFormat, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					err := tmpl.Execute(w, data)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				})
				http.Serve(listener, nil)
			}()

			ui.OpenURL("Showing "+outputFormat+" diffs in your default browser...", fmt.Sprintf("http://localhost:%d/%s", port, outputFormat))

			fmt.Println()

			return listener
		}

		listener := getNewListener()
		defer listener.Close()

		var relaunch bool

		for {
			if relaunch {
				listener.Close()
				listener = getNewListener()
				defer listener.Close()
				relaunch = false
			}

			if diffUiLineByLine {
				fmt.Printf("%s for side-by-side view\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("(s)"))
			} else {
				fmt.Printf("%s for line-by-line view\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("(l)"))
			}

			fmt.Printf("%s for git diff format\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("(g)"))
			fmt.Printf("%s to quit\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("(q)"))

			color.New(term.ColorHiMagenta, color.Bold).Print("Hotkey 👉 ")

			char, key, err := term.GetUserKeyInput()
			if err != nil {
				term.OutputErrorAndExit("Error getting key: %v", err)
				return
			}
			fmt.Println()

			if string(char) == "g" {
				_, err := lib.ExecPlandexCommandWithParams([]string{"diff"}, lib.ExecPlandexCommandParams{
					DisableSuggestions: true,
				})
				if err != nil {
					term.OutputErrorAndExit("Error showing git diff: %v", err)
				}
			} else if string(char) == "s" {
				diffUiSideBySide = true
				diffUiLineByLine = false
				relaunch = true
			} else if string(char) == "l" {
				diffUiSideBySide = false
				diffUiLineByLine = true
				relaunch = true
			} else if key == 13 || key == 10 || string(char) == "q" { // Check raw key codes for Enter/Return
				if term.IsRepl {
					fmt.Println()
					os.Exit(0)
				} else {
					break
				}
			} else if string(char) == "\x03" { // Ctrl+C
				os.Exit(0)
			} else {
				term.OutputErrorAndExit("Invalid command: %d | %s", key, string(char))
			}
		}
	} else {
		if plainTextOutput {
			fmt.Println(diffs)
		} else {
			term.PageOutput(diffs)
		}
		fmt.Println()
	}

	term.PrintCmds("", "apply", "reject")
}

var htmlTemplate = `<!doctype html>
<html lang="en-us">
  <head>
    <meta charset="utf-8" />
    <!-- Make sure to load the highlight.js CSS file before the Diff2Html CSS file -->
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/vs.min.css" />
    <link
      rel="stylesheet"
      type="text/css"
      href="https://cdn.jsdelivr.net/npm/diff2html/bundles/css/diff2html.min.css"
    />
    <script type="text/javascript" src="https://cdn.jsdelivr.net/npm/diff2html/bundles/js/diff2html-ui.min.js"></script>
  </head>
  <script>
    // Parse the JSON-encoded diff content
    const diffString = {{.DiffContent}};

    document.addEventListener('DOMContentLoaded', function () {
      var targetElement = document.getElementById('myDiffElement');
      var configuration = {
        outputFormat: '{{.OutputFormat}}'
      };
      var diff2htmlUi = new Diff2HtmlUI(targetElement, diffString, configuration);
      diff2htmlUi.draw();
      diff2htmlUi.highlightCode();
    });
  </script>
  <body>
    <div id="myDiffElement"></div>
  </body>
</html>`
