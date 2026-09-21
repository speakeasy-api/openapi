package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// InputModel is a simple text input TUI for getting file paths
type InputModel struct {
	textInput textinput.Model
	prompt    string
	err       error
	submitted bool
	cancelled bool
}

// NewInputModel creates a new input model with the given prompt and default value
func NewInputModel(prompt, defaultValue string) InputModel {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.SetValue(defaultValue)
	ti.Focus()
	ti.CharLimit = 256
	ti.SetWidth(60)

	return InputModel{
		textInput: ti,
		prompt:    prompt,
		err:       nil,
		submitted: false,
		cancelled: false,
	}
}

// Init initializes the model
func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.submitted = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the input
func (m InputModel) View() tea.View {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorThemePurple)).
		Padding(1, 2).
		Width(70)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorThemePurple))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorGray)).
		Italic(true)

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		titleStyle.Render(m.prompt),
		m.textInput.View(),
		helpStyle.Render("Enter: confirm • Esc: cancel"))

	return tea.NewView(lipgloss.Place(80, 24, lipgloss.Center, lipgloss.Center, style.Render(content)))
}

// GetValue returns the submitted value
func (m InputModel) GetValue() string {
	if m.submitted {
		return m.textInput.Value()
	}
	return ""
}

// IsCancelled returns true if the user cancelled
func (m InputModel) IsCancelled() bool {
	return m.cancelled
}

// PromptForFilePath shows a TUI prompt for a file path with a default value
// Returns the path or empty string if cancelled
func PromptForFilePath(prompt, defaultValue string) (string, error) {
	m := NewInputModel(prompt, defaultValue)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running input prompt: %w", err)
	}

	inputModel, ok := finalModel.(InputModel)
	if !ok {
		return "", errors.New("unexpected model type")
	}

	if inputModel.IsCancelled() {
		return "", nil
	}

	value := strings.TrimSpace(inputModel.GetValue())
	return value, nil
}
