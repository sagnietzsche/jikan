package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tickInterval       = time.Second
	defaultDBPath      = "jikan.db"
	defaultCSVPath     = "jikan_sessions.csv"
	statusMessageWidth = 96
)

type project struct {
	Name    string
	Elapsed time.Duration
}

type tickMsg time.Time

type model struct {
	projects  []project
	cursor    int
	active    int
	startedAt time.Time
	now       time.Time
	store     sessionStore
	status    string

	creating bool
	input    textinput.Model

	width  int
	height int
}

var (
	sumi       = lipgloss.Color("#161616")
	ink        = lipgloss.Color("#EDE6D6")
	muted      = lipgloss.Color("#9D9A92")
	gold       = lipgloss.Color("#C9A227")
	vermilion  = lipgloss.Color("#B4433F")
	pine       = lipgloss.Color("#2F5D50")
	washi      = lipgloss.Color("#F7F1E1")
	shadow     = lipgloss.Color("#2A2A2A")
	selectedBG = lipgloss.Color("#27231C")

	baseStyle = lipgloss.NewStyle().
			Foreground(ink).
			Background(sumi)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(gold).
			Padding(1, 2).
			Background(sumi)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(washi)

	titleMarkStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(vermilion)

	mutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	accentStyle = lipgloss.NewStyle().
			Foreground(gold)

	activeStyle = lipgloss.NewStyle().
			Foreground(vermilion).
			Bold(true)

	readyStyle = lipgloss.NewStyle().
			Foreground(pine)

	rowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedRowStyle = rowStyle.Copy().
				Foreground(washi).
				Background(selectedBG)

	helpStyle = lipgloss.NewStyle().
			Foreground(muted)
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "jikan: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("jikan", flag.ContinueOnError)
	dbPath := flags.String("db", defaultDBPath, "path to the sqlite database")
	exportPath := flags.String("export", "", "export tracked sessions to a CSV file and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	store, err := openSQLiteSessionStore(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	if *exportPath != "" {
		count, err := store.ExportCSV(context.Background(), *exportPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "exported %d sessions to %s\n", count, *exportPath)
		return nil
	}

	p := tea.NewProgram(newModelWithStore(time.Now(), store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func newModel() model {
	return newModelAt(time.Now())
}

func newModelAt(now time.Time) model {
	return newModelWithStore(now, nil)
}

func newModelWithStore(now time.Time, store sessionStore) model {
	input := textinput.New()
	input.Placeholder = "New project"
	input.CharLimit = 48
	input.Prompt = "Name "
	input.PromptStyle = accentStyle
	input.TextStyle = titleStyle
	input.Cursor.Style = activeStyle

	return model{
		projects: []project{
			{Name: "Morning Planning"},
			{Name: "Deep Work"},
			{Name: "Reading and Research"},
		},
		active: -1,
		now:    now,
		store:  store,
		input:  input,
		width:  86,
		height: 24,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), textinput.Blink)
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		m.now = time.Time(msg)
		return m, tick()
	case tea.KeyMsg:
		if m.creating {
			return m.updateCreation(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.stopActive()
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case " ", "enter":
			m.toggleSelected()
		case "n":
			m.creating = true
			m.status = ""
			m.input.SetValue("")
			m.input.Focus()
			return m, textinput.Blink
		case "r":
			m.resetSelected()
		case "e":
			m.exportSessions(defaultCSVPath)
		}
	}

	return m, nil
}

func (m model) updateCreation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stopActive()
		return m, tea.Quit
	case "esc":
		m.creating = false
		m.status = ""
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			return m, nil
		}
		m.projects = append(m.projects, project{Name: name})
		m.cursor = len(m.projects) - 1
		m.creating = false
		m.status = fmt.Sprintf("added %s", trimToWidth(name, statusMessageWidth))
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *model) moveCursor(delta int) {
	if len(m.projects) == 0 {
		m.cursor = 0
		return
	}

	m.cursor = (m.cursor + delta + len(m.projects)) % len(m.projects)
}

func (m *model) toggleSelected() {
	if len(m.projects) == 0 {
		return
	}

	if m.active == m.cursor {
		m.stopActive()
		return
	}

	if m.active >= 0 {
		m.stopActive()
	}

	m.active = m.cursor
	m.startedAt = m.now
}

func (m *model) stopActive() {
	if m.active < 0 || m.active >= len(m.projects) {
		m.active = -1
		m.startedAt = time.Time{}
		return
	}

	projectName := m.projects[m.active].Name
	duration := elapsedSince(m.startedAt, m.now)
	m.projects[m.active].Elapsed += duration
	if err := m.recordSession(projectName, m.startedAt, m.now, duration); err != nil {
		m.status = fmt.Sprintf("save failed: %s", trimToWidth(err.Error(), statusMessageWidth))
	} else if duration > 0 {
		m.status = fmt.Sprintf("saved %s %s", trimToWidth(projectName, 48), formatDuration(duration))
	}
	m.active = -1
	m.startedAt = time.Time{}
}

func (m *model) recordSession(projectName string, startedAt, endedAt time.Time, duration time.Duration) error {
	if duration <= 0 || m.store == nil {
		return nil
	}

	return m.store.RecordSession(context.Background(), trackedSession{
		ProjectName: projectName,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		Duration:    duration,
	})
}

