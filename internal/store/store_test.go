package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testUser = "00000000-0000-0000-0000-000000000001"

func TestTasksFiltersByUserAndSendsKey(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Write([]byte(`[{"id":"a","title":"Write report","status":"todo","priority":2,
			"tags":[],"planned_for":"2026-10-07","sort_order":0,
			"created_at":"2026-10-01T02:00:00.123456+00:00","updated_at":"2026-10-01T02:00:00+00:00"}]`))
	}))
	defer srv.Close()

	s := New(Config{URL: srv.URL, Key: "eyJtest", UserID: testUser})
	tasks, err := s.Tasks(context.Background(), "2026-10-06", "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}

	if got.URL.Path != "/rest/v1/tasks" {
		t.Errorf("path = %s", got.URL.Path)
	}
	if v := got.URL.Query().Get("user_id"); v != "eq."+testUser {
		t.Errorf("user_id filter = %q", v)
	}
	if got.Header.Get("apikey") != "eyJtest" || got.Header.Get("Authorization") != "Bearer eyJtest" {
		t.Errorf("missing auth headers")
	}
	if len(tasks) != 1 || tasks[0].Title != "Write report" || tasks[0].Day() != "2026-10-07" {
		t.Errorf("tasks = %+v", tasks)
	}
}

func TestSecretKeyNotSentAsBearer(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	s := New(Config{URL: srv.URL, Key: "sb_secret_test", UserID: testUser})
	if _, err := s.Overdue(context.Background(), "2026-10-07"); err != nil {
		t.Fatal(err)
	}
	if got.Header.Get("Authorization") != "" {
		t.Errorf("sb_secret key must not be sent as a Bearer token")
	}
}

func TestErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Invalid API key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	s := New(Config{URL: srv.URL, Key: "wrong", UserID: testUser})
	_, err := s.Tasks(context.Background(), "2026-10-07", "2026-10-07")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want a 401 error", err)
	}
}

func TestBadDateRejected(t *testing.T) {
	s := New(Config{URL: "http://127.0.0.1:1", Key: "k", UserID: testUser})
	if _, err := s.Tasks(context.Background(), "2026-10-07,user_id.neq.x", "2026-10-07"); err == nil {
		t.Error("expected an error for a bad date")
	}
}

func TestConfigFromEnv(t *testing.T) {
	cases := []struct {
		name, url, user string
		ok              bool
	}{
		{"good", "https://abc.supabase.co/", testUser, true},
		{"local http ok", "http://127.0.0.1:54321", testUser, true},
		{"remote http rejected", "http://abc.supabase.co", testUser, false},
		{"bad user id", "https://abc.supabase.co", "me", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TASKD_SUPABASE_URL", c.url)
			t.Setenv("TASKD_SERVICE_ROLE_KEY", "k")
			t.Setenv("TASKD_USER_ID", c.user)
			cfg, err := ConfigFromEnv()
			if (err == nil) != c.ok {
				t.Fatalf("err = %v, want ok=%v", err, c.ok)
			}
			if c.ok && strings.HasSuffix(cfg.URL, "/") {
				t.Errorf("trailing slash not trimmed: %s", cfg.URL)
			}
		})
	}
}