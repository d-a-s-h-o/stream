package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cheggaaa/pb/v3"
)

type model struct {
	choices      []ContentItem
	filtered     []ContentItem
	textInput    textinput.Model
	err          error
	charCount    int
	loading      bool
	loadComplete bool
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

const (
	InitialVisibleItems = 10
	LoadMoreIncrement   = 10
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runTest()
	} else {
		p := tea.NewProgram(initialModel())
		if err := p.Start(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			return
		}
	}
}

func initialModel() model {
	ti := textinput.NewModel()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 20

	m := model{
		choices:      []ContentItem{},
		filtered:     []ContentItem{},
		textInput:    ti,
		err:          nil,
		charCount:    0,
		loading:      true,
		loadComplete: false,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return loadContent()
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
		m.charCount = len(m.textInput.Value())
		return m, nil

	case msgContentReceived:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.choices = msg.content
		m.loading = false
		m.loadComplete = true
		m.filtered = filterChoices(m.choices, m.textInput.Value())
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	b := strings.Builder{}
	b.WriteString(m.textInput.View() + "\n\n")

	// Calculate column widths
	nameWidth := 20 + (m.charCount/5)*5
	yearWidth := 10
	typeWidth := 10

	// Format the movie list
	visibleItems := InitialVisibleItems
	if visibleItems > len(m.filtered) {
		visibleItems = len(m.filtered)
	}

	for _, item := range m.filtered[:visibleItems] {
		// Truncate name if too long
		name := item.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		} else {
			name += strings.Repeat(" ", nameWidth-len(name))
		}

		// Apply styling using lipgloss
		name = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(name)
		year := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(fmt.Sprintf("%-*d", yearWidth, item.Year))
		contentType := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("%-*s", typeWidth, item.Type))
		url := lipgloss.NewStyle().Foreground(lipgloss.Color("100")).Render(item.Url)

		// Append the formatted line to the output
		b.WriteString(fmt.Sprintf("%s | %s | %s | URL: %s\n", name, year, contentType, url))
	}

	if !m.loadComplete && m.loading {
		// Show loading message
		b.WriteString("\n[Loading...]")
	} else if !m.loadComplete && !m.loading {
		// Show load more message
		b.WriteString("\n[Load more...]")
	}

	return b.String()
}

func loadContent() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get("https://raw.githubusercontent.com/d-a-s-h-o/stream/master/content.json")
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

		// Sort the content alphabetically
		sort.Slice(content, func(i, j int) bool {
			return strings.ToLower(content[i].Name) < strings.ToLower(content[j].Name)
		})

		return msgContentReceived{content, nil}
	}
}

// Filter the choices based on the filter string
func filterChoices(choices []ContentItem, filter string) []ContentItem {
	filter = strings.ReplaceAll(filter, " ", "")
	if filter == "" {
		return choices
	}

	filtered := make([]ContentItem, 0)
	for _, choice := range choices {
		name := strings.ReplaceAll(choice.Name, " ", "")
		if strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
			filtered = append(filtered, choice)
		}
	}

	return filtered
}

func runTest() {
	content, err := getContent()
	if err != nil {
		fmt.Printf("Error getting content: %v\n", err)
		return
	}

	bar := pb.StartNew(len(content))
	bar.SetRefreshRate(time.Millisecond * 100)

	deadItems := make([]string, 0)

	for _, item := range content {
		resp, err := http.Head(item.Url)
		if err != nil || resp.StatusCode != http.StatusOK {
			deadItems = append(deadItems, item.Name)
		}

		bar.Increment()
		time.Sleep(time.Millisecond * 10) // Simulate processing time

		resp.Body.Close()
	}

	bar.Finish()

	if len(deadItems) > 0 {
		fmt.Println("Dead items:")
		for _, item := range deadItems {
			fmt.Println(item)
		}
	} else {
		fmt.Println("No dead items found.")
	}
}

func getContent() ([]ContentItem, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/d-a-s-h-o/stream/master/content.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var content []ContentItem
	err = json.NewDecoder(resp.Body).Decode(&content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func testLinks(content []ContentItem) []ContentItem {
	var wg sync.WaitGroup
	deadItems := make([]ContentItem, 0)

	for _, item := range content {
		wg.Add(1)
		go func(item ContentItem) {
			defer wg.Done()

			resp, err := http.Get(item.Url)
			if err != nil || resp.StatusCode != http.StatusOK {
				deadItems = append(deadItems, item)
			}
		}(item)
	}

	wg.Wait()
	return deadItems
}
