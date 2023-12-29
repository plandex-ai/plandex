package streamtui

import (
	"plandex/term"
	"plandex/types"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m.streamUpdate(&msg)

	case tea.KeyMsg:
		switch {
		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case bubbleKey.Matches(msg, m.keymap.scrollDown):
			m.scrollDown()
		case bubbleKey.Matches(msg, m.keymap.scrollUp):
			m.scrollUp()
		case bubbleKey.Matches(msg, m.keymap.pageDown):
			m.pageDown()
		case bubbleKey.Matches(msg, m.keymap.pageUp):
			m.pageUp()
		}
	}

	return m, nil
}

func (m *streamUIModel) windowResized(w, h int) {
	m.width = w
	m.height = h

	_, viewportHeight := m.getViewportDimensions()

	if m.ready {
		m.updateViewportDimensions()
	} else {
		m.replyViewport = viewport.New(w, viewportHeight)
		m.replyViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
		m.ready = true
	}
}

func (m *streamUIModel) updateReplyDisplay() {
	md, _ := term.GetMarkdown(m.reply)
	m.replyDisplay = md
	m.replyViewport.SetContent(md)
	m.updateViewportDimensions()
}

func (m *streamUIModel) updateViewportDimensions() {
	w, h := m.getViewportDimensions()
	m.replyViewport.Width = w
	m.replyViewport.Height = h
}

func (m *streamUIModel) getViewportDimensions() (int, int) {
	w := m.width
	h := m.height

	helpHeight := lipgloss.Height(m.renderHelp())
	buildHeight := lipgloss.Height(m.renderBuild())
	processingHeight := lipgloss.Height(m.renderProcessing())
	maxViewportHeight := h - (helpHeight + buildHeight + processingHeight)
	viewportHeight := min(maxViewportHeight, lipgloss.Height(m.replyDisplay))

	return w, viewportHeight
}

func (m streamUIModel) replyScrollable() bool {
	return m.replyViewport.TotalLineCount() > m.replyViewport.VisibleLineCount()
}

func (m *streamUIModel) scrollDown() {
	if m.replyScrollable() {
		m.replyViewport.LineDown(1)
	}
}

func (m *streamUIModel) scrollUp() {
	if m.replyScrollable() {
		m.replyViewport.LineUp(1)
	}
}

func (m *streamUIModel) pageDown() {
	if m.replyScrollable() {
		m.replyViewport.ViewDown()
	}
}

func (m *streamUIModel) pageUp() {
	if m.replyScrollable() {
		m.replyViewport.ViewUp()
	}
}

func (m *streamUIModel) streamUpdate(msg *types.StreamTUIUpdate) (tea.Model, tea.Cmd) {
	if msg.PlanTokenCount != nil {
		m.processing = false
		m.building = true

		m.tokensByPath[msg.PlanTokenCount.Path] += msg.PlanTokenCount.NumTokens
		if msg.PlanTokenCount.Finished {
			m.finishedByPath[msg.PlanTokenCount.Path] = true
		}
	} else if msg.ReplyChunk != "" {
		m.processing = false
		m.reply += msg.ReplyChunk
		m.updateReplyDisplay()

	} else if msg.Processing {
		m.processing = true
		return m, m.spinner.Tick
	}

	return m, nil
}
