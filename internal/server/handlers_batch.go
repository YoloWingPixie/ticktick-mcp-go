package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/zachshepherd/ticktick-mcp-go/internal/safety"
	"github.com/zachshepherd/ticktick-mcp-go/internal/ticktick"
)

const (
	itemStatusSuccess = "success"
	itemStatusFailed  = "failed"
	itemStatusSkipped = "skipped"
)

func sanitizeAPIError(err error) error {
	return fmt.Errorf("%s", sanitizeError(err))
}

func sanitizeError(err error) string {
	var apiErr *ticktick.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			return "unauthorized"
		case http.StatusForbidden:
			return "forbidden"
		case http.StatusNotFound:
			return "not found"
		case http.StatusTooManyRequests:
			return "rate limited"
		default:
			return fmt.Sprintf("API error (%d)", apiErr.StatusCode)
		}
	}
	if errors.Is(err, context.Canceled) {
		return "request canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timed out"
	}
	return "request failed"
}

func (s *Server) handleBatchCreateTasks(ctx context.Context, _ *mcp.CallToolRequest, input BatchCreateTasksInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateBatchSize(len(input.Tasks)); err != nil {
		return nil, nil, err
	}

	for i, t := range input.Tasks {
		if err := safety.ValidateCreateTask(t.Title, t.ProjectID, t.Content, t.Desc, t.StartDate, t.DueDate, t.Priority); err != nil {
			return nil, nil, fmt.Errorf("item %d: %w", i, err)
		}
	}

	results := make([]ItemResult, len(input.Tasks))
	affectedProjects := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i, t := range input.Tasks {
		wg.Add(1)
		go func(idx int, taskInput CreateTaskInput) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				results[idx] = ItemResult{Index: idx, Status: itemStatusSkipped, Error: "context canceled"}
				return
			}

			created, err := s.client.CreateTask(ctx, taskInput.toTask())
			if err != nil {
				results[idx] = ItemResult{Index: idx, Status: itemStatusFailed, Error: sanitizeError(err)}
				return
			}

			mu.Lock()
			affectedProjects[created.ProjectID] = struct{}{}
			mu.Unlock()
			results[idx] = ItemResult{Index: idx, Status: itemStatusSuccess, ID: created.ID}
		}(i, t)
	}
	wg.Wait()

	for pid := range affectedProjects {
		s.invalidateTaskCache(pid)
	}

	br := BatchResult{Requested: len(input.Tasks), Results: results}
	for _, r := range results {
		if r.Status == itemStatusSuccess {
			br.Succeeded++
		} else {
			br.Failed++
		}
	}

	return jsonResult(br)
}

func (s *Server) handleBatchCompleteTasks(ctx context.Context, _ *mcp.CallToolRequest, input BatchCompleteTasksInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateBatchSize(len(input.Tasks)); err != nil {
		return nil, nil, err
	}

	for i, t := range input.Tasks {
		if err := safety.ValidateProjectID(t.ProjectID); err != nil {
			return nil, nil, fmt.Errorf("item %d: %w", i, err)
		}
		if err := safety.ValidateTaskID(t.TaskID); err != nil {
			return nil, nil, fmt.Errorf("item %d: %w", i, err)
		}
	}

	results := make([]ItemResult, len(input.Tasks))
	affectedProjects := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i, t := range input.Tasks {
		wg.Add(1)
		go func(idx int, ci CompleteTaskInput) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				results[idx] = ItemResult{Index: idx, Status: itemStatusSkipped, Error: "context canceled"}
				return
			}

			err := s.client.CompleteTask(ctx, ci.ProjectID, ci.TaskID)
			if err != nil {
				results[idx] = ItemResult{Index: idx, Status: itemStatusFailed, Error: sanitizeError(err)}
				return
			}

			mu.Lock()
			affectedProjects[ci.ProjectID] = struct{}{}
			mu.Unlock()
			results[idx] = ItemResult{Index: idx, Status: itemStatusSuccess, ID: ci.TaskID}
		}(i, t)
	}
	wg.Wait()

	for pid := range affectedProjects {
		s.invalidateTaskCache(pid)
	}

	br := BatchResult{Requested: len(input.Tasks), Results: results}
	for _, r := range results {
		if r.Status == itemStatusSuccess {
			br.Succeeded++
		} else {
			br.Failed++
		}
	}

	return jsonResult(br)
}

func (s *Server) handleDeleteTask(ctx context.Context, _ *mcp.CallToolRequest, input DeleteTaskInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirmed {
		return nil, nil, fmt.Errorf("deletion not confirmed: set confirmed=true to permanently delete this task")
	}
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}
	if err := safety.ValidateTaskID(input.TaskID); err != nil {
		return nil, nil, err
	}

	if err := s.client.DeleteTask(ctx, input.ProjectID, input.TaskID); err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateTaskCache(input.ProjectID)

	return textResult(fmt.Sprintf("Task %s permanently deleted.", input.TaskID))
}

func (s *Server) handleDeleteProject(ctx context.Context, _ *mcp.CallToolRequest, input DeleteProjectInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirmed {
		return nil, nil, fmt.Errorf("deletion not confirmed: set confirmed=true to permanently delete this project and all its tasks")
	}
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}

	if err := s.client.DeleteProject(ctx, input.ProjectID); err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateProjectCache()
	s.invalidateTaskCache(input.ProjectID)

	return textResult(fmt.Sprintf("Project %s permanently deleted.", input.ProjectID))
}
