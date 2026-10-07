// Package store reads tasks from Supabase through its REST API (PostgREST).
// It is the only package that knows the URL and the key.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// Config is what the store needs to reach Supabase.
type Config struct {
	URL    string // https://<project-ref>.supabase.co
	Key    string // service role key (v0.9 only, replaced by OTP login later)
	UserID string // every query is filtered to this user
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ConfigFromEnv reads the config from environment variables and checks it.
func ConfigFromEnv() (Config, error) {
	c := Config{
		URL:    strings.TrimRight(os.Getenv("TASKD_SUPABASE_URL"), "/"),
		Key:    os.Getenv("TASKD_SERVICE_ROLE_KEY"),
		UserID: os.Getenv("TASKD_USER_ID"),
	}

	var missing []string
	if c.URL == "" {
		missing = append(missing, "TASKD_SUPABASE_URL")
	}
	if c.Key == "" {
		missing = append(missing, "TASKD_SERVICE_ROLE_KEY")
	}
	if c.UserID == "" {
		missing = append(missing, "TASKD_USER_ID")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}

	u, err := url.Parse(c.URL)
	if err != nil || u.Host == "" {
		return Config{}, fmt.Errorf("TASKD_SUPABASE_URL is not a valid URL")
	}
	local := u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"
	if u.Scheme != "https" && !local {
		return Config{}, errors.New("TASKD_SUPABASE_URL must use https (the key travels in every request)")
	}

	// The user ID goes into the query string, so only accept a real UUID.
	if !uuidRe.MatchString(c.UserID) {
		return Config{}, errors.New("TASKD_USER_ID is not a UUID")
	}
	return c, nil
}

// Store talks to Supabase.
type Store struct {
	cfg  Config
	http *http.Client
}

// New returns a Store for cfg.
func New(cfg Config) *Store {
	return &Store{cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}}
}

// Tasks returns the todo, doing and done tasks whose day (planned_for, or
// due_date when planned_for is empty) falls between from and to, inclusive.
// Dates are YYYY-MM-DD.
func (s *Store) Tasks(ctx context.Context, from, to string) ([]Task, error) {
	if err := checkDates(from, to); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("status", "in.(todo,doing,done)")
	q.Set("or", fmt.Sprintf(
		"(and(planned_for.gte.%s,planned_for.lte.%s),and(planned_for.is.null,due_date.gte.%s,due_date.lte.%s))",
		from, to, from, to))
	return s.get(ctx, q)
}

// Overdue returns unfinished tasks whose day is before the given date.
func (s *Store) Overdue(ctx context.Context, before string) ([]Task, error) {
	if err := checkDates(before); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("status", "in.(todo,doing)")
	q.Set("or", fmt.Sprintf("(planned_for.lt.%s,and(planned_for.is.null,due_date.lt.%s))", before, before))
	return s.get(ctx, q)
}

// get runs a read on public.tasks. The user filter is added here, last, so
// no caller can forget it or override it.
func (s *Store) get(ctx context.Context, q url.Values) ([]Task, error) {
	q.Set("select", "*")
	q.Set("order", "priority.asc,sort_order.asc")
	q.Set("user_id", "eq."+s.cfg.UserID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.URL+"/rest/v1/tasks?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", s.cfg.Key)
	// Legacy keys are JWTs (they start with "eyJ") and also go in the
	// Authorization header. New sb_secret_ keys are not JWTs and must not.
	if strings.HasPrefix(s.cfg.Key, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Key)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reaching Supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("supabase returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tasks []Task
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}
	return tasks, nil
}

// checkDates makes sure every value is a real YYYY-MM-DD date before it
// goes into a query.
func checkDates(dates ...string) error {
	for _, d := range dates {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return fmt.Errorf("bad date %q", d)
		}
	}
	return nil
}