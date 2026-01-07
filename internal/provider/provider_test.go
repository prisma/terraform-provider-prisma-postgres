// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// ID counters for generating unique resource IDs.
var (
	projectIDCounter    atomic.Int64
	databaseIDCounter   atomic.Int64
	connectionIDCounter atomic.Int64
)

func nextProjectID() string {
	return fmt.Sprintf("proj_test%d", projectIDCounter.Add(1))
}

func nextDatabaseID() string {
	return fmt.Sprintf("db_test%d", databaseIDCounter.Add(1))
}

func nextConnectionID() string {
	return fmt.Sprintf("conn_test%d", connectionIDCounter.Add(1))
}

// mockAPIServer provides a configurable mock HTTP server for testing.
type mockAPIServer struct {
	server   *httptest.Server
	mu       sync.RWMutex
	handlers map[string]http.HandlerFunc

	// State storage for simulating persistent resources.
	projects    map[string]*client.Project
	databases   map[string]*client.Database
	connections map[string]*client.Connection

	// Track last created IDs for dynamic handler registration.
	lastProjectID    string
	lastDatabaseID   string
	lastConnectionID string
}

// newMockAPIServer creates a new mock API server.
func newMockAPIServer() *mockAPIServer {
	m := &mockAPIServer{
		handlers:    make(map[string]http.HandlerFunc),
		projects:    make(map[string]*client.Project),
		databases:   make(map[string]*client.Database),
		connections: make(map[string]*client.Connection),
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path

		m.mu.RLock()
		handler, ok := m.handlers[key]
		m.mu.RUnlock()

		if ok {
			handler(w, r)
			return
		}

		// Default 404 for unhandled routes.
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))

	return m
}

// URL returns the server URL.
func (m *mockAPIServer) URL() string {
	return m.server.URL
}

// Close shuts down the server.
func (m *mockAPIServer) Close() {
	m.server.Close()
}

// Handle registers a handler for a specific method and path.
func (m *mockAPIServer) Handle(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	m.handlers[method+" "+path] = handler
	m.mu.Unlock()
}

// SetupProjectHandlers configures handlers for project CRUD operations.
func (m *mockAPIServer) SetupProjectHandlers() {
	// Generate ID for this test run.
	projectID := nextProjectID()
	m.lastProjectID = projectID

	// Create project.
	m.Handle("POST", "/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		var req client.CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		project := &client.Project{
			ID:        m.lastProjectID,
			Type:      "project",
			Name:      req.Name,
			CreatedAt: "2025-01-07T00:00:00Z",
		}

		m.mu.Lock()
		m.projects[project.ID] = project
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.CreateProjectResponse{Data: *project})
	})

	// Get project.
	m.Handle("GET", "/v1/projects/"+projectID, func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		project, ok := m.projects[m.lastProjectID]
		m.mu.RUnlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.GetProjectResponse{Data: *project})
	})

	// Delete project.
	m.Handle("DELETE", "/v1/projects/"+projectID, func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.projects, m.lastProjectID)
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
}

// SetupDatabaseHandlers configures handlers for database CRUD operations.
func (m *mockAPIServer) SetupDatabaseHandlers() {
	// Generate ID for this test run.
	databaseID := nextDatabaseID()
	m.lastDatabaseID = databaseID

	// Create database.
	m.Handle("POST", "/v1/projects/"+m.lastProjectID+"/databases", func(w http.ResponseWriter, r *http.Request) {
		var req client.CreateDatabaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		region := req.Region
		if region == "" {
			region = "us-east-1"
		}

		database := &client.Database{
			ID:               m.lastDatabaseID,
			Type:             "database",
			Name:             req.Name,
			Status:           "ready",
			CreatedAt:        "2025-01-07T00:00:00Z",
			ConnectionString: "prisma://accelerate.prisma-data.net/?api_key=test_key",
			DirectConnection: &client.DirectConnection{
				Host: region + ".db.prisma-data.net",
				User: "prisma_user",
				Pass: "test_password",
			},
			Region: &client.Region{
				ID:   region,
				Name: "Test Region",
			},
		}

		m.mu.Lock()
		m.databases[database.ID] = database
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.CreateDatabaseResponse{Data: *database})
	})

	// Get database.
	m.Handle("GET", "/v1/databases/"+databaseID, func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		database, ok := m.databases[m.lastDatabaseID]
		project := m.projects[m.lastProjectID]
		m.mu.RUnlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		// Note: GET doesn't return connection string or direct connection details.
		resp := client.Database{
			ID:        database.ID,
			Type:      database.Type,
			Name:      database.Name,
			Status:    database.Status,
			CreatedAt: database.CreatedAt,
			Region:    database.Region,
			Project: &client.ProjectRef{
				ID:   project.ID,
				Name: project.Name,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.GetDatabaseResponse{Data: resp})
	})

	// Delete database.
	m.Handle("DELETE", "/v1/databases/"+databaseID, func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.databases, m.lastDatabaseID)
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
}

// SetupConnectionHandlers configures handlers for connection CRUD operations.
func (m *mockAPIServer) SetupConnectionHandlers() {
	// Generate ID for this test run.
	connectionID := nextConnectionID()
	m.lastConnectionID = connectionID

	// Create connection.
	m.Handle("POST", "/v1/databases/"+m.lastDatabaseID+"/connections", func(w http.ResponseWriter, r *http.Request) {
		var req client.CreateConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		connection := &client.Connection{
			ID:               m.lastConnectionID,
			Type:             "connection",
			Name:             req.Name,
			CreatedAt:        "2025-01-07T00:00:00Z",
			ConnectionString: "prisma://accelerate.prisma-data.net/?api_key=conn_test_key",
			Host:             "accelerate.prisma-data.net",
			User:             "prisma",
			Pass:             "conn_test_password",
		}

		m.mu.Lock()
		m.connections[connection.ID] = connection
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.CreateConnectionResponse{Data: *connection})
	})

	// List connections (used to find connection by ID).
	m.Handle("GET", "/v1/databases/"+m.lastDatabaseID+"/connections", func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		database := m.databases[m.lastDatabaseID]
		var conns []client.Connection
		for _, conn := range m.connections {
			conns = append(conns, client.Connection{
				ID:        conn.ID,
				Type:      conn.Type,
				Name:      conn.Name,
				CreatedAt: conn.CreatedAt,
				Database: &client.DatabaseRef{
					ID:   database.ID,
					Name: database.Name,
				},
			})
		}
		m.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.ListConnectionsResponse{
			Data:       conns,
			Pagination: &client.Pagination{HasMore: false},
		})
	})

	// Delete connection.
	m.Handle("DELETE", "/v1/connections/"+connectionID, func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.connections, m.lastConnectionID)
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
}

// SetupRegionHandlers configures handlers for region data source.
func (m *mockAPIServer) SetupRegionHandlers() {
	m.Handle("GET", "/v1/regions/postgres", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.ListRegionsResponse{
			Data: []client.Region{
				{ID: "us-east-1", Name: "US East (N. Virginia)", Status: "available"},
				{ID: "us-west-1", Name: "US West (N. California)", Status: "available"},
				{ID: "eu-west-3", Name: "Europe (Paris)", Status: "available"},
			},
		})
	})
}

// testProtoV6ProviderFactories returns provider factories for testing.
func testProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"prisma-postgres": providerserver.NewProtocol6WithError(New("test")()),
	}
}
