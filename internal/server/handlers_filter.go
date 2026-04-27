package server

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/YoloWingPixie/ticktick-mcp-go/internal/safety"
	"github.com/YoloWingPixie/ticktick-mcp-go/internal/ticktick"
)

func parseTickTickDate(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	// TickTick sometimes uses +0000 instead of +00:00
	return time.Parse("2006-01-02T15:04:05-0700", s)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func isToday(t, now time.Time) bool {
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func isThisWeek(t, now time.Time) bool {
	today := startOfDay(now)
	daysUntilEndOfWeek := 7 - int(today.Weekday())
	endOfWeek := today.AddDate(0, 0, daysUntilEndOfWeek+1)
	target := startOfDay(t.In(now.Location()))
	return !target.Before(today) && target.Before(endOfWeek)
}

func isOverdue(t, now time.Time) bool {
	return startOfDay(t.In(now.Location())).Before(startOfDay(now))
}

func isWithinDays(t, now time.Time, days int) bool {
	today := startOfDay(now)
	cutoff := today.AddDate(0, 0, days+1)
	target := startOfDay(t.In(now.Location()))
	return !target.Before(today) && target.Before(cutoff)
}

func filterTasksByDueDate(tasks []ticktick.Task, now time.Time, pred func(time.Time, time.Time) bool) []ticktick.Task {
	var matched []ticktick.Task
	for _, t := range tasks {
		if t.Status == ticktick.TaskStatusCompleted || t.DueDate == "" {
			continue
		}
		due, err := parseTickTickDate(t.DueDate)
		if err != nil {
			continue
		}
		if pred(due, now) {
			matched = append(matched, t)
		}
	}
	return matched
}

func (s *Server) handleGetTasksDueToday(ctx context.Context, _ *mcp.CallToolRequest, _ GetTasksDueTodayInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	return jsonResult(filterTasksByDueDate(tasks, time.Now(), isToday))
}

func (s *Server) handleGetTasksDueThisWeek(ctx context.Context, _ *mcp.CallToolRequest, _ GetTasksDueThisWeekInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	return jsonResult(filterTasksByDueDate(tasks, time.Now(), isThisWeek))
}

func (s *Server) handleGetOverdueTasks(ctx context.Context, _ *mcp.CallToolRequest, _ GetOverdueTasksInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	return jsonResult(filterTasksByDueDate(tasks, time.Now(), isOverdue))
}

func (s *Server) handleGetTasksByPriority(ctx context.Context, _ *mcp.CallToolRequest, input GetTasksByPriorityInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidatePriority(input.Priority); err != nil {
		return nil, nil, err
	}
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	var matched []ticktick.Task
	for _, t := range tasks {
		if t.Priority == input.Priority {
			matched = append(matched, t)
		}
	}
	return jsonResult(matched)
}

func (s *Server) handleSearchTasks(ctx context.Context, _ *mcp.CallToolRequest, input SearchTasksInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return nil, nil, &safety.ValidationError{Field: "query", Message: "must not be empty"}
	}
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	query := strings.ToLower(input.Query)
	var matched []ticktick.Task
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Title), query) ||
			(t.Content != "" && strings.Contains(strings.ToLower(t.Content), query)) {
			matched = append(matched, t)
		}
	}
	return jsonResult(matched)
}

func (s *Server) handleGetEngagedTasks(ctx context.Context, _ *mcp.CallToolRequest, _ GetEngagedTasksInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	now := time.Now()
	today := startOfDay(now)
	var matched []ticktick.Task
	for _, t := range tasks {
		if t.Status == ticktick.TaskStatusCompleted || t.StartDate == "" {
			continue
		}
		start, err := parseTickTickDate(t.StartDate)
		if err != nil {
			continue
		}
		if !startOfDay(start.In(now.Location())).After(today) {
			matched = append(matched, t)
		}
	}
	return jsonResult(matched)
}

func (s *Server) handleGetNextTasks(ctx context.Context, _ *mcp.CallToolRequest, _ GetNextTasksInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}
	now := time.Now()
	return jsonResult(filterTasksByDueDate(tasks, now, func(t, n time.Time) bool {
		return isWithinDays(t, n, 7)
	}))
}
