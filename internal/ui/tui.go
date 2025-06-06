package ui

import (
	"context"
	"fmt"
	"strings"

	"geminielf/internal/ai"
	"geminielf/internal/git"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLoading state = iota
	stateConfirm
	stateCommitting
	stateSuccess
	stateError
)

type model struct {
	aiClient      *ai.VertexAIClient
	diff          string
	commitMessage string
	err           error
	state         state
	spinner       spinner.Model
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}



var (
	// シンプルなスタイル
	titleStyle = lipgloss.NewStyle().Bold(true)
	
	messageStyle = lipgloss.NewStyle().
		Padding(1, 0).
		Italic(true)

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	
	return &model{
		aiClient: aiClient,
		diff:     diff,
		state:    stateLoading,
		spinner:  s,
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
		case stateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = stateCommitting
				return m, tea.Batch(m.spinner.Tick, m.commitChanges())
			case "n", "N", "q", "ctrl+c":
				return m, tea.Quit
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
		return fmt.Sprintf("%s Generating commit message...", m.spinner.View())

	case stateConfirm:
		return fmt.Sprintf("%s\n\n%s\n\n%s",
			messageStyle.Render(m.commitMessage),
			"Commit this message?",
			promptStyle.Render("(y)es / (n)o"))

	case stateCommitting:
		return fmt.Sprintf("%s Committing changes...", m.spinner.View())

	case stateSuccess:
		return successStyle.Render(fmt.Sprintf("Committed: %s", m.commitMessage))

	case stateError:
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
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
	return err
}

