package antagent

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextInputModel struct {
	m     textinput.Model
	text  string
	width int
}

func GetInput() (r string, err error) {
	var m tea.Model
	if m, err = tea.NewProgram(NewTextInputModel()).Run(); err != nil {
		return
	}
	if m, ok := m.(TextInputModel); ok {
		r = strings.TrimSpace(m.text)
		return
	}
	err = fmt.Errorf("unknown model type")
	return
}

func NewTextInputModel() TextInputModel {
	ti := textinput.New()
	ti.Placeholder = "Type your message... or \\command..."
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 100

	return TextInputModel{m: ti}
}

func (this TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (this TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		this.width = msg.Width
		if this.width-4 > 0 {
			this.m.Width = this.width - 4 - len(this.m.Prompt)
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			this.text = this.m.Value()
			return this, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			return this, tea.Quit
		}
	}

	this.m, cmd = this.m.Update(msg)
	return this, cmd
}

func (this TextInputModel) View() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#C49C7B")). // RGB: 196, 156, 123
		Padding(0, 1).
		Width(this.width-2).
		Render(this.m.View()) + "\n"
}

func PrintLogo() {
	// Gradient colors for the logo
	colors := []string{
		"#4285F4", // Google Blue
		"#5B92E5",
		"#739FD6",
		"#8CACC7",
		"#A5B9B8", // Transition to White/Grey
	}

	// 3D Block ASCII Art
	logoLines := []string{
		`    ____                    ____                                   __  `,
		`   / __ \___  ___  ____    / __ \___  ________  ____  ________  / /_ `,
		`  / / / / _ \/ _ \/ __ \  / /_/ / _ \/ ___/ _ \/ __ \/ ___/ _ \/ __ \`,
		` / /_/ /  __/  __/ /_/ / / _, _/  __(__  )  __/ /_/ / /  /  __/ / / /`,
		`/_____/\___/\___/ .___/ /_/ |_|\___/____/\___/\__,_/_/   \___/_/ /_/ `,
		`               /_/                                                   `,
	}

	fmt.Println()
	for i, line := range logoLines {
		color := colors[0]
		if i < len(colors) {
			color = colors[i]
		}
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(line))
	}

	// Subtitle with decorative elements
	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00BFFF")).
		Italic(true).
		Render("✨  AI-Powered Deep Research Assistant  ✨")

	border := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render("───────────────────────────────────────────────────")

	fmt.Println()
	fmt.Println(lipgloss.PlaceHorizontal(80, lipgloss.Center, subtitle))
	fmt.Println(lipgloss.PlaceHorizontal(80, lipgloss.Center, border))
	fmt.Println()
}
