package streamtui

import (
	"context"
	"fmt"
	"log"
	"os"
	"plandex-cli/api"
	"plandex-cli/lib"
	"plandex-cli/term"
	"strings"
	"time"

	shared "plandex-shared"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
)

func (m streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("Stream TUI - Update received message:", spew.Sdump(msg))

	switch msg := msg.(type) {

	case spinner.TickMsg:
		state := m.readState()

		if state.processing || state.starting {
			m.updateState(func() {
				spinnerModel, _ := m.spinner.Update(msg)
				m.spinner = spinnerModel
			})
		}
		if state.building {
			m.updateState(func() {
				buildSpinnerModel, _ := m.buildSpinner.Update(msg)
				m.buildSpinner = buildSpinnerModel
			})
		}
		return m, m.Tick()

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case shared.StreamMessage:
		return m.streamUpdate(&msg, false)

	case contextLoadDoneMsg:
		if msg.err != nil {
			log.Println("failed to auto load context files:", msg.err)
			m.updateState(func() {
				m.err = msg.err
				m.processing = false
			})
			return m, tea.Quit
		}

		// We have the loaded content in msg.text
		m.updateState(func() {
			if msg.text != "" {
				m.reply += "\n\n" + msg.text + "\n\n"
			}
			// and keep processing
			m.processing = true
		})
		m.updateReplyDisplay()
		return m, m.Tick()

	case delayFileRestartMsg:
		m.updateState(func() {
			m.finishedByPath[msg.path] = false
		})

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
		case bubbleKey.Matches(msg, m.keymap.stop) || bubbleKey.Matches(msg, m.keymap.quit):
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			apiErr := api.Client.StopPlan(ctx, lib.CurrentPlanId, lib.CurrentBranch)
			if apiErr != nil {
				log.Println("stop plan api error:", apiErr)
				m.updateState(func() {
					m.apiErr = apiErr
				})
			}
			m.updateState(func() {
				m.stopped = true
			})
			return m, tea.Quit

		case bubbleKey.Matches(msg, m.keymap.background):
			state := m.readState()
			if state.canSendToBg {
				m.updateState(func() {
					m.background = true
				})
				return m, tea.Quit
			}

		case bubbleKey.Matches(msg, m.keymap.scrollDown) && !m.promptingMissingFile:
			m.scrollDown()
		case bubbleKey.Matches(msg, m.keymap.scrollUp) && !m.promptingMissingFile:
			m.scrollUp()
		case bubbleKey.Matches(msg, m.keymap.pageDown) && !m.promptingMissingFile:
			m.pageDown()
		case bubbleKey.Matches(msg, m.keymap.pageUp) && !m.promptingMissingFile:
			m.pageUp()
		case bubbleKey.Matches(msg, m.keymap.up) && m.building:
			m.up()
		case bubbleKey.Matches(msg, m.keymap.down) && m.building:
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

	case buildStatusPollMsg:
		state := m.readState()

		numPaths := len(m.tokensByPath)
		numFinished := 0

		for _, isBuilt := range m.finishedByPath {
			if isBuilt {
				numFinished++
			}
		}

		if !state.finished && !state.stopped && !state.background && numPaths > 0 && numPaths != numFinished {
			status, apiErr := api.Client.GetBuildStatus(lib.CurrentPlanId, lib.CurrentBranch)
			if apiErr != nil {
				return m, m.pollBuildStatus()
			}

			m.updateState(func() {
				for path, isBuilt := range status.BuiltFiles {
					isBuilding := status.IsBuildingByPath[path]
					if isBuilt && !isBuilding {
						m.finishedByPath[path] = true
					}
				}
			})
		}
		return m, m.pollBuildStatus()
	}

	return m, nil
}

