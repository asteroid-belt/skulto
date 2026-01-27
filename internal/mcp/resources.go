package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// resourcePrefix is the URI scheme for Skulto resources.
const resourcePrefix = "skulto://"

// parseSkillURI extracts the slug from a skulto://skill/{slug} URI.
func parseSkillURI(uri string) (slug string, isMetadata bool, err error) {
	if !strings.HasPrefix(uri, resourcePrefix+"skill/") {
		return "", false, fmt.Errorf("invalid URI scheme: %s", uri)
	}

	path := strings.TrimPrefix(uri, resourcePrefix+"skill/")
	if strings.HasSuffix(path, "/metadata") {
		slug = strings.TrimSuffix(path, "/metadata")
		isMetadata = true
	} else {
		slug = path
		isMetadata = false
	}

	if slug == "" {
		return "", false, fmt.Errorf("empty slug in URI: %s", uri)
	}

	return slug, isMetadata, nil
}

// handleSkillContentResource handles skulto://skill/{slug} resources.
func (s *Server) handleSkillContentResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	slug, _, err := parseSkillURI(req.Params.URI)
	if err != nil {
		return nil, err
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", slug)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/markdown",
			Text:     skill.Content,
		},
	}, nil
}

// handleSkillMetadataResource handles skulto://skill/{slug}/metadata resources.
func (s *Server) handleSkillMetadataResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	slug, _, err := parseSkillURI(req.Params.URI)
	if err != nil {
		return nil, err
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", slug)
	}

	resp := toSkillResponse(skill, false) // Metadata only, no content
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %v", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
