// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient creates a client configured to use the test server.
func newTestClient(server *httptest.Server) *Client {
	return NewClient(Config{
		ServiceToken: "test-token",
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})
}

// TestNewClient verifies client creation with various configurations.
func TestNewClient(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		c := NewClient(Config{
			ServiceToken: "token123",
		})

		if c.serviceToken != "token123" {
			t.Errorf("expected serviceToken 'token123', got %q", c.serviceToken)
		}
		if c.baseURL != BaseURL {
			t.Errorf("expected baseURL %q, got %q", BaseURL, c.baseURL)
		}
		if c.httpClient == nil {
			t.Error("expected httpClient to be set")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		customClient := &http.Client{}
		c := NewClient(Config{
			ServiceToken: "custom-token",
			BaseURL:      "https://custom.api.com",
			UserAgent:    "custom-agent/1.0",
			HTTPClient:   customClient,
		})

		if c.serviceToken != "custom-token" {
			t.Errorf("expected serviceToken 'custom-token', got %q", c.serviceToken)
		}
		if c.baseURL != "https://custom.api.com" {
			t.Errorf("expected baseURL 'https://custom.api.com', got %q", c.baseURL)
		}
		if c.userAgent != "custom-agent/1.0" {
			t.Errorf("expected userAgent 'custom-agent/1.0', got %q", c.userAgent)
		}
	})
}

// TestAPIError verifies error message formatting.
func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Not Found",
		Body:       `{"error": "resource not found"}`,
	}

	expected := "Prisma API error (status 404): Not Found"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

// TestCreateProject verifies project creation.
func TestCreateProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/projects" {
				t.Errorf("expected /v1/projects, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
			}

			var req CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req.Name != "test-project" {
				t.Errorf("expected name 'test-project', got %q", req.Name)
			}
			if req.CreateDatabase != false {
				t.Errorf("expected createDatabase false, got %v", req.CreateDatabase)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(CreateProjectResponse{
				Data: Project{
					ID:        "proj_123",
					Type:      "project",
					Name:      "test-project",
					CreatedAt: "2025-01-07T00:00:00Z",
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		project, err := client.CreateProject(context.Background(), "test-project", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if project.ID != "proj_123" {
			t.Errorf("expected ID 'proj_123', got %q", project.ID)
		}
		if project.Name != "test-project" {
			t.Errorf("expected Name 'test-project', got %q", project.Name)
		}
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "invalid request"}`))
		}))
		defer server.Close()

		client := newTestClient(server)
		_, err := client.CreateProject(context.Background(), "", false)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != 400 {
			t.Errorf("expected status 400, got %d", apiErr.StatusCode)
		}
	})
}

// TestGetProject verifies project retrieval.
func TestGetProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/v1/projects/proj_123" {
				t.Errorf("expected /v1/projects/proj_123, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GetProjectResponse{
				Data: Project{
					ID:        "proj_123",
					Type:      "project",
					Name:      "test-project",
					CreatedAt: "2025-01-07T00:00:00Z",
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		project, err := client.GetProject(context.Background(), "proj_123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if project.ID != "proj_123" {
			t.Errorf("expected ID 'proj_123', got %q", project.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "project not found"}`))
		}))
		defer server.Close()

		client := newTestClient(server)
		_, err := client.GetProject(context.Background(), "nonexistent")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status 404, got %d", apiErr.StatusCode)
		}
	})
}

