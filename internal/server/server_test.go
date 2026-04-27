package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/YoloWingPixie/ticktick-mcp-go/internal/ticktick"
)

// newTestServer creates a minimal Server (without MCP tool registration)
// backed by an httptest server, suitable for testing cache and task fan-out.
func newTestServer(t *testing.T, handler http.Handler, cacheTTL time.Duration) *Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	client := ticktick.NewClientWithBaseURL(&http.Client{}, ts.URL, "test")

	projectCache, err := lru.New[string, cacheEntry[[]ticktick.Project]](64)
	if err != nil {
		t.Fatalf("create project cache: %v", err)
	}
	dataCache, err := lru.New[string, cacheEntry[*ticktick.ProjectData]](256)
	if err != nil {
		t.Fatalf("create data cache: %v", err)
	}

	return &Server{
		client:       client,
		mode:         ModeAllowDestructive,
		cacheTTL:     cacheTTL,
		projectCache: projectCache,
		dataCache:    dataCache,
	}
}

func TestCacheHit(t *testing.T) {
	var calls atomic.Int32
	projects := []ticktick.Project{{ID: "aabbccddeeff00112233aa01", Name: "Inbox"}}
	data, _ := json.Marshal(projects)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	s := newTestServer(t, handler, 5*time.Minute)
	ctx := context.Background()

	// First call
	got1, err := s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("first getCachedProjects() error = %v", err)
	}
	if len(got1) != 1 {
		t.Fatalf("got %d projects, want 1", len(got1))
	}

	// Second call should hit cache
	got2, err := s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("second getCachedProjects() error = %v", err)
	}
	if len(got2) != 1 {
		t.Fatalf("got %d projects, want 1", len(got2))
	}

	if n := calls.Load(); n != 1 {
		t.Errorf("backend called %d times, want 1 (second call should hit cache)", n)
	}
}

func TestCacheExpiry(t *testing.T) {
	var calls atomic.Int32
	projects := []ticktick.Project{{ID: "aabbccddeeff00112233aa01", Name: "Inbox"}}
	data, _ := json.Marshal(projects)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	s := newTestServer(t, handler, 1*time.Millisecond)
	ctx := context.Background()

	// First call
	_, err := s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("first getCachedProjects() error = %v", err)
	}

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Second call should miss cache
	_, err = s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("second getCachedProjects() error = %v", err)
	}

	if n := calls.Load(); n != 2 {
		t.Errorf("backend called %d times, want 2 (cache should have expired)", n)
	}
}

func TestCacheInvalidation(t *testing.T) {
	var calls atomic.Int32
	projects := []ticktick.Project{{ID: "aabbccddeeff00112233aa01", Name: "Inbox"}}
	data, _ := json.Marshal(projects)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	s := newTestServer(t, handler, 5*time.Minute)
	ctx := context.Background()

	// First call
	_, err := s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("first getCachedProjects() error = %v", err)
	}

	// Invalidate
	s.invalidateProjectCache()

	// Call again - should miss cache
	_, err = s.getCachedProjects(ctx)
	if err != nil {
		t.Fatalf("getCachedProjects() after invalidation error = %v", err)
	}

	if n := calls.Load(); n != 2 {
		t.Errorf("backend called %d times, want 2 (invalidation should force re-fetch)", n)
	}
}

func TestGetAllTasks_FanoutAndCollect(t *testing.T) {
	projects := []ticktick.Project{
		{ID: "aabbccddeeff001122330001", Name: "Project1"},
		{ID: "aabbccddeeff001122330002", Name: "Project2"},
		{ID: "aabbccddeeff001122330003", Name: "Project3"},
	}

	projectDataMap := map[string]*ticktick.ProjectData{
		"aabbccddeeff001122330001": {
			Project: projects[0],
			Tasks: []ticktick.Task{
				{ID: "task01", Title: "P1 Task 1"},
				{ID: "task02", Title: "P1 Task 2"},
			},
		},
		"aabbccddeeff001122330002": {
			Project: projects[1],
			Tasks: []ticktick.Task{
				{ID: "task03", Title: "P2 Task 1"},
				{ID: "task04", Title: "P2 Task 2"},
			},
		},
		"aabbccddeeff001122330003": {
			Project: projects[2],
			Tasks: []ticktick.Task{
				{ID: "task05", Title: "P3 Task 1"},
				{ID: "task06", Title: "P3 Task 2"},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/project" {
			data, _ := json.Marshal(projects)
			w.Write(data)
			return
		}

		// Match /project/{id}/data
		for id, pd := range projectDataMap {
			if r.URL.Path == "/project/"+id+"/data" {
				data, _ := json.Marshal(pd)
				w.Write(data)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	})

	s := newTestServer(t, handler, 5*time.Minute)
	ctx := context.Background()

	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		t.Fatalf("getAllTasks() error = %v", err)
	}

	if len(tasks) != 6 {
		t.Errorf("got %d tasks, want 6", len(tasks))
	}

	// Verify all tasks are present (order may vary due to concurrency)
	titles := make(map[string]bool)
	for _, task := range tasks {
		titles[task.Title] = true
	}

	for _, want := range []string{"P1 Task 1", "P1 Task 2", "P2 Task 1", "P2 Task 2", "P3 Task 1", "P3 Task 2"} {
		if !titles[want] {
			t.Errorf("missing task %q", want)
		}
	}
}
