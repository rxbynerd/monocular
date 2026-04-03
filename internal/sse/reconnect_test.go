package sse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchSessions(t *testing.T) {
	sessions := []SessionInfo{
		{ID: "ses_01", Slug: "fix-bug", Title: "Fix auth bug", Directory: "/proj"},
		{ID: "ses_02", Slug: "add-tests", Title: "Add tests", Directory: "/proj"},
	}
	data, _ := json.Marshal(sessions)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("expected Accept: application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := FetchSessions(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result))
	}
	if result[0].ID != "ses_01" || result[0].Slug != "fix-bug" {
		t.Errorf("unexpected first session: %+v", result[0])
	}
	if result[1].ID != "ses_02" || result[1].Title != "Add tests" {
		t.Errorf("unexpected second session: %+v", result[1])
	}
}

func TestFetchSessionsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := FetchSessions(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestFetchSessionsEmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := FetchSessions(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d sessions", len(result))
	}
}

func TestFetchSessionsContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // slow server
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := FetchSessions(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
