package streamtui

import (
	"log"
	"plandex/term"
	"plandex/types"
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const displayThrottle = 150 * time.Millisecond

func (m streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.processing {
			spinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = spinnerModel
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case types.StreamTUIUpdate:
		now := time.Now()

		if msg.PlanTokenCount != nil {
			m.processing = false
			m.building = true

			if m.replyDisplay != m.reply {
				m.updateReplyDisplay(now)
			}

			m.tokensByPath[msg.PlanTokenCount.Path] += msg.PlanTokenCount.NumTokens
			if msg.PlanTokenCount.Finished {
				m.finishedByPath[msg.PlanTokenCount.Path] = true
			}
		} else if msg.ReplyChunk != "" {
			m.processing = false

			elapsed := now.Sub(m.replyDisplayUpdatedAt)

			m.reply += msg.ReplyChunk

			if elapsed >= displayThrottle {
				m.updateReplyDisplay(now)
			}

		} else if msg.Processing {
			m.processing = true
			if m.replyDisplay != m.reply {
				m.updateReplyDisplay(now)
			}
			return m, m.spinner.Tick
		}

	case tea.KeyMsg:
		switch {
		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *streamUIModel) windowResized(w, h int) {
	m.width = w
	m.height = h
	if m.ready {
		m.replyViewport.Width = w
		m.replyViewport.Height = h
	} else {
		m.replyViewport = viewport.New(w, h)
		m.replyViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
		m.ready = true
	}
}

func (m *streamUIModel) updateReplyDisplay(now time.Time) {
	m.replyDisplay = m.reply
	md, err := term.GetMarkdown(m.reply)
	if err != nil {
		log.Println("error parsing markdown:", err)
		return
	}

	m.replyViewport.SetContent(md)
	m.replyDisplayUpdatedAt = now
}
