package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geminielf/internal/ai"
	"geminielf/internal/git"

	"github.com/charmbracelet/bubbles/progress"
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
	progress      progress.Model
	width         int
	height        int
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
	// グラデーションカラーパレット
	primaryGradient = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.AdaptiveColor{
				Light: "#667eea",
				Dark:  "#764ba2",
			})

	// メインタイトルスタイル
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#667eea")).
			Padding(1, 3).
			Margin(1, 0).
			Bold(true).
			Align(lipgloss.Center)

	// サブタイトルスタイル
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E")).
			Italic(true).
			Align(lipgloss.Center).
			MarginBottom(2)

	// 確認ダイアログスタイル
	confirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#667eea")).
			Padding(2, 3).
			Margin(1, 2).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})

	// コミットメッセージスタイル
	commitMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6EDF3")).
			Background(lipgloss.Color("#21262D")).
			Padding(1, 2).
			Margin(1, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Italic(true)

	// アクションボタンスタイル
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#28a745")).
			Padding(0, 2).
			Margin(0, 1).
			Bold(true).
			Border(lipgloss.RoundedBorder())

	cancelButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#dc3545")).
			Padding(0, 2).
			Margin(0, 1).
			Bold(true).
			Border(lipgloss.RoundedBorder())

	// 成功メッセージスタイル
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#28a745")).
			Padding(2, 3).
			Margin(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#28a745")).
			Bold(true)

	// エラーメッセージスタイル
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#dc3545")).
			Padding(2, 3).
			Margin(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#dc3545")).
			Bold(true)

	// ローディングスタイル
	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#667eea")).
			Bold(true).
			Margin(1, 0)

	// ヘルプテキストスタイル
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E")).
			Align(lipgloss.Center).
			MarginTop(1)

	// ボーダー装飾スタイル
	decoratorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#667eea")).
			Align(lipgloss.Center).
			Margin(0, 0, 1, 0)
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#667eea"))
	
	// プログレスバーの設定
	prog := progress.New(progress.WithDefaultGradient())
	prog.Full = '█'
	prog.Empty = '░'
	prog.Width = 40
	
	return &model{
		aiClient: aiClient,
		diff:     diff,
		state:    stateLoading,
		spinner:  s,
		progress: prog,
		width:    80,
		height:   24,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateCommitMessage())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(40, m.width-10)
		
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
	// メインタイトルとサブタイトル
	title := titleStyle.Render("🚀 geminielf")
	subtitle := subtitleStyle.Render("AI-Powered Git Commit Message Generator")
	decorator := decoratorStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	header := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		decorator,
	)
	
	switch m.state {
	case stateLoading:
		loadingText := loadingStyle.Render(fmt.Sprintf("%s Generating commit message...", m.spinner.View()))
		progressBar := m.progress.ViewAs(0.7) // シミュレートされたプログレス
		progressStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#667eea")).
			Align(lipgloss.Center).
			Margin(1, 0)
		styledProgress := progressStyle.Render(progressBar)
		
		loadingContent := lipgloss.JoinVertical(lipgloss.Center,
			loadingText,
			"",
			"🧠 AI is analyzing your changes...",
			styledProgress,
		)
		return lipgloss.JoinVertical(lipgloss.Center, header, "", loadingContent)

	case stateConfirm:
		messageBox := commitMessageStyle.Render(m.commitMessage)
		buttons := lipgloss.JoinHorizontal(lipgloss.Center,
			buttonStyle.Render("✓ Yes (y)"),
			cancelButtonStyle.Render("✗ No (n)"),
		)
		helpText := helpStyle.Render("Press 'y' to commit or 'n' to cancel")
		
		content := lipgloss.JoinVertical(lipgloss.Center,
			"📝 Generated Commit Message:",
			messageBox,
			"",
			"🤔 Commit with this message?",
			buttons,
			helpText,
		)
		
		confirmBox := confirmStyle.Render(content)
		return lipgloss.JoinVertical(lipgloss.Center, header, "", confirmBox)

	case stateCommitting:
		committingText := loadingStyle.Render(fmt.Sprintf("%s Committing changes...", m.spinner.View()))
		progressBar := m.progress.ViewAs(0.9) // コミット中の高いプログレス
		progressStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#28a745")).
			Align(lipgloss.Center).
			Margin(1, 0)
		styledProgress := progressStyle.Render(progressBar)
		
		committingContent := lipgloss.JoinVertical(lipgloss.Center,
			committingText,
			"",
			"💾 Applying changes to repository...",
			styledProgress,
		)
		return lipgloss.JoinVertical(lipgloss.Center, header, "", committingContent)

	case stateSuccess:
		successContent := lipgloss.JoinVertical(lipgloss.Center,
			"🎉 Success!",
			"",
			"✨ Your changes have been committed successfully!",
			"🚀 The AI-generated message has been applied.",
			"",
			"⏱️  Closing in 2 seconds...",
		)
		successBox := successStyle.Render(successContent)
		return lipgloss.JoinVertical(lipgloss.Center, header, "", successBox)

	case stateError:
		errorContent := lipgloss.JoinVertical(lipgloss.Center,
			"❌ Error Occurred",
			"",
			fmt.Sprintf("🔍 Details: %v", m.err),
			"",
			"💡 Please check your configuration and try again.",
			"🔧 Make sure Git is properly configured.",
			"",
			"⏱️  Closing in 2 seconds...",
		)
		errorBox := errorStyle.Render(errorContent)
		return lipgloss.JoinVertical(lipgloss.Center, header, "", errorBox)
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

// ヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}