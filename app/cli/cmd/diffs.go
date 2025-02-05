package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"plandex/ui"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// var plainTextOutput bool
var showDiffUi bool
var diffUiSideBySide = true
var diffUiLineByLine bool

var diffsCmd = &cobra.Command{
	Use:     "diff",
	Aliases: []string{"diffs"},
	Short:   "Show diffs for the pending changes in git diff format",
	Run:     diffs,
}

func init() {
	RootCmd.AddCommand(diffsCmd)

	diffsCmd.Flags().BoolVarP(&plainTextOutput, "plain", "p", false, "Output diffs in plain text with no ANSI codes")
	diffsCmd.Flags().BoolVar(&showDiffUi, "ui", false, "Show diffs in a browser UI")
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

	if showDiffUi {
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
			return
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
			return
		}

		// Use :0 to let the OS pick an available port
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			term.OutputErrorAndExit("Error starting server: %v", err)
			return
		}
		defer listener.Close()

		// Get the actual port chosen
		port := listener.Addr().(*net.TCPAddr).Port

		// Start web server
		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				err := tmpl.Execute(w, data)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			})
			http.Serve(listener, nil)
		}()

		ui.OpenURL("Showing diffs in your default browser...", fmt.Sprintf("http://localhost:%d", port))

		fmt.Println()
		color.New(color.Bold).Println("Press enter to continue")

		// Wait for enter key
		fmt.Scanln()
	} else {
		if plainTextOutput {
			fmt.Println(diffs)
		} else {
			term.PageOutput(diffs)
		}
	}
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
