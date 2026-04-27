package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/YoloWingPixie/ticktick-mcp-go/internal/safety"
	"github.com/YoloWingPixie/ticktick-mcp-go/internal/ticktick"
)

func (s *Server) handleGetProjects(ctx context.Context, _ *mcp.CallToolRequest, _ GetProjectsInput) (*mcp.CallToolResult, any, error) {
	projects, err := s.getCachedProjects(ctx)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	return jsonResult(projects)
}

func (s *Server) handleGetProject(ctx context.Context, _ *mcp.CallToolRequest, input GetProjectInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}

	project, err := s.client.GetProject(ctx, input.ProjectID)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	return jsonResult(project)
}

func (s *Server) handleGetProjectWithData(ctx context.Context, _ *mcp.CallToolRequest, input GetProjectWithDataInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}

	pd, err := s.getCachedProjectData(ctx, input.ProjectID)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	return jsonResult(pd)
}

func (s *Server) handleCreateProject(ctx context.Context, _ *mcp.CallToolRequest, input CreateProjectInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateCreateProject(input.Name); err != nil {
		return nil, nil, err
	}

	project := &ticktick.Project{
		Name:     input.Name,
		Color:    input.Color,
		ViewMode: input.ViewMode,
		Kind:     input.Kind,
	}

	created, err := s.client.CreateProject(ctx, project)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateProjectCache()

	return jsonResult(created)
}

func (s *Server) handleUpdateProject(ctx context.Context, _ *mcp.CallToolRequest, input UpdateProjectInput) (*mcp.CallToolResult, any, error) {
	if err := safety.ValidateProjectID(input.ProjectID); err != nil {
		return nil, nil, err
	}

	existing, err := s.client.GetProject(ctx, input.ProjectID)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	if input.Name != "" {
		if err := safety.ValidateProjectName(input.Name); err != nil {
			return nil, nil, err
		}
		existing.Name = input.Name
	}
	if input.Color != "" {
		existing.Color = input.Color
	}
	if input.ViewMode != "" {
		existing.ViewMode = input.ViewMode
	}

	updated, err := s.client.UpdateProject(ctx, existing)
	if err != nil {
		return nil, nil, sanitizeAPIError(err)
	}

	s.invalidateProjectCache()

	return jsonResult(updated)
}
