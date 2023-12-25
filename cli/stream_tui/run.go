package streamtui

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

var ui *tea.Program

func StartStreamUI() error {
	initial := initialModel()

	ui = tea.NewProgram(initial, tea.WithAltScreen())

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
	ui.Quit()
}

func Send(msg tea.Msg) {
	if ui == nil {
		log.Println("stream ui is nil, can't send message")
		return
	}

	ui.Send(msg)
}