func (m *streamUIModel) windowResized(w, h int) {
	m.updateState(func() {
		m.width = w
		m.height = h
	})

	state := m.readState()

	_, viewportHeight := state.getViewportDimensions()

	if state.ready {
		m.updateViewportDimensions()
	} else {
		m.updateState(func() {
			m.mainViewport = viewport.New(w, viewportHeight)
			m.mainViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
		})

		m.updateReplyDisplay()

		m.updateState(func() {
			m.ready = true
		})
	}
}

func (m *streamUIModel) updateReplyDisplay() {
	state := m.readState()

	if state.buildOnly {
		return
	}

	s := ""

	if state.prompt != "" {
		promptTxt := term.GetPlain(state.prompt)

		s += color.New(color.BgGreen, color.Bold, color.FgHiWhite).Sprintf(" 💬 User prompt 👇 ")
		s += "\n\n" + strings.TrimSpace(promptTxt) + "\n"
	}

	if state.reply != "" {
		replyMd, _ := term.GetMarkdown(state.reply)
		s += "\n" + color.New(color.BgBlue, color.Bold, color.FgHiWhite).Sprintf(" 🤖 Plandex reply 👇 ")
		s += "\n\n" + strings.TrimSpace(replyMd)
	} else {
		s += "\n"
	}

	m.updateState(func() {
		m.mainDisplay = s
		m.mainViewport.SetContent(s)
	})

	m.updateViewportDimensions()

	if state.atScrollBottom {
		m.updateState(func() {
			m.mainViewport.GotoBottom()
		})
	}
}

func (m *streamUIModel) updateViewportDimensions() {
	state := m.readState()
	w, h := state.getViewportDimensions()

	m.updateState(func() {
		m.mainViewport.Width = w
		m.mainViewport.Height = h
	})
}

func (m *streamUIModel) getViewportDimensions() (int, int) {
	w := m.width
	h := m.height

	helpHeight := lipgloss.Height(m.renderHelp())

	var buildHeight int
	if m.building {
		if m.buildViewCollapsed {
			buildHeight = 3
		} else {
			buildHeight = len(m.getRows(false))
		}
	}

	var processingHeight int
	if m.starting || m.processing {
		processingHeight = lipgloss.Height(m.renderProcessing())
	}

	maxViewportHeight := h - (helpHeight + processingHeight + buildHeight)
	if maxViewportHeight < 0 {
		maxViewportHeight = 0
	}
	viewportHeight := min(maxViewportHeight, lipgloss.Height(m.mainDisplay))
	viewportWidth := w

	return viewportWidth, viewportHeight
}

func (m streamUIModel) replyScrollable() bool {
	return m.mainViewport.TotalLineCount() > m.mainViewport.VisibleLineCount()
}

func (m *streamUIModel) scrollDown() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.LineDown(1)
		})
	}

	state = m.readState()

	m.updateState(func() {
		m.atScrollBottom = !state.replyScrollable() || state.mainViewport.AtBottom()
	})
}

func (m *streamUIModel) scrollUp() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.LineUp(1)
			m.atScrollBottom = false
		})
	}
}

func (m *streamUIModel) pageDown() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.ViewDown()
		})
	}

	state = m.readState()

	m.updateState(func() {
		m.atScrollBottom = !state.replyScrollable() || state.mainViewport.AtBottom()
	})
}

func (m *streamUIModel) pageUp() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.ViewUp()
			m.atScrollBottom = false
		})
	}
}

func (m *streamUIModel) scrollStart() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.GotoTop()
			m.atScrollBottom = false
		})
	}
}

func (m *streamUIModel) scrollEnd() {
	state := m.readState()

	if state.replyScrollable() {
		m.updateState(func() {
			m.mainViewport.GotoBottom()
			m.atScrollBottom = true
		})
	}
}

