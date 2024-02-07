package streamtui

import (
	"fmt"
	"log"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

var ui *tea.Program
var mu sync.Mutex
var wg sync.WaitGroup

func StartStreamUI() error {
	initial := initialModel()

	mu.Lock()
	ui = tea.NewProgram(initial, tea.WithAltScreen())
	mu.Unlock()

	wg.Add(1)
	m, err := ui.Run()
	wg.Done()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	if m.(streamUIModel).err != nil {
		return fmt.Errorf("error in changes UI: %v", m.(streamUIModel).err)
	}

	return nil
}

func Quit() {
	if ui == nil {
		log.Println("stream ui is nil, can't quit")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Quit()

	wg.Wait() // Wait for the UI to fully terminate

}

func Send(msg tea.Msg) {
	if ui == nil {
		log.Println("stream ui is nil")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Send(msg)
}
