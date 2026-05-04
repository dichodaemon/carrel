package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// Model holds the dashboard state.
type Model struct {
	reg     registry.Registry
	width   int
	height  int

	// Data
	consumers []registry.Consumer
	sources   []registry.Source
	entries   []registry.Entry

	// UI state
	tab      int    // 0=status, 1=verify, 2=entries
	cursor   int    // selected item
	loading  bool
	err      error
	quitting bool
}

// tabs defines the available dashboard tabs.
var tabs = []string{"Status", "Verify", "Entries"}

// NewModel creates a new dashboard model.
func NewModel(reg registry.Registry) Model {
	return Model{
		reg: reg,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadConsumers(m.reg),
		loadSources(m.reg),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % len(tabs)
			m.cursor = 0
		case "shift+tab", "left", "h":
			m.tab = (m.tab - 1 + len(tabs)) % len(tabs)
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			max := m.listLen()
			if m.cursor < max-1 {
				m.cursor++
			}
		case "r":
			// Refresh
			m.loading = true
			return m, tea.Batch(
				loadConsumers(m.reg),
				loadSources(m.reg),
			)
		}

	case consumersMsg:
		m.consumers = msg
		m.loading = false
		return m, loadSources(m.reg)

	case sourcesMsg:
		m.sources = msg
		m.loading = false

	case registryErr:
		m.err = msg
		m.loading = false
	}

	return m, nil
}

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	if m.loading {
		return tea.NewView("Loading...\n")
	}

	var b strings.Builder
	b.WriteString(renderTabs(m.tab, tabs))
	b.WriteString("\n")

	switch m.tab {
	case 0:
		b.WriteString(m.renderStatus())
	case 1:
		b.WriteString(m.renderVerify())
	case 2:
		b.WriteString(m.renderEntries())
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\nError: %v\n", m.err))
	}

	return tea.NewView(lipgloss.NewStyle().Margin(1).Render(b.String()))
}

func (m Model) listLen() int {
	switch m.tab {
	case 0:
		return len(m.consumers) + len(m.sources)
	case 1:
		return len(m.consumers)
	case 2:
		return len(m.entries)
	}
	return 0
}

func (m Model) renderStatus() string {
	var b strings.Builder
	b.WriteString("Consumers:\n")
	for i, c := range m.consumers {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %-20s %s\n", cursor, c.Alias, c.Path))
	}

	b.WriteString("\nSources:\n")
	for i, s := range m.sources {
		cursor := " "
		if m.cursor == len(m.consumers)+i {
			cursor = ">"
		}
		scope := "universal"
		if s.Scope == registry.ScopeTargetSpecific {
			scope = "target"
		}
		b.WriteString(fmt.Sprintf(" %s %-20s %s [%s]\n", cursor, s.Alias, s.Path, scope))
	}

	b.WriteString("\n[R]efresh  [Tab] next  [Q]uit\n")
	return b.String()
}

func (m Model) renderVerify() string {
	var b strings.Builder
	b.WriteString("Deployment verification:\n\n")
	for i, c := range m.consumers {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		_, _, err := m.reg.LastDeployment(c.ID)
		status := "deployed"
		if err == registry.ErrNoDeployment {
			status = "no deployments"
		} else if err != nil {
			status = fmt.Sprintf("error: %v", err)
		}
		b.WriteString(fmt.Sprintf(" %s %-20s %s\n", cursor, c.Alias, status))
	}
	b.WriteString("\n[R]efresh  [Tab] next  [Q]uit\n")
	return b.String()
}

func (m Model) renderEntries() string {
	if len(m.entries) == 0 {
		return "No entries loaded.\n\n[Tab] back  [Q]uit\n"
	}
	var b strings.Builder
	b.WriteString("Entries:\n")
	for i, e := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %s/%s (%d)\n", cursor, e.Name, e.RelativePath, e.Type))
	}
	b.WriteString("\n[R]efresh  [Tab] next  [Q]uit\n")
	return b.String()
}

// Messages for async I/O

type consumersMsg []registry.Consumer
type sourcesMsg []registry.Source
type registryErr error

func loadConsumers(reg registry.Registry) tea.Cmd {
	return func() tea.Msg {
		consumers, err := reg.ListConsumers()
		if err != nil {
			return registryErr(err)
		}
		return consumersMsg(consumers)
	}
}

func loadSources(reg registry.Registry) tea.Cmd {
	return func() tea.Msg {
		sources, err := reg.ListSources()
		if err != nil {
			return registryErr(err)
		}
		return sourcesMsg(sources)
	}
}

// Ensure unused imports are retained
var _ = uuid.New


func renderTabs(active int, tabs []string) string {
	var parts []string
	for i, t := range tabs {
		if i == active {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Underline(true).Render(t))
		} else {
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(t))
		}
	}
	return strings.Join(parts, " | ")
}