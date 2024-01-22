package streamtui

import (
	"fmt"
	"log"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

var ui *tea.Program
var mu sync.Mutex

func StartStreamUI() error {
	initial := initialModel()

	mu.Lock()
	ui = tea.NewProgram(initial, tea.WithAltScreen())
	mu.Unlock()

	_, err := ui.Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
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
}

func Send(msg tea.Msg) {
	if ui == nil {
		log.Println("stream ui is nil, can't send message")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Send(msg)
}
