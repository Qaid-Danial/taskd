package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Qaid-Danial/taskd/internal/store"
)

// fakeSource returns fixed tasks, so the TUI can be tested without Supabase.
type fakeSource struct{ tasks []store.Task }

func (f fakeSource) Tasks(ctx context.Context, from, to string) ([]store.Task, error) {
	return f.tasks, nil
}

func (f fakeSource) Overdue(ctx context.Context, before string) ([]store.Task, error) {
	return nil, nil
}

func TestFirstLoadAndHideDone(t *testing.T) {
	src := fakeSource{tasks: []store.Task{
		{ID: "1", Title: "open", Status: "todo", Priority: 3},
		{ID: "2", Title: "finished", Status: "done", Priority: 3},
	}}
	m := New(src)

	// Run the command Init returns and feed its message back, the same
	// way Bubble Tea does.
	msg := m.Init()()
	next, _ := m.Update(msg)
	m = next.(Model)

	if m.loading || len(m.items()) != 2 {
		t.Fatalf("after first load: loading=%v items=%d, want false and 2", m.loading, len(m.items()))
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	m = next.(Model)
	if len(m.items()) != 1 {
		t.Errorf("with done hidden: items=%d, want 1", len(m.items()))
	}
}

func TestStaleLoadIgnored(t *testing.T) {
	m := New(fakeSource{})
	m.reqID = 5
	next, _ := m.Update(loadedMsg{id: 4, tasks: []store.Task{{Title: "old"}}})
	if len(next.(Model).items()) != 0 {
		t.Error("an older load overwrote a newer one")
	}
}