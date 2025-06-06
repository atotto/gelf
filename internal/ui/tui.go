package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type msgAutoQuit struct{}


var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	confirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	
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
			return m, m.autoQuitAfterDelay()
		} else {
			m.state = stateSuccess
			return m, m.autoQuitAfterDelay()
		}
		
	case msgAutoQuit:
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
	title := titleStyle.Render("🤖 geminielf")
	
	switch m.state {
	case stateLoading:
		return fmt.Sprintf("%s\n\n%s Generating commit message...", title, m.spinner.View())

	case stateConfirm:
		content := fmt.Sprintf("Generated commit message:\n\n%s\n\nCommit with this message? (y/n)", m.commitMessage)
		return title + "\n" + confirmStyle.Render(content)

	case stateCommitting:
		return fmt.Sprintf("%s\n\n%s Committing changes...", title, m.spinner.View())

	case stateSuccess:
		successMsg := "🎉 Changes committed successfully!\n\n✨ Your commit has been created with the generated message."
		return title + "\n\n" + successStyle.Render(successMsg)

	case stateError:
		errorMsg := fmt.Sprintf("❌ Error: %v\n\n💡 Please check your configuration and try again.", m.err)
		return title + "\n\n" + errorStyle.Render(errorMsg)
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

func (m *model) autoQuitAfterDelay() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return msgAutoQuit{}
	})
}


func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}