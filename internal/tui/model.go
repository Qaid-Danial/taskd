// Package tui is the Bubble Tea app. It gets tasks through the Source
// interface and never sees the Supabase URL or key.
package tui

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Qaid-Danial/taskd/internal/store"
)

// Source is everything the TUI needs from the data layer. *store.Store
// satisfies it, and tests can pass a fake.
type Source interface {
	Tasks(ctx context.Context, from, to string) ([]store.Task, error)
	Overdue(ctx context.Context, before string) ([]store.Task, error)
}

// Model is the whole state of the app.
type Model struct {
	src Source

	view   view      // Day, Week or Month
	anchor time.Time // any day inside the period on screen

	tasks   []store.Task // tasks in the period
	overdue []store.Task // unfinished tasks from before the period
	loading bool
	err     error
	reqID   int // id of the latest load, to ignore slow old answers

	cursor   int  // index into m.items()
	hideDone bool // h toggles this
	detail   bool // showing one task's details

	width, height int
}

// loadedMsg carries the result of a load back into Update.
type loadedMsg struct {
	id      int
	tasks   []store.Task
	overdue []store.Task
	err     error
}

// New returns the starting state: Day view on today, with the first load
// already marked as in flight (Init starts it).
func New(src Source) Model {
	return Model{src: src, view: dayView, anchor: today(), loading: true, reqID: 1}
}

// Init runs once at startup and starts the first load. Init can't return a
// changed model, which is why New already set loading and reqID.
func (m Model) Init() tea.Cmd {
	return m.fetch()
}

// load marks the model as loading and starts a new fetch.
func (m Model) load() (Model, tea.Cmd) {
	m.reqID++
	m.loading = true
	m.err = nil
	return m, m.fetch()
}

// fetch returns a command that loads the period on screen. Bubble Tea runs
// it in the background, so the UI never freezes on the network.
func (m Model) fetch() tea.Cmd {
	id, src := m.reqID, m.src
	from, to := span(m.view, m.anchor)
	t := today()
	withOverdue := !t.Before(from) && !t.After(to) // only when today is on screen

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		tasks, err := src.Tasks(ctx, from.Format(dateLayout), to.Format(dateLayout))
		if err != nil {
			return loadedMsg{id: id, err: err}
		}
		var overdue []store.Task
		if withOverdue {
			// Before the period's first day, so nothing shows up twice.
			overdue, err = src.Overdue(ctx, from.Format(dateLayout))
		}
		return loadedMsg{id: id, tasks: tasks, overdue: overdue, err: err}
	}
}

// Update handles every event and returns the new state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case loadedMsg:
		if msg.id != m.reqID {
			return m, nil // an older request finished late; ignore it
		}
		m.loading = false
		m.err = msg.err
		m.tasks = sortTasks(msg.tasks)
		m.overdue = sortTasks(msg.overdue)
		m.clampCursor()
		return m, nil

	case tea.KeyPressMsg:
		if m.detail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "enter", "backspace":
		m.detail = false
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		m.cursor++
	case "k", "up":
		m.cursor--
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.items()) - 1

	case "enter":
		if len(m.items()) > 0 {
			m.detail = true
		}

	case "h":
		m.hideDone = !m.hideDone

	case "tab":
		return m.switchView((m.view + 1) % 3)
	case "shift+tab":
		return m.switchView((m.view + 2) % 3)
	case "1":
		return m.switchView(dayView)
	case "2":
		return m.switchView(weekView)
	case "3":
		return m.switchView(monthView)

	case "left":
		m.anchor = shift(m.view, m.anchor, -1)
		m.cursor = 0
		return m.load()
	case "right":
		m.anchor = shift(m.view, m.anchor, 1)
		m.cursor = 0
		return m.load()
	case "t":
		m.anchor = today()
		m.cursor = 0
		return m.load()

	case "r":
		return m.load()
	}
	m.clampCursor()
	return m, nil
}

func (m Model) switchView(v view) (tea.Model, tea.Cmd) {
	if v == m.view {
		return m, nil
	}
	m.view = v
	m.cursor = 0
	return m.load()
}

// items is the list the cursor moves through, in screen order: overdue
// first, then the period's tasks. Done tasks drop out when hidden.
func (m Model) items() []store.Task {
	var out []store.Task
	for _, list := range [][]store.Task{m.overdue, m.tasks} {
		for _, t := range list {
			if m.hideDone && t.Done() {
				continue
			}
			out = append(out, t)
		}
	}
	return out
}

func (m *Model) clampCursor() {
	n := len(m.items())
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// sortTasks orders tasks by day, then unfinished before done, then
// priority, then the manual sort order.
func sortTasks(tasks []store.Task) []store.Task {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.Day() != b.Day() {
			return a.Day() < b.Day()
		}
		if a.Done() != b.Done() {
			return !a.Done()
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.SortOrder < b.SortOrder
	})
	return tasks
}