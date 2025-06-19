package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/EkeMinusYou/gelf/internal/ai"
	"github.com/EkeMinusYou/gelf/internal/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLoading state = iota
	stateConfirm
	stateEditing
	stateCommitting
	stateSuccess
	stateError
)

type model struct {
	aiClient        *ai.VertexAIClient
	diff            string
	commitMessage   string
	originalMessage string
	err             error
	state           state
	spinner         spinner.Model
	textInput       textinput.Model
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}





var (
	// エレガントなスタイル
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))
	
	messageStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Margin(0).
		Italic(true)

	commitMessageHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true).
		Margin(1, 0, 0, 0)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")).
		Padding(0, 1).
		Margin(1, 0)

	// 生成中の洗練されたスタイル
	generatingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)


	loadingFrameStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Margin(1, 0)

	confirmFrameStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Padding(1, 2).
		Margin(1, 0)

	editFrameStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		Margin(1, 0)

	editPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Bold(true).
		Margin(1, 0, 0, 0)
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	
	ti := textinput.New()
	ti.Placeholder = "Enter your commit message..."
	ti.CharLimit = 200
	ti.Width = 60
	
	return &model{
		aiClient:  aiClient,
		diff:     diff,
		state:    stateLoading,
		spinner:  s,
		textInput: ti,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateCommitMessage())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateLoading:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = stateCommitting
				return m, tea.Batch(m.spinner.Tick, m.commitChanges())
			case "e", "E":
				m.originalMessage = m.commitMessage
				m.textInput.SetValue(m.commitMessage)
				m.textInput.Focus()
				m.state = stateEditing
				return m, textinput.Blink
			case "n", "N", "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateEditing:
			switch msg.String() {
			case "enter":
				m.commitMessage = strings.TrimSpace(m.textInput.Value())
				if m.commitMessage == "" {
					m.commitMessage = m.originalMessage
				}
				m.textInput.Blur()
				m.state = stateConfirm
			case "esc":
				m.commitMessage = m.originalMessage
				m.textInput.Blur()
				m.state = stateConfirm
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		case stateSuccess, stateError:
			return m, tea.Quit
		}


	case msgCommitGenerated:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
		} else {
			m.commitMessage = msg.message
			m.state = stateConfirm
		}

	case msgCommitDone:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
		} else {
			m.state = stateSuccess
		}
		return m, tea.Quit
	}

	// Update spinner
	if m.state == stateLoading || m.state == stateCommitting {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) View() string {
	switch m.state {
	case stateLoading:
		content := fmt.Sprintf("%s %s",
			m.spinner.View(),
			generatingStyle.Render("Generating commit message..."))
		return loadingFrameStyle.Render(content)

	case stateConfirm:
		header := commitMessageHeaderStyle.Render("📝 Generated Commit Message:")
		message := messageStyle.Render(m.commitMessage)
		prompt := promptStyle.Render("Commit this message? (y)es / (e)dit / (n)o")
		
		content := fmt.Sprintf("%s\n\n%s\n%s", header, message, prompt)
		return confirmFrameStyle.Render(content)

	case stateEditing:
		header := commitMessageHeaderStyle.Render("✏️  Edit Commit Message:")
		inputView := m.textInput.View()
		prompt := editPromptStyle.Render("Press Enter to confirm, Esc to cancel")
		
		content := fmt.Sprintf("%s\n\n%s\n%s", header, inputView, prompt)
		return editFrameStyle.Render(content)

	case stateCommitting:
		content := fmt.Sprintf("%s %s",
			m.spinner.View(),
			generatingStyle.Render("Committing changes..."))
		return loadingFrameStyle.Render(content)

	case stateSuccess:
		return ""

	case stateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
	}

	return ""
}


func (m *model) generateCommitMessage() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		message, err := m.aiClient.GenerateCommitMessage(ctx, m.diff)
		return msgCommitGenerated{
			message: strings.TrimSpace(message),
			err:     err,
		}
	})
}

func (m *model) commitChanges() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		err := git.CommitChanges(m.commitMessage)
		return msgCommitDone{err: err}
	})
}






func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	// Print success message after TUI exits so it remains visible
	if m.state == stateSuccess {
		successFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")).
			Padding(0, 1).
			Margin(1, 0)
		
		successMessage := successStyle.Render(fmt.Sprintf("✓ Committed: %s", m.commitMessage))
		fmt.Print(successFrame.Render(successMessage) + "\n")
	}
	
	return err
}

// Review TUI model
type reviewModel struct {
	aiClient *ai.VertexAIClient
	diff     string
	review   string
	err      error
	state    reviewState
	spinner  spinner.Model
	sub      chan msgReviewChunk
}

type reviewState int

const (
	reviewStateLoading reviewState = iota
	reviewStateStreaming
	reviewStateDisplay
	reviewStateError
)

type msgReviewGenerated struct {
	review string
	err    error
}

type msgReviewChunk struct {
	chunk string
}

type msgReviewComplete struct {
	err error
}

func NewReviewTUI(aiClient *ai.VertexAIClient, diff string) *reviewModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	
	return &reviewModel{
		aiClient: aiClient,
		diff:     diff,
		state:    reviewStateLoading,
		spinner:  s,
		sub:      make(chan msgReviewChunk, 100),
	}
}

func (m *reviewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateReviewStreaming(), m.waitForActivity(m.sub))
}

func (m *reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case reviewStateLoading, reviewStateStreaming:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case reviewStateDisplay, reviewStateError:
			return m, tea.Quit
		}

	case msgReviewChunk:
		if m.state == reviewStateLoading {
			m.state = reviewStateStreaming
		}
		m.review += msg.chunk
		return m, m.waitForActivity(m.sub)

	case msgReviewComplete:
		if msg.err != nil {
			m.err = msg.err
			m.state = reviewStateError
		} else {
			m.state = reviewStateDisplay
		}
		return m, tea.Quit

	case msgReviewGenerated:
		if msg.err != nil {
			m.err = msg.err
			m.state = reviewStateError
		} else {
			m.review = msg.review
			m.state = reviewStateDisplay
		}
		return m, tea.Quit
	}

	// Update spinner
	if m.state == reviewStateLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *reviewModel) View() string {
	switch m.state {
	case reviewStateLoading:
		content := fmt.Sprintf("%s %s",
			m.spinner.View(),
			generatingStyle.Render("Analyzing code for review..."))
		return loadingFrameStyle.Render(content)

	case reviewStateStreaming:
		// Display streaming content without frame
		return m.review

	case reviewStateDisplay:
		return "" // Review will be printed after TUI exits

	case reviewStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
	}

	return ""
}

func (m *reviewModel) generateReview() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		review, err := m.aiClient.ReviewCode(ctx, m.diff)
		return msgReviewGenerated{
			review: strings.TrimSpace(review),
			err:    err,
		}
	})
}

func (m *reviewModel) generateReviewStreaming() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		
		go func() {
			defer close(m.sub)
			
			err := m.aiClient.ReviewCodeStreaming(ctx, m.diff, func(chunk string) {
				select {
				case m.sub <- msgReviewChunk{chunk: chunk}:
				case <-ctx.Done():
					return
				}
			})
			
			if err != nil {
				m.sub <- msgReviewChunk{} // Signal error by closing
			}
		}()
		
		return nil
	})
}

func (m *reviewModel) waitForActivity(sub chan msgReviewChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-sub
		if !ok {
			// Channel closed, streaming completed
			return msgReviewComplete{err: nil}
		}
		return chunk
	}
}

func (m *reviewModel) Run() (string, error) {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	if m.err != nil {
		return "", m.err
	}
	
	return m.review, err
}

