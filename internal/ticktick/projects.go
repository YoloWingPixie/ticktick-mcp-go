package ticktick

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	data, err := c.get(ctx, "/project")
	if err != nil {
		return nil, fmt.Errorf("get projects: %w", err)
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("get projects: decode: %w", err)
	}
	return projects, nil
}

func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	data, err := c.get(ctx, "/project/"+url.PathEscape(projectID))
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("get project: decode: %w", err)
	}
	return &project, nil
}

func (c *Client) GetProjectData(ctx context.Context, projectID string) (*ProjectData, error) {
	data, err := c.get(ctx, "/project/"+url.PathEscape(projectID)+"/data")
	if err != nil {
		return nil, fmt.Errorf("get project data: %w", err)
	}

	var pd ProjectData
	if err := json.Unmarshal(data, &pd); err != nil {
		return nil, fmt.Errorf("get project data: decode: %w", err)
	}
	return &pd, nil
}

func (c *Client) CreateProject(ctx context.Context, project *Project) (*Project, error) {
	data, err := c.post(ctx, "/project", project)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	var created Project
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("create project: decode: %w", err)
	}
	return &created, nil
}

func (c *Client) UpdateProject(ctx context.Context, project *Project) (*Project, error) {
	data, err := c.postIdempotent(ctx, "/project/"+url.PathEscape(project.ID), project)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	var updated Project
	if err := json.Unmarshal(data, &updated); err != nil {
		return nil, fmt.Errorf("update project: decode: %w", err)
	}
	return &updated, nil
}

func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	if err := c.del(ctx, "/project/"+url.PathEscape(projectID)); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
