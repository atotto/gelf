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
	"github.com/charmbracelet/glamour"
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
	diffSummary     git.DiffSummary
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






func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = loadingStyle
	
	ti := textinput.New()
	ti.Placeholder = "Enter your commit message..."
	ti.CharLimit = 200
	ti.Width = 60
	
	diffSummary := git.ParseDiffSummary(diff)
	
	return &model{
		aiClient:    aiClient,
		diff:        diff,
		diffSummary: diffSummary,
		state:       stateLoading,
		spinner:     s,
		textInput:   ti,
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
		loadingText := fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Generating commit message..."))
		
		diffSummary := m.formatDiffSummary()
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s", diffSummary, loadingText)
		}
		return loadingText

	case stateConfirm:
		diffSummary := m.formatDiffSummary()
		header := titleStyle.Render("📝 Generated Commit Message:")
		message := messageStyle.Render(m.commitMessage)
		prompt := promptStyle.Render("Commit this message? (y)es / (e)dit / (n)o")
		
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", diffSummary, header, message, prompt)
		}
		return fmt.Sprintf("%s\n\n%s\n\n%s", header, message, prompt)

	case stateEditing:
		diffSummary := m.formatDiffSummary()
		header := titleStyle.Render("✏️  Edit Commit Message:")
		inputView := m.textInput.View()
		prompt := editPromptStyle.Render("Press Enter to confirm, Esc to cancel")
		
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", diffSummary, header, inputView, prompt)
		}
		return fmt.Sprintf("%s\n\n%s\n\n%s", header, inputView, prompt)

	case stateCommitting:
		return fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Committing changes..."))

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

func (m *model) formatDiffSummary() string {
	if len(m.diffSummary.Files) == 0 {
		return ""
	}
	
	var parts []string
	parts = append(parts, diffStyle.Render("📄 Changed Files:"))
	
	for _, file := range m.diffSummary.Files {
		fileName := fileStyle.Render(file.Name)
		
		var changes []string
		if file.AddedLines > 0 {
			changes = append(changes, addedStyle.Render(fmt.Sprintf("+%d", file.AddedLines)))
		}
		if file.DeletedLines > 0 {
			changes = append(changes, deletedStyle.Render(fmt.Sprintf("-%d", file.DeletedLines)))
		}
		
		if len(changes) > 0 {
			parts = append(parts, fmt.Sprintf(" • %s (%s)", fileName, strings.Join(changes, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf(" • %s", fileName))
		}
	}
	
	return strings.Join(parts, "\n")
}






func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	// Print success message after TUI exits so it remains visible
	if m.state == stateSuccess {
		header := successStyle.Render("✓ COMMIT SUCCESSFUL")
		message := messageStyle.Render(m.commitMessage)
		
		fmt.Printf("%s\n%s\n", header, message)
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
	noStyle  bool
	renderer *glamour.TermRenderer
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

// リッチなカラースタイル（枠なし）
var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")) // シアン

	messageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).   // ブライトホワイト
		Bold(true).
		Italic(true)

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).    // ブルー
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).    // グリーン
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).    // レッド
		Bold(true)

	loadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).    // シアン
		Bold(true)

	editPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).    // イエロー
		Bold(true).
		Margin(1, 0, 0, 0)
	
	diffStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).    // ライトグレー
		Margin(1, 0, 0, 0)
	
	fileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).    // マゼンタ
		Bold(true)
	
	addedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))     // グリーン
	
	deletedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))     // レッド

)

func NewReviewTUI(aiClient *ai.VertexAIClient, diff string, noStyle bool) *reviewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = loadingStyle
	
	var renderer *glamour.TermRenderer
	if !noStyle {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(80),
		)
		if err == nil {
			renderer = r
		}
	}
	
	return &reviewModel{
		aiClient: aiClient,
		diff:     diff,
		state:    reviewStateLoading,
		spinner:  s,
		sub:      make(chan msgReviewChunk, 100),
		noStyle:  noStyle,
		renderer: renderer,
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
		return fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Analyzing code for review..."))

	case reviewStateStreaming:
		// Display streaming content with styling if available
		if m.noStyle || m.renderer == nil {
			return m.review
		}
		
		// Try to render the current content with glamour
		styled, err := m.renderer.Render(m.review)
		if err != nil {
			// Fallback to plain text if rendering fails
			return m.review
		}
		
		return styled

	case reviewStateDisplay:
		return "" // Review will be printed after TUI exits

	case reviewStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
	}

	return ""
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

