package ui

import (
	"context"
	"fmt"
	"strings"

	"geminielf/internal/ai"
	"geminielf/internal/git"

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
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}

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
	return &model{
		aiClient: aiClient,
		diff:     diff,
		state:    stateLoading,
	}
}

func (m *model) Init() tea.Cmd {
	return m.generateCommitMessage()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = stateCommitting
				return m, m.commitChanges()
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
	}

	return m, nil
}

func (m *model) View() string {
	switch m.state {
	case stateLoading:
		return titleStyle.Render("🤖 geminielf") + "\n\n⏳ Generating commit message..."

	case stateConfirm:
		content := fmt.Sprintf("Generated commit message:\n\n%s\n\nCommit with this message? (y/n)", m.commitMessage)
		return titleStyle.Render("🤖 geminielf") + "\n" + confirmStyle.Render(content)

	case stateCommitting:
		return titleStyle.Render("🤖 geminielf") + "\n\n📝 Committing changes..."

	case stateSuccess:
		return titleStyle.Render("🤖 geminielf") + "\n\n" + successStyle.Render("✅ Changes committed successfully!")

	case stateError:
		return titleStyle.Render("🤖 geminielf") + "\n\n" + errorStyle.Render(fmt.Sprintf("❌ Error: %v", m.err))
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