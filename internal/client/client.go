// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package client provides an HTTP client for the Prisma Postgres Management API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// BaseURL is the base URL for the Prisma Postgres API.
	BaseURL = "https://api.prisma.io"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
)

// Client is an HTTP client for the Prisma Postgres API.
type Client struct {
	httpClient   *http.Client
	serviceToken string
	userAgent    string
	baseURL      string
}

// Config holds configuration for creating a new Client.
type Config struct {
	ServiceToken string
	UserAgent    string
	BaseURL      string
	HTTPClient   *http.Client
}

// NewClient creates a new Prisma API client.
func NewClient(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultTimeout,
		}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "terraform-provider-prisma-postgres/1.0"
	}

	return &Client{
		httpClient:   httpClient,
		serviceToken: cfg.ServiceToken,
		userAgent:    userAgent,
		baseURL:      baseURL,
	}
}

// APIError represents an error response from the Prisma API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Prisma API error (status %d): %s", e.StatusCode, e.Message)
}

// doRequest performs an HTTP request to the Prisma API.
func (c *Client) doRequest(ctx context.Context, method, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ProjectRef is a minimal project reference in API responses.
type ProjectRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WorkspaceRef is a minimal workspace reference in API responses.
type WorkspaceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Project represents a Prisma Postgres project.
type Project struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"` // Always "project"
	Name      string        `json:"name"`
	CreatedAt string        `json:"createdAt"`
	Workspace *WorkspaceRef `json:"workspace,omitempty"`
	Database  *Database     `json:"database,omitempty"` // Only on create when createDatabase=true
}

// CreateProjectRequest is the request body for creating a project.
type CreateProjectRequest struct {
	Name           string `json:"name"`
	CreateDatabase bool   `json:"createDatabase"`
}

// CreateProjectResponse is the response from creating a project.
type CreateProjectResponse struct {
	Data Project `json:"data"`
}

// GetProjectResponse is the response from getting a project.
type GetProjectResponse struct {
	Data Project `json:"data"`
}

// CreateProject creates a new Prisma Postgres project.
func (c *Client) CreateProject(ctx context.Context, name string, createDatabase bool) (*Project, error) {
	req := CreateProjectRequest{
		Name:           name,
		CreateDatabase: createDatabase,
	}

	var resp CreateProjectResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/projects", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// GetProject retrieves a project by ID.
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var resp GetProjectResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/projects/"+id, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/v1/projects/"+id, nil, nil)
}

// DirectConnection represents direct PostgreSQL connection details.
type DirectConnection struct {
	Host string `json:"host"`
	Pass string `json:"pass"`
	User string `json:"user"`
}

// DatabaseRef is a minimal database reference in API responses.
type DatabaseRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// APIKey represents an auto-created connection/API key in create responses.
type APIKey struct {
	ID               string `json:"id"`
	Type             string `json:"type"` // Always "connection"
	Name             string `json:"name"`
	CreatedAt        string `json:"createdAt"`
	ConnectionString string `json:"connectionString"`
}

// Database represents a Prisma Postgres database.
type Database struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"` // Always "database"
	Name             string            `json:"name"`
	Status           string            `json:"status"` // failure, provisioning, ready, recovering
	CreatedAt        string            `json:"createdAt"`
	IsDefault        bool              `json:"isDefault"`
	Project          *ProjectRef       `json:"project,omitempty"` // Returned by GET
	Region           *Region           `json:"region,omitempty"`
	APIKeys          []APIKey          `json:"apiKeys,omitempty"`          // Only on create
	ConnectionString string            `json:"connectionString,omitempty"` // Only on create
	DirectConnection *DirectConnection `json:"directConnection,omitempty"` // Only on create
}

// Region represents a Prisma Postgres region.
type Region struct {
	ID     string `json:"id"`
	Type   string `json:"type,omitempty"` // "region"
	Name   string `json:"name"`
	Status string `json:"status,omitempty"` // "available" or "unavailable"
}

// CreateDatabaseRequest is the request body for creating a database.
type CreateDatabaseRequest struct {
	Name      string `json:"name"`
	Region    string `json:"region,omitempty"`
	IsDefault bool   `json:"isDefault"`
}

// CreateDatabaseResponse is the response from creating a database.
type CreateDatabaseResponse struct {
	Data Database `json:"data"`
}

// GetDatabaseResponse is the response from getting a database.
type GetDatabaseResponse struct {
	Data Database `json:"data"`
}

// CreateDatabase creates a new database in a project.
func (c *Client) CreateDatabase(ctx context.Context, projectID, name, region string) (*Database, error) {
	req := CreateDatabaseRequest{
		Name:      name,
		Region:    region,
		IsDefault: false,
	}

	var resp CreateDatabaseResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/projects/"+projectID+"/databases", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// GetDatabase retrieves a database by ID.
func (c *Client) GetDatabase(ctx context.Context, id string) (*Database, error) {
	var resp GetDatabaseResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/databases/"+id, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// DeleteDatabase deletes a database by ID.
func (c *Client) DeleteDatabase(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/v1/databases/"+id, nil, nil)
}

// Connection represents a Prisma Postgres database connection/API key.
type Connection struct {
	ID               string       `json:"id"`
	Type             string       `json:"type"` // Always "connection"
	Name             string       `json:"name"`
	CreatedAt        string       `json:"createdAt"`
	Database         *DatabaseRef `json:"database,omitempty"`         // Reference to parent database
	ConnectionString string       `json:"connectionString,omitempty"` // Only on create
	Host             string       `json:"host,omitempty"`             // Only on create
	User             string       `json:"user,omitempty"`             // Only on create
	Pass             string       `json:"pass,omitempty"`             // Only on create
}

// Pagination represents pagination info in list responses.
type Pagination struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// CreateConnectionRequest is the request body for creating a connection.
type CreateConnectionRequest struct {
	Name string `json:"name"`
}

// CreateConnectionResponse is the response from creating a connection.
type CreateConnectionResponse struct {
	Data Connection `json:"data"`
}

// ListConnectionsResponse is the response from listing connections.
type ListConnectionsResponse struct {
	Data       []Connection `json:"data"`
	Pagination *Pagination  `json:"pagination,omitempty"`
}

// CreateConnection creates a new connection for a database.
func (c *Client) CreateConnection(ctx context.Context, databaseID, name string) (*Connection, error) {
	req := CreateConnectionRequest{
		Name: name,
	}

	var resp CreateConnectionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/databases/"+databaseID+"/connections", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// ListConnections lists all connections for a database.
func (c *Client) ListConnections(ctx context.Context, databaseID string) ([]Connection, error) {
	var resp ListConnectionsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/databases/"+databaseID+"/connections", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// DeleteConnection deletes a connection by ID.
func (c *Client) DeleteConnection(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/v1/connections/"+id, nil, nil)
}

// ListRegionsResponse is the response from listing regions.
type ListRegionsResponse struct {
	Data []Region `json:"data"`
}

// ListRegions lists all available Postgres regions.
func (c *Client) ListRegions(ctx context.Context) ([]Region, error) {
	var resp ListRegionsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/regions/postgres", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}
