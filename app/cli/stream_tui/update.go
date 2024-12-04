package streamtui

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/lib"
	"plandex/term"
	"strings"
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func (m streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("Update received message:", spew.Sdump(msg))

	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.processing || m.starting {
			spinnerModel, _ := m.spinner.Update(msg)
			m.spinner = spinnerModel
		}
		if m.building {
			buildSpinnerModel, _ := m.buildSpinner.Update(msg)
			m.buildSpinner = buildSpinnerModel
		}
		return m, m.Tick()

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case shared.StreamMessage:
		return m.streamUpdate(&msg, false)

	case delayFileRestartMsg:
		m.finishedByPath[msg.path] = false

	// Scroll wheel doesn't seem to work--not sure why
	// case tea.MouseMsg:
	// 	if !m.promptingMissingFile {
	// 		if msg.Type == tea.MouseWheelUp {
	// 			m.mainViewport.LineUp(3)
	// 		} else if msg.Type == tea.MouseWheelDown {
	// 			m.mainViewport.LineDown(3)
	// 		}
	// 	}

	case tea.KeyMsg:
		switch {

		// more intuitive for ctrl+c to stop than send to background
		// case bubbleKey.Matches(msg, m.keymap.quit):
		// 	m.background = true
		// 	return &m, tea.Quit

		case bubbleKey.Matches(msg, m.keymap.stop) || bubbleKey.Matches(msg, m.keymap.quit):
			apiErr := api.Client.StopPlan(lib.CurrentPlanId, lib.CurrentBranch)
			if apiErr != nil {
				log.Println("stop plan api error:", apiErr)
				m.apiErr = apiErr
			}
			m.stopped = true
			return m, tea.Quit

		case bubbleKey.Matches(msg, m.keymap.scrollDown) && !m.promptingMissingFile:
			m.scrollDown()
		case bubbleKey.Matches(msg, m.keymap.scrollUp) && !m.promptingMissingFile:
			m.scrollUp()
		case bubbleKey.Matches(msg, m.keymap.pageDown) && !m.promptingMissingFile:
			m.pageDown()
		case bubbleKey.Matches(msg, m.keymap.pageUp) && !m.promptingMissingFile:
			m.pageUp()
		case bubbleKey.Matches(msg, m.keymap.up):
			m.up()
		case bubbleKey.Matches(msg, m.keymap.down):
			m.down()
		case bubbleKey.Matches(msg, m.keymap.start) && !m.promptingMissingFile:
			m.scrollStart()
		case bubbleKey.Matches(msg, m.keymap.end) && !m.promptingMissingFile:
			m.scrollEnd()
		case m.promptingMissingFile && bubbleKey.Matches(msg, m.keymap.enter):
			return m.selectedMissingFileOpt()

		default:
			m.resolveEscapeSequence(msg.String())
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
		m.mainViewport = viewport.New(w, viewportHeight)
		m.mainViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
		m.updateReplyDisplay()
		m.ready = true
	}
}

func (m *streamUIModel) updateReplyDisplay() {
	if m.buildOnly {
		return
	}

	s := ""

	if m.prompt != "" {
		promptTxt := term.GetPlain(m.prompt)

		s += color.New(color.BgGreen, color.Bold, color.FgHiWhite).Sprintf(" 💬 User prompt 👇 ")
		s += "\n\n" + strings.TrimSpace(promptTxt) + "\n"
	}

	if m.reply != "" {
		replyMd, _ := term.GetMarkdown(m.reply)
		s += "\n" + color.New(color.BgBlue, color.Bold, color.FgHiWhite).Sprintf(" 🤖 Plandex reply 👇 ")
		s += "\n\n" + strings.TrimSpace(replyMd)
	} else {
		s += "\n"
	}

	m.mainDisplay = s
	m.mainViewport.SetContent(s)
	m.updateViewportDimensions()

	if m.atScrollBottom {
		m.mainViewport.GotoBottom()
	}
}

func (m *streamUIModel) updateViewportDimensions() {
	// log.Println("updateViewportDimensions")

	w, h := m.getViewportDimensions()
	m.mainViewport.Width = w
	m.mainViewport.Height = h
}

func (m *streamUIModel) getViewportDimensions() (int, int) {
	w := m.width
	h := m.height

	helpHeight := lipgloss.Height(m.renderHelp())

	var buildHeight int
	if m.building {
		buildHeight = len(m.getRows(false))
	}

	// log.Println("building:", m.building)
	// log.Println("buildHeight:", buildHeight)

	var processingHeight int
	if m.starting || m.processing {
		processingHeight = lipgloss.Height(m.renderProcessing())
	}

	maxViewportHeight := h - (helpHeight + processingHeight + buildHeight)
	viewportHeight := min(maxViewportHeight, lipgloss.Height(m.mainDisplay))
	viewportWidth := w

	// log.Println("viewportWidth:", viewportWidth)

	return viewportWidth, viewportHeight
}

