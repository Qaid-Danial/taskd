package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Qaid-Danial/taskd/internal/store"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	tabStyle    = lipgloss.NewStyle().Padding(0, 1)
	activeTab   = tabStyle.Bold(true).Reverse(true)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	overdueHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	selected    = lipgloss.NewStyle().Reverse(true)
	doneStyle   = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	faint       = lipgloss.NewStyle().Faint(true)
	red         = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	labelStyle  = lipgloss.NewStyle().Faint(true).Width(11)

	priorityStyles = map[int]lipgloss.Style{
		1: lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		2: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		3: lipgloss.NewStyle(),
		4: lipgloss.NewStyle().Faint(true),
	}
	priorityNames = map[int]string{1: "urgent", 2: "important", 3: "normal", 4: "someday"}
	statusMarks   = map[string]string{"todo": "[ ]", "doing": "[~]", "done": "[x]"}
)

const help = "j/k move  ←/→ prev/next  t today  tab view  enter details  h hide done  r refresh  q quit"

// View draws the screen from the current state. It never changes the model.
func (m Model) View() tea.View {
	var s string
	if m.detail {
		s = m.detailView()
	} else {
		s = m.listView()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m Model) header() string {
	var tabs []string
	for i, name := range viewNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if view(i) == m.view {
			tabs = append(tabs, activeTab.Render(label))
		} else {
			tabs = append(tabs, tabStyle.Render(label))
		}
	}

	from, to := span(m.view, m.anchor)
	var period string
	switch m.view {
	case dayView:
		period = from.Format("Monday, 2 January 2006")
	case weekView:
		period = from.Format("2 Jan") + " - " + to.Format("2 Jan 2006")
	case monthView:
		period = from.Format("January 2006")
	}

	status := ""
	if m.loading {
		status = faint.Render("  loading...")
	}
	return titleStyle.Render("taskd") + "  " + strings.Join(tabs, "") + "  " + period + status
}

func (m Model) listView() string {
	var lines []string // body lines
	cursorLine := 0
	items := m.items()

	if m.err != nil {
		lines = append(lines, errStyle.Render("Error: "+m.err.Error()), faint.Render("Press r to try again."))
	} else if !m.loading && len(items) == 0 {
		lines = append(lines, faint.Render("Nothing here. Enjoy the free time."))
	}

	// Walk the items in screen order and add a heading whenever the group
	// changes. Overdue tasks come first and share one heading.
	nOverdue := 0
	for _, t := range m.overdue {
		if !(m.hideDone && t.Done()) {
			nOverdue++
		}
	}
	group := ""
	todayStr := today().Format(dateLayout)
	for i, t := range items {
		g := t.Day()
		if i < nOverdue {
			g = "overdue"
		}
		if g != group {
			group = g
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			switch {
			case g == "overdue":
				lines = append(lines, overdueHead.Render("Overdue"))
			case g == todayStr:
				lines = append(lines, headerStyle.Render(pretty(g)+" (today)"))
			default:
				lines = append(lines, headerStyle.Render(pretty(g)))
			}
		}
		if i == m.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.taskLine(t, i == m.cursor, i < nOverdue))
	}

	// Keep the cursor on screen: header (2 lines) + body + footer (2 lines).
	body := m.height - 4
	if body < 1 {
		body = len(lines)
	}
	offset := 0
	if len(lines) > body {
		offset = cursorLine - body/2
		offset = max(0, min(offset, len(lines)-body))
		lines = lines[offset : offset+body]
	}
	for len(lines) < body {
		lines = append(lines, "")
	}

	footer := help
	if m.hideDone {
		footer += faint.Render("  (done hidden)")
	}
	out := []string{m.header(), ""}
	out = append(out, lines...)
	out = append(out, "", faint.Render(footer))
	return m.fit(strings.Join(out, "\n"))
}

func (m Model) taskLine(t store.Task, isCursor, isOverdue bool) string {
	mark := statusMarks[t.Status]
	prio := fmt.Sprintf("P%d", t.Priority)

	var extra []string
	if t.Area != nil {
		extra = append(extra, *t.Area)
	}
	if isOverdue {
		extra = append(extra, "from "+pretty(t.Day()))
	}
	if t.DueDate != nil && t.PlannedFor != nil && *t.DueDate != *t.PlannedFor {
		extra = append(extra, "due "+pretty(*t.DueDate))
	}
	tail := ""
	if len(extra) > 0 {
		tail = "  " + strings.Join(extra, " · ")
	}

	if isCursor {
		return selected.Render(fmt.Sprintf("> %s %s %s%s", mark, prio, t.Title, tail))
	}
	if t.Done() {
		return "  " + doneStyle.Render(fmt.Sprintf("%s %s %s", mark, prio, t.Title)) + faint.Render(tail)
	}
	line := "  " + mark + " " + priorityStyles[t.Priority].Render(prio) + " " + t.Title
	if isOverdue {
		return line + red.Render(tail)
	}
	return line + faint.Render(tail)
}

func (m Model) detailView() string {
	items := m.items()
	if m.cursor >= len(items) {
		return ""
	}
	t := items[m.cursor]

	opt := func(p *string) string {
		if p == nil || *p == "" {
			return faint.Render("-")
		}
		return *p
	}
	date := func(p *string) string {
		if p == nil {
			return faint.Render("-")
		}
		return pretty(*p)
	}
	row := func(label, value string) string { return labelStyle.Render(label) + value }

	estimate := faint.Render("-")
	if t.EstimateMinutes != nil {
		estimate = fmt.Sprintf("%d min", *t.EstimateMinutes)
	}
	tags := faint.Render("-")
	if len(t.Tags) > 0 {
		tags = strings.Join(t.Tags, ", ")
	}
	completed := faint.Render("-")
	if t.CompletedAt != nil {
		completed = t.CompletedAt.In(kl).Format("Mon 2 Jan 2006 15:04")
	}

	lines := []string{
		m.header(),
		"",
		titleStyle.Render(t.Title),
		"",
		row("Status", t.Status),
		row("Priority", fmt.Sprintf("P%d %s", t.Priority, priorityNames[t.Priority])),
		row("Area", opt(t.Area)),
		row("Tags", tags),
		row("Planned", date(t.PlannedFor)),
		row("Due", date(t.DueDate)),
		row("Estimate", estimate),
		row("Completed", completed),
		row("Created", t.CreatedAt.In(kl).Format("Mon 2 Jan 2006 15:04")),
		row("Updated", t.UpdatedAt.In(kl).Format("Mon 2 Jan 2006 15:04")),
		"",
		labelStyle.Render("Notes"),
		opt(t.Notes),
		"",
		faint.Render("esc back  q quit"),
	}
	return m.fit(strings.Join(lines, "\n"))
}

// fit cuts every line to the window width so long titles don't wrap and
// push the layout around.
func (m Model) fit(s string) string {
	if m.width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	return strings.Join(lines, "\n")
}