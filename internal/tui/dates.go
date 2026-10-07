package tui

import (
	"time"
	_ "time/tzdata" // bundle time zone data: Windows doesn't ship it for Go
)

// dateLayout is how Postgres sends a date column: YYYY-MM-DD.
const dateLayout = "2006-01-02"

// kl is the time zone taskd uses everywhere, same as the MCP server.
var kl = mustLoad("Asia/Kuala_Lumpur")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// view is which period the screen shows.
type view int

const (
	dayView view = iota
	weekView
	monthView
)

var viewNames = []string{"Day", "Week", "Month"}

// today is midnight today in Malaysia time.
func today() time.Time {
	now := time.Now().In(kl)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, kl)
}

// span returns the first and last day of the period that contains d.
// Weeks start on Monday.
func span(v view, d time.Time) (time.Time, time.Time) {
	switch v {
	case weekView:
		offset := (int(d.Weekday()) + 6) % 7 // Monday = 0 ... Sunday = 6
		start := d.AddDate(0, 0, -offset)
		return start, start.AddDate(0, 0, 6)
	case monthView:
		start := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		return start, start.AddDate(0, 1, -1)
	default:
		return d, d
	}
}

// shift moves d by n periods (days, weeks or months).
func shift(v view, d time.Time, n int) time.Time {
	switch v {
	case weekView:
		return d.AddDate(0, 0, 7*n)
	case monthView:
		// Go from the 1st, so 31 Jan + 1 month is February, not 3 March.
		return time.Date(d.Year(), d.Month()+time.Month(n), 1, 0, 0, 0, 0, d.Location())
	default:
		return d.AddDate(0, 0, n)
	}
}

// pretty turns "2026-10-07" into "Wed 7 Oct".
func pretty(day string) string {
	t, err := time.ParseInLocation(dateLayout, day, kl)
	if err != nil {
		return day
	}
	return t.Format("Mon 2 Jan")
}