// TestDeleteProject verifies project deletion.
func TestDeleteProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/v1/projects/proj_123" {
				t.Errorf("expected /v1/projects/proj_123, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := newTestClient(server)
		err := client.DeleteProject(context.Background(), "proj_123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestCreateDatabase verifies database creation.
func TestCreateDatabase(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/projects/proj_123/databases" {
				t.Errorf("expected /v1/projects/proj_123/databases, got %s", r.URL.Path)
			}

			var req CreateDatabaseRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req.Name != "production" {
				t.Errorf("expected name 'production', got %q", req.Name)
			}
			if req.Region != "us-east-1" {
				t.Errorf("expected region 'us-east-1', got %q", req.Region)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(CreateDatabaseResponse{
				Data: Database{
					ID:               "db_456",
					Type:             "database",
					Name:             "production",
					Status:           "ready",
					CreatedAt:        "2025-01-07T00:00:00Z",
					ConnectionString: "prisma://accelerate.prisma-data.net/?api_key=xxx",
					DirectConnection: &DirectConnection{
						Host: "us-east-1.db.prisma-data.net",
						User: "prisma_user",
						Pass: "secret_password",
					},
					Region: &Region{
						ID:   "us-east-1",
						Name: "US East (N. Virginia)",
					},
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		db, err := client.CreateDatabase(context.Background(), "proj_123", "production", "us-east-1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if db.ID != "db_456" {
			t.Errorf("expected ID 'db_456', got %q", db.ID)
		}
		if db.Status != "ready" {
			t.Errorf("expected Status 'ready', got %q", db.Status)
		}
		if db.DirectConnection == nil {
			t.Fatal("expected DirectConnection to be set")
		}
		if db.DirectConnection.Host != "us-east-1.db.prisma-data.net" {
			t.Errorf("expected Host 'us-east-1.db.prisma-data.net', got %q", db.DirectConnection.Host)
		}
	})
}

// TestGetDatabase verifies database retrieval.
func TestGetDatabase(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/v1/databases/db_456" {
				t.Errorf("expected /v1/databases/db_456, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GetDatabaseResponse{
				Data: Database{
					ID:        "db_456",
					Type:      "database",
					Name:      "production",
					Status:    "ready",
					CreatedAt: "2025-01-07T00:00:00Z",
					Project: &ProjectRef{
						ID:   "proj_123",
						Name: "test-project",
					},
					Region: &Region{
						ID:   "us-east-1",
						Name: "US East (N. Virginia)",
					},
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		db, err := client.GetDatabase(context.Background(), "db_456")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if db.ID != "db_456" {
			t.Errorf("expected ID 'db_456', got %q", db.ID)
		}
		if db.Project == nil {
			t.Fatal("expected Project to be set")
		}
		if db.Project.ID != "proj_123" {
			t.Errorf("expected Project.ID 'proj_123', got %q", db.Project.ID)
		}
	})
}

// TestDeleteDatabase verifies database deletion.
func TestDeleteDatabase(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/v1/databases/db_456" {
				t.Errorf("expected /v1/databases/db_456, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := newTestClient(server)
		err := client.DeleteDatabase(context.Background(), "db_456")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestCreateConnection verifies connection creation.
func TestCreateConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/databases/db_456/connections" {
				t.Errorf("expected /v1/databases/db_456/connections, got %s", r.URL.Path)
			}

			var req CreateConnectionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req.Name != "api-key" {
				t.Errorf("expected name 'api-key', got %q", req.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(CreateConnectionResponse{
				Data: Connection{
					ID:               "conn_789",
					Type:             "connection",
					Name:             "api-key",
					CreatedAt:        "2025-01-07T00:00:00Z",
					ConnectionString: "prisma://accelerate.prisma-data.net/?api_key=yyy",
					Host:             "accelerate.prisma-data.net",
					User:             "prisma",
					Pass:             "api_secret",
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		conn, err := client.CreateConnection(context.Background(), "db_456", "api-key")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if conn.ID != "conn_789" {
			t.Errorf("expected ID 'conn_789', got %q", conn.ID)
		}
		if conn.ConnectionString == "" {
			t.Error("expected ConnectionString to be set")
		}
	})
}

// TestListConnections verifies listing connections.
func TestListConnections(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/v1/databases/db_456/connections" {
				t.Errorf("expected /v1/databases/db_456/connections, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ListConnectionsResponse{
				Data: []Connection{
					{
						ID:        "conn_789",
						Type:      "connection",
						Name:      "api-key-1",
						CreatedAt: "2025-01-07T00:00:00Z",
					},
					{
						ID:        "conn_790",
						Type:      "connection",
						Name:      "api-key-2",
						CreatedAt: "2025-01-07T01:00:00Z",
					},
				},
				Pagination: &Pagination{
					HasMore: false,
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		conns, err := client.ListConnections(context.Background(), "db_456")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(conns) != 2 {
			t.Errorf("expected 2 connections, got %d", len(conns))
		}
		if conns[0].ID != "conn_789" {
			t.Errorf("expected first connection ID 'conn_789', got %q", conns[0].ID)
		}
	})
}

// TestDeleteConnection verifies connection deletion.
func TestDeleteConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/v1/connections/conn_789" {
				t.Errorf("expected /v1/connections/conn_789, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := newTestClient(server)
		err := client.DeleteConnection(context.Background(), "conn_789")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestListRegions verifies listing regions.
func TestListRegions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/v1/regions/postgres" {
				t.Errorf("expected /v1/regions/postgres, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ListRegionsResponse{
				Data: []Region{
					{ID: "us-east-1", Name: "US East (N. Virginia)", Status: "available"},
					{ID: "us-west-1", Name: "US West (N. California)", Status: "available"},
					{ID: "eu-west-3", Name: "Europe (Paris)", Status: "available"},
				},
			})
		}))
		defer server.Close()

		client := newTestClient(server)
		regions, err := client.ListRegions(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 3 {
			t.Errorf("expected 3 regions, got %d", len(regions))
		}
		if regions[0].ID != "us-east-1" {
			t.Errorf("expected first region ID 'us-east-1', got %q", regions[0].ID)
		}
	})
}

// TestInvalidJSONResponse verifies handling of malformed API responses.
func TestInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetProject(context.Background(), "proj_123")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("expected decode error, got: %v", err)
	}
}

// TestRequestHeaders verifies all required headers are set.
func TestRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all expected headers.
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", ct)
		}
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("expected Accept 'application/json', got %q", accept)
		}
		if ua := r.Header.Get("User-Agent"); ua != "test-agent/1.0" {
			t.Errorf("expected User-Agent 'test-agent/1.0', got %q", ua)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRegionsResponse{Data: []Region{}})
	}))
	defer server.Close()

	client := NewClient(Config{
		ServiceToken: "test-token",
		UserAgent:    "test-agent/1.0",
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})

	_, err := client.ListRegions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