func (m *streamUIModel) streamUpdate(msg *shared.StreamMessage, deferUIUpdate bool) (tea.Model, tea.Cmd) {
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
			m.updateState(func() {
				m.prompt = msg.InitPrompt
			})
		}
		if msg.InitBuildOnly {
			m.updateState(func() {
				m.buildOnly = true
			})
		}
		if len(msg.InitReplies) > 0 {
			m.updateState(func() {
				m.reply = strings.Join(msg.InitReplies, "\n\n👇\n\n")
			})
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

		state := m.readState()

		if state.starting {
			m.updateState(func() {
				m.starting = false
			})
		}

		if state.processing {
			log.Println("Non-empty message reply, setting processing to false")
			m.updateState(func() {
				m.processing = false
				if state.promptedMissingFile || state.autoLoadedMissingFile {
					log.Println("Prompted missing file or auto loaded missing file, resetting (and skipping 👇 marker)")
					m.promptedMissingFile = false
					m.autoLoadedMissingFile = false
				} else {
					log.Println("Not prompted missing file or auto loaded missing file, adding 👇 marker")
					m.reply += "\n\n👇\n\n"
				}
			})
		}

		m.updateState(func() {
			m.reply += msg.ReplyChunk
		})

		if !deferUIUpdate {
			m.updateReplyDisplay()
		}

	case shared.StreamMessageBuildInfo:
		state := m.readState()

		if state.starting {
			m.updateState(func() {
				m.starting = false
			})
		}

		m.updateState(func() {
			m.building = true
		})
		wasFinished := state.finishedByPath[msg.BuildInfo.Path]
		nowFinished := msg.BuildInfo.Finished

		m.updateState(func() {
			if msg.BuildInfo.Removed {
				m.removedByPath[msg.BuildInfo.Path] = true
			} else {
				m.removedByPath[msg.BuildInfo.Path] = false
			}
		})

		if msg.BuildInfo.Finished {
			m.updateState(func() {
				m.tokensByPath[msg.BuildInfo.Path] = 0
				m.finishedByPath[msg.BuildInfo.Path] = true
			})
		} else {
			if wasFinished && !nowFinished {
				// delay for a second before marking not finished again (so check flashes green prior to restarting build)
				log.Println("Stream message build info - delaying for 1 second before marking not finished again")
				return m, startDelay(msg.BuildInfo.Path, time.Second*1)
			} else {
				m.updateState(func() {
					m.finishedByPath[msg.BuildInfo.Path] = false
				})
			}

			m.updateState(func() {
				m.tokensByPath[msg.BuildInfo.Path] += msg.BuildInfo.NumTokens
			})
		}

		// Auto-collapse if build info takes up too much space
		state = m.readState()
		if !state.userToggledBuild && state.building {
			rows := len(m.getRows(false))
			m.updateState(func() {
				m.buildViewCollapsed = rows > 3
			})
		}

		if !deferUIUpdate {
			m.updateViewportDimensions()
		}

		return m, m.Tick()

	case shared.StreamMessageDescribing:
		log.Println("Message describing, setting processing to true")
		m.updateState(func() {
			m.processing = true
		})
		return m, m.Tick()

		// Instead of blocking here, we'll spawn a command
	case shared.StreamMessageLoadContext:
		m.updateState(func() {
			m.processing = true
		})
		return m, tea.Batch(
			loadContextCmd(msg.LoadContextFiles),
			tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
				return spinner.TickMsg{}
			}),
		)

	case shared.StreamMessageError:
		log.Println("Stream message error:", spew.Sdump(msg))

		state := m.readState()
		if state.autoLoadContextCancelFn != nil {
			state.autoLoadContextCancelFn()
		}

		m.updateState(func() {
			m.apiErr = msg.Error
		})
		return m, tea.Quit

	case shared.StreamMessageFinished:
		m.updateState(func() {
			m.finished = true
		})
		return m, tea.Quit

	case shared.StreamMessageAborted:
		m.updateState(func() {
			m.stopped = true
		})
		return m, tea.Quit

	case shared.StreamMessageRepliesFinished:
		log.Println("Replies finished, setting processing to false")
		state := m.readState()

		m.updateState(func() {
			m.processing = false
		})

		if state.building {
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
				m.up()
				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[B" || escSeq == "alt+[B" {
				m.down()
				escReceivedAt = time.Time{}
				escSeq = ""
			}
		}
	}
}