func (m *model) exportSessions(path string) {
	if m.store == nil {
		m.status = "export unavailable: no session store"
		return
	}

	count, err := m.store.ExportCSV(context.Background(), path)
	if err != nil {
		m.status = fmt.Sprintf("export failed: %s", trimToWidth(err.Error(), statusMessageWidth))
		return
	}
	m.status = fmt.Sprintf("exported %d sessions to %s", count, path)
}

func (m *model) resetSelected() {
	if len(m.projects) == 0 {
		return
	}

	m.projects[m.cursor].Elapsed = 0
	if m.active == m.cursor {
		m.startedAt = m.now
	}
	m.status = fmt.Sprintf("reset display total for %s", trimToWidth(m.projects[m.cursor].Name, 48))
}

func (m model) elapsedFor(index int) time.Duration {
	if index < 0 || index >= len(m.projects) {
		return 0
	}

	elapsed := m.projects[index].Elapsed
	if m.active == index {
		elapsed += elapsedSince(m.startedAt, m.now)
	}
	return elapsed.Truncate(time.Second)
}

func (m model) totalElapsed() time.Duration {
	var total time.Duration
	for i := range m.projects {
		total += m.elapsedFor(i)
	}
	return total
}

func elapsedSince(start, end time.Time) time.Duration {
	if start.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

func (m model) View() string {
	frameWidth := m.frameWidth()
	width := frameWidth - 6

	var b strings.Builder
	b.WriteString(m.header(width))
	b.WriteString("\n\n")
	b.WriteString(m.summary(width))
	b.WriteString("\n\n")
	b.WriteString(m.projectList(width))

	if m.creating {
		b.WriteString("\n\n")
		b.WriteString(m.creationBox(width))
	}

	b.WriteString("\n\n")
	if m.status != "" {
		b.WriteString(mutedStyle.Render(trimToWidth(m.status, width)))
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("j/k move  space start/stop  n new  r reset  e export  q quit"))

	return baseStyle.Render(frameStyle.Width(frameWidth).Render(b.String()))
}

func (m model) frameWidth() int {
	width := m.width
	if width <= 0 {
		width = 86
	}

	width -= 2
	if width < 48 {
		return 48
	}
	if width > 102 {
		return 102
	}
	return width
}

func (m model) header(width int) string {
	left := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleMarkStyle.Render("time"),
		titleStyle.Render("  jikan"),
	)
	right := mutedStyle.Render(m.now.Format("2006.01.02"))
	space := width - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

func (m model) summary(width int) string {
	activeName := "none"
	if m.active >= 0 && m.active < len(m.projects) {
		activeName = m.projects[m.active].Name
	}

	total := accentStyle.Render(formatDuration(m.totalElapsed()))
	active := activeStyle.Render(trimToWidth(activeName, width-24))

	line := fmt.Sprintf("Total %s   Running %s", total, active)
	return lipgloss.NewStyle().
		Foreground(ink).
		Border(lipgloss.ThickBorder(), false, false, true, false).
		BorderForeground(shadow).
		Padding(0, 0, 1, 0).
		Width(width).
		Render(line)
}

func (m model) projectList(width int) string {
	if len(m.projects) == 0 {
		return mutedStyle.Render("No projects yet. Press n to add one.")
	}

	showStatus := width >= 58
	nameWidth := width - 20
	if showStatus {
		nameWidth = width - 31
	}
	if nameWidth < 10 {
		nameWidth = 10
	}

	lines := make([]string, 0, len(m.projects))
	for i, p := range m.projects {
		selected := i == m.cursor
		running := i == m.active

		marker := " "
		if selected {
			marker = ">"
		}

		state := readyStyle.Render("idle")
		symbol := mutedStyle.Render("-")
		if running {
			state = activeStyle.Render("running")
			symbol = activeStyle.Render("*")
		}

		name := trimToWidth(p.Name, nameWidth)
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			mutedStyle.Width(2).Render(marker),
			lipgloss.NewStyle().Width(2).Render(symbol),
			lipgloss.NewStyle().Width(nameWidth).Render(name),
			accentStyle.Width(10).Align(lipgloss.Right).Render(formatDuration(m.elapsedFor(i))),
		)
		if showStatus {
			row = lipgloss.JoinHorizontal(
				lipgloss.Top,
				row,
				lipgloss.NewStyle().Width(3).Render(""),
				lipgloss.NewStyle().Width(8).Align(lipgloss.Right).Render(state),
			)
		}

		style := rowStyle
		if selected {
			style = selectedRowStyle
		}
		lines = append(lines, style.Width(width).Render(row))
	}

	return strings.Join(lines, "\n")
}

func (m model) creationBox(width int) string {
	boxWidth := width - 4
	if boxWidth < 30 {
		boxWidth = 30
	}
	if boxWidth > 72 {
		boxWidth = 72
	}

	content := strings.Join([]string{
		titleStyle.Render("New task"),
		m.input.View(),
		mutedStyle.Render("enter save  esc cancel"),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pine).
		Padding(1, 2).
		Width(boxWidth).
		Render(content)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	total := int64(d.Truncate(time.Second).Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}

	const marker = "..."
	target := width - lipgloss.Width(marker)
	if target <= 0 {
		return strings.Repeat(".", width)
	}

	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next) > target {
			break
		}
		b.WriteRune(r)
	}

	return b.String() + marker
}
