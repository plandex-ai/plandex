package streamtui

import (
	"fmt"
	"log"
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
		color.New(color.FgHiRed, color.Bold).Printf("error in stream UI: %v\n", mod.err)
		return fmt.Errorf("error in stream UI: %v", mod.err)
	}

	if mod.apiErr != nil {
		fmt.Println()
		color.New(color.FgHiRed, color.Bold).Printf("error in stream UI: %s\n", mod.apiErr.Msg)
		return fmt.Errorf("error in stream UI: %v", mod.apiErr)
	}

	if mod.stopped {
		fmt.Println()
		color.New(color.BgBlack, color.Bold, color.FgHiRed).Print(" 🛑 stopped early ")
		fmt.Println()
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