func (m *streamUIModel) up() {
	state := m.readState()
	if state.promptingMissingFile {
		m.updateState(func() {
			m.missingFileSelectedIdx = max(m.missingFileSelectedIdx-1, 0)
		})
	} else {
		m.updateState(func() {
			m.buildViewCollapsed = false
			m.userToggledBuild = true
		})
	}
}

func (m *streamUIModel) down() {
	state := m.readState()
	if state.promptingMissingFile {
		m.updateState(func() {
			m.missingFileSelectedIdx = min(m.missingFileSelectedIdx+1, len(missingFileSelectOpts)-1)
		})
	} else {
		m.updateState(func() {
			m.buildViewCollapsed = true
			m.userToggledBuild = true
		})
	}
}

func (m *streamUIModel) selectedMissingFileOpt() (tea.Model, tea.Cmd) {
	state := m.readState()
	choice := promptChoices[state.missingFileSelectedIdx]

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
		m.updateState(func() {
			m.apiErr = apiErr
		})
		return m, nil
	}

	if choice == shared.RespondMissingFileChoiceSkip {
		replyLines := strings.Split(state.reply, "\n")
		m.updateState(func() {
			m.reply = strings.Join(replyLines[:len(replyLines)-3], "\n")
		})
		m.updateReplyDisplay()
	}

	m.updateState(func() {
		m.promptingMissingFile = false
		m.missingFilePath = ""
		m.missingFileSelectedIdx = 0
		m.missingFileContent = ""
		m.missingFileTokens = 0
		m.promptedMissingFile = true
		m.processing = true
	})

	return m, func() tea.Msg {
		<-m.sharedTicker.C
		return spinner.TickMsg{}
	}
}

func (m *streamUIModel) checkMissingFile(msg *shared.StreamMessage) (tea.Model, tea.Cmd) {
	if msg.MissingFilePath != "" {
		log.Println("checkMissingFile - received missing file message | path:", msg.MissingFilePath)

		if msg.MissingFileAutoContext {
			log.Println("checkMissingFile - received missing file message | auto context")
			m.updateState(func() {
				m.processing = true
				m.autoLoadedMissingFile = true
			})

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
					content := string(shared.NormalizeEOL(bytes))

					log.Println("checkMissingFile - calling RespondMissingFile")
					apiErr := api.Client.RespondMissingFile(lib.CurrentPlanId, lib.CurrentBranch, shared.RespondMissingFileRequest{
						Choice:   shared.RespondMissingFileChoiceLoad,
						FilePath: msg.MissingFilePath,
						Body:     content,
					})

					if apiErr != nil {
						log.Println("missing file prompt api error:", apiErr)
						m.updateState(func() {
							m.apiErr = apiErr
						})
						return tea.Quit
					}

					log.Println("checkMissingFile - RespondMissingFile success")

					return nil
				},
			)
		}

		m.updateState(func() {
			m.promptingMissingFile = true
			m.missingFilePath = msg.MissingFilePath
		})

		bytes, err := os.ReadFile(m.missingFilePath)
		if err != nil {
			log.Println("failed to read file:", err)
			m.updateState(func() {
				m.err = fmt.Errorf("failed to read file: %w", err)
			})
			return m, nil
		}

		missingFileContent := string(bytes)
		m.updateState(func() {
			m.missingFileContent = missingFileContent
		})

		numTokens := shared.GetNumTokensEstimate(missingFileContent)
		m.updateState(func() {
			m.missingFileTokens = numTokens
		})
	}

	return m, nil
}

// contextLoadDoneMsg is sent when the long-running AutoLoadContextFiles completes
type contextLoadDoneMsg struct {
	text string
	err  error
}

func loadContextCmd(loadContextFiles []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Run the long operation directly
		text, err := lib.AutoLoadContextFiles(ctx, loadContextFiles)

		// Return the result as a message
		return contextLoadDoneMsg{
			text: text,
			err:  err,
		}
	}
}