func (m streamUIModel) replyScrollable() bool {
	return m.mainViewport.TotalLineCount() > m.mainViewport.VisibleLineCount()
}

func (m *streamUIModel) scrollDown() {
	if m.replyScrollable() {
		m.mainViewport.LineDown(1)
	}

	m.atScrollBottom = !m.replyScrollable() || m.mainViewport.AtBottom()
}

func (m *streamUIModel) scrollUp() {
	if m.replyScrollable() {
		m.mainViewport.LineUp(1)
		m.atScrollBottom = false
	}
}

func (m *streamUIModel) pageDown() {
	if m.replyScrollable() {
		m.mainViewport.ViewDown()
	}

	m.atScrollBottom = !m.replyScrollable() || m.mainViewport.AtBottom()
}

func (m *streamUIModel) pageUp() {
	if m.replyScrollable() {
		m.mainViewport.ViewUp()
		m.atScrollBottom = false
	}
}

func (m *streamUIModel) scrollStart() {
	if m.replyScrollable() {
		m.mainViewport.GotoTop()
		m.atScrollBottom = false
	}
}

func (m *streamUIModel) scrollEnd() {
	if m.replyScrollable() {
		m.mainViewport.GotoBottom()
		m.atScrollBottom = true
	}
}

func (m *streamUIModel) streamUpdate(msg *shared.StreamMessage, deferUIUpdate bool) (tea.Model, tea.Cmd) {

	// log.Println("streamUI received message:", msg.Type)
	// log.Println(spew.Sdump(msg))

	switch msg.Type {

	case shared.StreamMessageMulti:
		cmds := []tea.Cmd{}
		for _, subMsg := range msg.StreamMessages {
			teaModel, cmd := m.streamUpdate(&subMsg, true)

			m = teaModel.(*streamUIModel)

			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		m.updateReplyDisplay()
		m.updateViewportDimensions()

		return m, tea.Batch(cmds...)

	case shared.StreamMessageConnectActive:

		if msg.InitPrompt != "" {
			m.prompt = msg.InitPrompt
		}
		if msg.InitBuildOnly {
			m.buildOnly = true
		}
		if len(msg.InitReplies) > 0 {
			m.reply = strings.Join(msg.InitReplies, "\n\n👇\n\n")
		}
		m.updateReplyDisplay()

		return m.checkMissingFile(msg)

	case shared.StreamMessagePromptMissingFile:
		return m.checkMissingFile(msg)

	case shared.StreamMessageReply:
		// ignore empty reply messages
		if msg.ReplyChunk == "" {
			return m, nil
		}

		if m.starting {
			m.starting = false
		}

		if m.processing {
			log.Println("Non-empty message reply, setting processing to false")
			m.processing = false
			if m.promptedMissingFile || m.autoLoadedMissingFile {
				log.Println("Prompted missing file or auto loaded missing file, resetting (and skipping 👇 marker)")
				m.promptedMissingFile = false
				m.autoLoadedMissingFile = false
			} else {
				log.Println("Not prompted missing file or auto loaded missing file, adding 👇 marker")
				m.reply += "\n\n👇\n\n"
			}
		}

		m.reply += msg.ReplyChunk

		if !deferUIUpdate {
			m.updateReplyDisplay()
		}

	case shared.StreamMessageBuildInfo:
		if m.starting {
			m.starting = false
		}

		m.building = true
		wasFinished := m.finishedByPath[msg.BuildInfo.Path]
		nowFinished := msg.BuildInfo.Finished

		if msg.BuildInfo.Finished {
			m.tokensByPath[msg.BuildInfo.Path] = 0
			m.finishedByPath[msg.BuildInfo.Path] = true
		} else {
			if wasFinished && !nowFinished {
				// delay for a second before marking not finished again (so check flashes green prior to restarting build)
				return m, startDelay(msg.BuildInfo.Path, time.Second*1)
			} else {
				m.finishedByPath[msg.BuildInfo.Path] = false
			}

			m.tokensByPath[msg.BuildInfo.Path] += msg.BuildInfo.NumTokens
		}

		if !deferUIUpdate {
			m.updateViewportDimensions()
		}

		return m, m.Tick()

	case shared.StreamMessageDescribing:
		log.Println("Message describing, setting processing to true")
		m.processing = true

		return m, m.Tick()

	case shared.StreamMessageLoadContext:
		log.Println("Stream message auto-load context")
		msg, err := lib.AutoLoadContextFiles(msg.LoadContextFiles)
		if err != nil {
			log.Println("failed to auto load context files:", err)
			m.err = err
			return m, tea.Quit
		}

		m.reply += "\n\n" + msg + "\n\n"
		m.updateReplyDisplay()

		return m, m.Tick()

	case shared.StreamMessageError:
		m.apiErr = msg.Error
		return m, tea.Quit

	case shared.StreamMessageFinished:
		// log.Println("stream finished")
		m.finished = true
		return m, tea.Quit

	case shared.StreamMessageAborted:
		m.stopped = true
		return m, tea.Quit

	case shared.StreamMessageRepliesFinished:
		log.Println("Replies finished, setting processing to false")
		m.processing = false

		if m.building {
			return m, m.Tick()
		}
	}

	return m, nil
}

type delayFileRestartMsg struct {
	path string
}

func startDelay(path string, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return delayFileRestartMsg{path: path}
	}
}

