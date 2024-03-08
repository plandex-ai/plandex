package streamtui

import (
	"fmt"
	"log"
	"os"
	"plandex/term"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

var ui *tea.Program
var mu sync.Mutex
var wg sync.WaitGroup

var prestartReply string

func StartStreamUI(prompt string, buildOnly bool) error {
	initial := initialModel(prestartReply, prompt, buildOnly)

	mu.Lock()
	ui = tea.NewProgram(initial, tea.WithAltScreen())
	mu.Unlock()

	wg.Add(1)
	m, err := ui.Run()
	wg.Done()

	if err != nil {
		return fmt.Errorf("error running stream UI: %v", err)
	}

	mod := m.(*streamUIModel)

	fmt.Println()

	if !mod.buildOnly {
		fmt.Println(mod.mainDisplay)
	}

	if len(mod.finishedByPath) > 0 || len(mod.tokensByPath) > 0 {
		fmt.Println(mod.renderStaticBuild())
	}

	if mod.err != nil {
		fmt.Println()
		term.OutputErrorAndExit(mod.err.Error())
	}

	if mod.apiErr != nil {
		fmt.Println()
		term.OutputErrorAndExit(mod.apiErr.Msg)
	}

	if mod.stopped {
		fmt.Println()
		color.New(color.BgBlack, color.Bold, color.FgHiRed).Println(" 🛑 Stopped early ")
		fmt.Println()
		term.PrintCmds("", "log", "rewind", "tell")
		os.Exit(0)
	} else if mod.background {
		fmt.Println()
		color.New(color.BgBlack, color.Bold, color.FgHiGreen).Println(" ✅ Plan is active in the background ")
		fmt.Println()
		term.PrintCmds("", "ps", "connect", "stop")
		os.Exit(0)
	}

	return nil
}

func Quit() {
	if ui == nil {
		log.Println("stream UI is nil, can't quit")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Quit()

	wg.Wait() // Wait for the UI to fully terminate

}

func Send(msg shared.StreamMessage) {
	if ui == nil {
		// log.Println("stream ui is nil")
		prestartReply += msg.ReplyChunk
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Send(msg)
}
