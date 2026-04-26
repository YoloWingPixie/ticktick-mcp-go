package server

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/zachshepherd/ticktick-mcp-go/internal/safety"
	"github.com/zachshepherd/ticktick-mcp-go/internal/ticktick"
)

func (s *Server) handleGetTask(ctx context.Context, _ *mcp.CallToolRequest, input GetTaskInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}
	if err := safety.ValidateTaskID(input.TaskID); err != nil {
		return nil, nil, err
	}

	task, err := s.client.GetTask(ctx, input.ProjectID, input.TaskID)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	return jsonResult(task)
}

func (s *Server) handleGetAllTasks(ctx context.Context, _ *mcp.CallToolRequest, _ GetAllTasksInput) (*mcp.CallToolResult, any, error) {
	tasks, err := s.getAllTasks(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	return jsonResult(tasks)
}

func (s *Server) handleCreateTask(ctx context.Context, _ *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateCreateTask(input.Title, input.ProjectID, input.Content, input.Desc, input.StartDate, input.DueDate, input.Priority); err != nil {
		return nil, nil, err
	}

	created, err := s.client.CreateTask(ctx, input.toTask())
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateTaskCache(created.ProjectID)

	return jsonResult(created)
}

func (s *Server) handleUpdateTask(ctx context.Context, _ *mcp.CallToolRequest, input UpdateTaskInput) (*mcp.CallToolResult, any, error) {
	priority := 0
	if input.Priority != nil {
		priority = *input.Priority
	}
	if err := safety.ValidateUpdateTask(input.TaskID, input.ProjectID, input.Title, input.Content, input.Desc, input.StartDate, input.DueDate, priority); err != nil {
		return nil, nil, err
	}

	// Fetch current task so we only overwrite provided fields
	existing, err := s.client.GetTask(ctx, input.ProjectID, input.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch task for update: %w", sanitizeAPIError(err))
	}

	if input.Title != "" {
		existing.Title = input.Title
	}
	if input.Content != "" {
		existing.Content = input.Content
	}
	if input.Desc != "" {
		existing.Desc = input.Desc
	}
	if input.IsAllDay != nil {
		existing.IsAllDay = *input.IsAllDay
	}
	if input.StartDate != "" {
		existing.StartDate = input.StartDate
	}
	if input.DueDate != "" {
		existing.DueDate = input.DueDate
	}
	if input.TimeZone != "" {
		existing.TimeZone = input.TimeZone
	}
	if input.Priority != nil {
		existing.Priority = *input.Priority
	}
	if input.Reminders != nil {
		existing.Reminders = input.Reminders
	}
	if input.RepeatFlag != "" {
		existing.RepeatFlag = input.RepeatFlag
	}

	updated, err := s.client.UpdateTask(ctx, existing)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateTaskCache(input.ProjectID)

	return jsonResult(updated)
}

func (s *Server) handleCompleteTask(ctx context.Context, _ *mcp.CallToolRequest, input CompleteTaskInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}
	if err := safety.ValidateTaskID(input.TaskID); err != nil {
		return nil, nil, err
	}

	if err := s.client.CompleteTask(ctx, input.ProjectID, input.TaskID); err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateTaskCache(input.ProjectID)

	return textResult(fmt.Sprintf("Task %s completed.", input.TaskID))
}

func (s *Server) handleMoveTask(ctx context.Context, _ *mcp.CallToolRequest, input MoveTaskInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateTaskID(input.TaskID); err != nil {
		return nil, nil, err
	}
	if err := safety.ValidateProjectID(input.FromProject); err != nil {
		return nil, nil, err
	}
	if err := safety.ValidateProjectID(input.ToProject); err != nil {
		return nil, nil, err
	}

	// No move API exists — simulate with get + create + complete
	original, err := s.client.GetTask(ctx, input.FromProject, input.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("get task from source project: %w", sanitizeAPIError(err))
	}

	newTask := &ticktick.Task{
		Title:      original.Title,
		ProjectID:  input.ToProject,
		Content:    original.Content,
		Desc:       original.Desc,
		IsAllDay:   original.IsAllDay,
		StartDate:  original.StartDate,
		DueDate:    original.DueDate,
		TimeZone:   original.TimeZone,
		Priority:   original.Priority,
		Reminders:  original.Reminders,
		RepeatFlag: original.RepeatFlag,
		Items:      original.Items,
	}

	created, err := s.client.CreateTask(ctx, newTask)
	if err != nil {
		return nil, nil, fmt.Errorf("create task in destination project: %w", sanitizeAPIError(err))
	}

	if err := s.client.CompleteTask(ctx, input.FromProject, input.TaskID); err != nil {
		return nil, nil, fmt.Errorf("complete original task (new task %s was created in destination): %w", created.ID, sanitizeAPIError(err))
	}

	s.invalidateTaskCache(input.FromProject)
	s.invalidateTaskCache(input.ToProject)

	return jsonResult(created)
}
