package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	choices   []ContentItem
	filtered  []ContentItem
	textInput textinput.Model
	err       error
}

type msgContentReceived struct {
	content []ContentItem
	err     error
}

type ContentItem struct {
	Name string `json:"name"`
	Year int    `json:"year"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return
	}
}

func initialModel() model {
	ti := textinput.NewModel()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 20

	m := model{
		choices:   []ContentItem{},
		filtered:  []ContentItem{},
		textInput: ti,
		err:       nil,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return getContent
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyEnter:
			return m, tea.Quit
		}

		m.textInput, _ = m.textInput.Update(msg)
		m.filtered = filterChoices(m.choices, m.textInput.Value())
		return m, nil

	case msgContentReceived:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.choices = msg.content
		m.filtered = filterChoices(m.choices, m.textInput.Value())
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	b := strings.Builder{}
	b.WriteString(m.textInput.View() + "\n\n")

	// Format the movie list
	for _, item := range m.filtered {
		url := item.Url
		if len(url) > 20 {
			url = url[:10] + "..." + url[len(url)-10:]
		}

		// Apply styling using lipgloss
		name := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(item.Name)
		year := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(fmt.Sprintf("%d", item.Year))
		contentType := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(item.Type)
		urlShortened := lipgloss.NewStyle().Foreground(lipgloss.Color("100")).Render(url)

		// Append the formatted line to the output
		b.WriteString(fmt.Sprintf("%s | %s | %s | URL: %s\n", name, year, contentType, urlShortened))
	}

	return b.String()
}

func getContent() tea.Msg {
	resp, err := http.Get("https://dasho.dev/content.json")
	if err != nil {
		return msgContentReceived{nil, err}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return msgContentReceived{nil, err}
	}

	var content []ContentItem
	err = json.Unmarshal(body, &content)
	if err != nil {
		return msgContentReceived{nil, err}
	}

	return msgContentReceived{content, nil}
}

// Filter the choices based on the filter string
func filterChoices(choices []ContentItem, filter string) []ContentItem {
	if filter == "" {
		return choices
	}

	var filtered []ContentItem
	for _, choice := range choices {
		if strings.Contains(strings.ToLower(choice.Name), strings.ToLower(filter)) {
			filtered = append(filtered, choice)
		}
	}

	return filtered
}
