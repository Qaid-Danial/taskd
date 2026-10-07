package tui

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.ParseInLocation(dateLayout, s, kl)
	return t
}

func TestSpan(t *testing.T) {
	cases := []struct {
		v            view
		in, from, to string
	}{
		{dayView, "2026-10-07", "2026-10-07", "2026-10-07"},
		{weekView, "2026-10-07", "2026-10-05", "2026-10-11"}, // Wednesday
		{weekView, "2026-10-11", "2026-10-05", "2026-10-11"}, // Sunday stays in the same week
		{monthView, "2026-02-14", "2026-02-01", "2026-02-28"},
	}
	for _, c := range cases {
		from, to := span(c.v, day(c.in))
		if from.Format(dateLayout) != c.from || to.Format(dateLayout) != c.to {
			t.Errorf("span(%s, %s) = %s..%s, want %s..%s", viewNames[c.v], c.in,
				from.Format(dateLayout), to.Format(dateLayout), c.from, c.to)
		}
	}
}

func TestShiftMonthFrom31st(t *testing.T) {
	got := shift(monthView, day("2026-01-31"), 1).Format(dateLayout)
	if got != "2026-02-01" {
		t.Errorf("got %s, want 2026-02-01", got)
	}
}