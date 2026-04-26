package ticktick

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) GetTask(ctx context.Context, projectID, taskID string) (*Task, error) {
	path := "/project/" + url.PathEscape(projectID) + "/task/" + url.PathEscape(taskID)
	data, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("get task: decode: %w", err)
	}
	return &task, nil
}

func (c *Client) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	data, err := c.post(ctx, "/task", task)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	var created Task
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("create task: decode: %w", err)
	}
	return &created, nil
}

func (c *Client) UpdateTask(ctx context.Context, task *Task) (*Task, error) {
	data, err := c.postIdempotent(ctx, "/task/"+url.PathEscape(task.ID), task)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	var updated Task
	if err := json.Unmarshal(data, &updated); err != nil {
		return nil, fmt.Errorf("update task: decode: %w", err)
	}
	return &updated, nil
}

func (c *Client) CompleteTask(ctx context.Context, projectID, taskID string) error {
	path := "/project/" + url.PathEscape(projectID) + "/task/" + url.PathEscape(taskID) + "/complete"
	_, err := c.postIdempotent(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	return nil
}

func (c *Client) DeleteTask(ctx context.Context, projectID, taskID string) error {
	path := "/project/" + url.PathEscape(projectID) + "/task/" + url.PathEscape(taskID)
	if err := c.del(ctx, path); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}
