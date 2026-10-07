package store

import "time"

// Task mirrors one row of public.tasks. Columns that can be NULL are
// pointers, so "no value" (nil) is different from an empty value.
type Task struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Notes           *string    `json:"notes"`
	Status          string     `json:"status"` // todo, doing, done, archived
	Priority        int        `json:"priority"`
	Area            *string    `json:"area"`
	Tags            []string   `json:"tags"`
	PlannedFor      *string    `json:"planned_for"` // YYYY-MM-DD
	DueDate         *string    `json:"due_date"`    // YYYY-MM-DD
	EstimateMinutes *int       `json:"estimate_minutes"`
	SortOrder       int        `json:"sort_order"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Day is the date the task is shown under: planned_for if set, otherwise
// due_date, otherwise "".
func (t Task) Day() string {
	if t.PlannedFor != nil {
		return *t.PlannedFor
	}
	if t.DueDate != nil {
		return *t.DueDate
	}
	return ""
}

// Done reports whether the task is finished.
func (t Task) Done() bool { return t.Status == "done" }