var escReceivedAt time.Time
var escSeq string

func (m *streamUIModel) resolveEscapeSequence(val string) {
	if val == "esc" || val == "alt+[" {
		escReceivedAt = time.Now()
		go func() {
			time.Sleep(51 * time.Millisecond)
			escReceivedAt = time.Time{}
			escSeq = ""
		}()
	}

	if !escReceivedAt.IsZero() {
		elapsed := time.Since(escReceivedAt)

		if elapsed < 50*time.Millisecond {
			escSeq += val

			if escSeq == "esc[A" || escSeq == "alt+[A" {
				// log.Println("up")
				m.up()

				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[B" || escSeq == "alt+[B" {
				// log.Println("down")
				m.down()

				escReceivedAt = time.Time{}
				escSeq = ""
			}
		}
	}
}

func (m *streamUIModel) up() {
	if m.promptingMissingFile {
		m.missingFileSelectedIdx = max(m.missingFileSelectedIdx-1, 0)
	}
}

func (m *streamUIModel) down() {
	if m.promptingMissingFile {
		m.missingFileSelectedIdx = min(m.missingFileSelectedIdx+1, len(missingFileSelectOpts)-1)
	}

}

func (m *streamUIModel) selectedMissingFileOpt() (tea.Model, tea.Cmd) {
	choice := promptChoices[m.missingFileSelectedIdx]

	if choice == "" {
		return m, nil
	}

	apiErr := api.Client.RespondMissingFile(lib.CurrentPlanId, lib.CurrentBranch, shared.RespondMissingFileRequest{
		Choice:   choice,
		FilePath: m.missingFilePath,
		Body:     m.missingFileContent,
	})

	if apiErr != nil {
		log.Println("missing file prompt api error:", apiErr)
		m.apiErr = apiErr
		return m, nil
	}

	if choice == shared.RespondMissingFileChoiceSkip {
		replyLines := strings.Split(m.reply, "\n")
		m.reply = strings.Join(replyLines[:len(replyLines)-3], "\n")
		m.updateReplyDisplay()
	}

	m.promptingMissingFile = false
	m.missingFilePath = ""
	m.missingFileSelectedIdx = 0
	m.missingFileContent = ""
	m.missingFileTokens = 0
	m.promptedMissingFile = true
	m.processing = true

	return m, func() tea.Msg {
		<-m.sharedTicker.C
		return spinner.TickMsg{}
	}
}

func (m *streamUIModel) checkMissingFile(msg *shared.StreamMessage) (tea.Model, tea.Cmd) {
	if msg.MissingFilePath != "" {
		if msg.MissingFileAutoContext {
			m.processing = true
			m.autoLoadedMissingFile = true

			return m, tea.Batch(
				func() tea.Msg {
					<-m.sharedTicker.C
					return spinner.TickMsg{}
				},
				func() tea.Msg {
					bytes, err := os.ReadFile(msg.MissingFilePath)
					if err != nil {
						log.Println("failed to read file:", err)
						m.err = fmt.Errorf("failed to read file: %w", err)
						return tea.Quit
					}
					content := string(bytes)

					apiErr := api.Client.RespondMissingFile(lib.CurrentPlanId, lib.CurrentBranch, shared.RespondMissingFileRequest{
						Choice:   shared.RespondMissingFileChoiceLoad,
						FilePath: msg.MissingFilePath,
						Body:     content,
					})

					if apiErr != nil {
						log.Println("missing file prompt api error:", apiErr)
						m.apiErr = apiErr
						return tea.Quit
					}

					return nil
				},
			)
		}

		m.promptingMissingFile = true
		m.missingFilePath = msg.MissingFilePath

		bytes, err := os.ReadFile(m.missingFilePath)
		if err != nil {
			log.Println("failed to read file:", err)
			m.err = fmt.Errorf("failed to read file: %w", err)
			return m, nil
		}
		m.missingFileContent = string(bytes)

		numTokens, err := shared.GetNumTokens(m.missingFileContent)

		if err != nil {
			log.Println("failed to get num tokens:", err)
			m.err = fmt.Errorf("failed to get num tokens: %w", err)
			return m, nil
		}

		m.missingFileTokens = numTokens
	}

	return m, nil
}
