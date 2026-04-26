package ticktick

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_GetProjects_Success(t *testing.T) {
	projects := []Project{
		{ID: "aabbccddeeff00112233aa01", Name: "Inbox"},
		{ID: "aabbccddeeff00112233aa02", Name: "Work"},
	}
	data, _ := json.Marshal(projects)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	got, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2", len(got))
	}
	if got[0].Name != "Inbox" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Inbox")
	}
	if got[1].Name != "Work" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "Work")
	}
}

func TestClient_GetTask_Success(t *testing.T) {
	task := Task{
		ID:        "aabbccddeeff00112233aa01",
		ProjectID: "112233aabbccddeeff001122",
		Title:     "Test Task",
		Priority:  3,
	}
	data, _ := json.Marshal(task)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/project/112233aabbccddeeff001122/task/aabbccddeeff00112233aa01"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	got, err := client.GetTask(context.Background(), "112233aabbccddeeff001122", "aabbccddeeff00112233aa01")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Title != "Test Task" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Task")
	}
	if got.Priority != 3 {
		t.Errorf("Priority = %d, want 3", got.Priority)
	}
}

func TestClient_CreateTask_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/task" {
			t.Errorf("path = %q, want /task", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		body, _ := io.ReadAll(r.Body)
		var task Task
		if err := json.Unmarshal(body, &task); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if task.Title != "New Task" {
			t.Errorf("request body title = %q, want %q", task.Title, "New Task")
		}

		resp := Task{ID: "aabbccddeeff00112233aa99", ProjectID: "112233aabbccddeeff001122", Title: "New Task"}
		data, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	created, err := client.CreateTask(context.Background(), &Task{Title: "New Task"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if created.ID != "aabbccddeeff00112233aa99" {
		t.Errorf("ID = %q, want %q", created.ID, "aabbccddeeff00112233aa99")
	}
}

func TestClient_Retry_On5xx(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Internal Server Error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"aabbccddeeff00112233aa01","name":"Inbox"}]`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	projects, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects() error = %v, expected success after retries", err)
	}
	if len(projects) != 1 {
		t.Errorf("got %d projects, want 1", len(projects))
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestClient_NoRetry_NonIdempotentWrite(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal Server Error"}`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	_, err := client.CreateTask(context.Background(), &Task{Title: "Test"})
	if err == nil {
		t.Fatal("CreateTask() expected error on 500")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retries for non-idempotent writes)", got)
	}
}

func TestClient_401_ReturnsErrUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	_, err := client.GetProjects(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", err)
	}
}

func TestClient_429_RespectsRetryAfter(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Too Many Requests"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"aabbccddeeff00112233aa01","name":"Inbox"}]`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	projects, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects() error = %v, expected success after retry", err)
	}
	if len(projects) != 1 {
		t.Errorf("got %d projects, want 1", len(projects))
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "test")
	_, err := client.GetProjects(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error = %v, want context canceled", err)
	}
}

func TestClient_UserAgent(t *testing.T) {
	var gotUA string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(&http.Client{}, ts.URL, "1.2.3")
	_, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects() error = %v", err)
	}

	want := "ticktick-mcp-go/1.2.3"
	if gotUA != want {
		t.Errorf("User-Agent = %q, want %q", gotUA, want)
	}
}
