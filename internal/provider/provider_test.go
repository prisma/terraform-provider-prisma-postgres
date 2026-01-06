// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// mockAPIServer provides a configurable mock HTTP server for testing.
type mockAPIServer struct {
	server   *httptest.Server
	mu       sync.RWMutex
	handlers map[string]http.HandlerFunc

	// State storage for simulating persistent resources.
	projects    map[string]*client.Project
	databases   map[string]*client.Database
	connections map[string]*client.Connection
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
	// Create project.
	m.Handle("POST", "/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		var req client.CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		project := &client.Project{
			ID:        "proj_test123",
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
	m.Handle("GET", "/v1/projects/proj_test123", func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		project, ok := m.projects["proj_test123"]
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
	m.Handle("DELETE", "/v1/projects/proj_test123", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.projects, "proj_test123")
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
}

// SetupDatabaseHandlers configures handlers for database CRUD operations.
func (m *mockAPIServer) SetupDatabaseHandlers() {
	// Create database.
	m.Handle("POST", "/v1/projects/proj_test123/databases", func(w http.ResponseWriter, r *http.Request) {
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
			ID:               "db_test456",
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
	m.Handle("GET", "/v1/databases/db_test456", func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		database, ok := m.databases["db_test456"]
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
				ID:   "proj_test123",
				Name: "test-project",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.GetDatabaseResponse{Data: resp})
	})

	// Delete database.
	m.Handle("DELETE", "/v1/databases/db_test456", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.databases, "db_test456")
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
}

// SetupConnectionHandlers configures handlers for connection CRUD operations.
func (m *mockAPIServer) SetupConnectionHandlers() {
	// Create connection.
	m.Handle("POST", "/v1/databases/db_test456/connections", func(w http.ResponseWriter, r *http.Request) {
		var req client.CreateConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		connection := &client.Connection{
			ID:               "conn_test789",
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
	m.Handle("GET", "/v1/databases/db_test456/connections", func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		var conns []client.Connection
		for _, conn := range m.connections {
			conns = append(conns, client.Connection{
				ID:        conn.ID,
				Type:      conn.Type,
				Name:      conn.Name,
				CreatedAt: conn.CreatedAt,
				Database: &client.DatabaseRef{
					ID:   "db_test456",
					Name: "test-database",
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
	m.Handle("DELETE", "/v1/connections/conn_test789", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		delete(m.connections, "conn_test789")
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

// testAccProtoV6ProviderFactories returns provider factories for testing.
func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"prisma-postgres": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// TestAccProjectResource tests the project resource lifecycle.
func TestAccProjectResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()

	// Set environment for the test.
	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testAccProjectResourceConfig("test-project"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prisma-postgres_project.test", "name", "test-project"),
					resource.TestCheckResourceAttrSet("prisma-postgres_project.test", "id"),
					resource.TestCheckResourceAttrSet("prisma-postgres_project.test", "created_at"),
				),
			},
		},
	})
}

func testAccProjectResourceConfig(name string) string {
	return `
resource "prisma-postgres_project" "test" {
  name = "` + name + `"
}
`
}

// TestAccRegionsDataSource tests the regions data source.
func TestAccRegionsDataSource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupRegionHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccRegionsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prisma-postgres_regions.test", "regions.#", "3"),
					resource.TestCheckResourceAttr("data.prisma-postgres_regions.test", "regions.0.id", "us-east-1"),
				),
			},
		},
	})
}

func testAccRegionsDataSourceConfig() string {
	return `
data "prisma-postgres_regions" "test" {}
`
}

// TestAccDatabaseResource tests the database resource lifecycle.
func TestAccDatabaseResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()
	mock.SetupDatabaseHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prisma-postgres_database.test", "name", "test-database"),
					resource.TestCheckResourceAttr("prisma-postgres_database.test", "region", "us-east-1"),
					resource.TestCheckResourceAttrSet("prisma-postgres_database.test", "id"),
					resource.TestCheckResourceAttrSet("prisma-postgres_database.test", "status"),
					resource.TestCheckResourceAttrSet("prisma-postgres_database.test", "connection_string"),
					resource.TestCheckResourceAttrSet("prisma-postgres_database.test", "direct_url"),
				),
			},
		},
	})
}

func testAccDatabaseResourceConfig() string {
	return `
resource "prisma-postgres_project" "test" {
  name = "test-project"
}

resource "prisma-postgres_database" "test" {
  project_id = prisma-postgres_project.test.id
  name       = "test-database"
  region     = "us-east-1"
}
`
}

// TestAccConnectionResource tests the connection resource lifecycle.
func TestAccConnectionResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()
	mock.SetupDatabaseHandlers()
	mock.SetupConnectionHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prisma-postgres_connection.test", "name", "test-connection"),
					resource.TestCheckResourceAttrSet("prisma-postgres_connection.test", "id"),
					resource.TestCheckResourceAttrSet("prisma-postgres_connection.test", "connection_string"),
					resource.TestCheckResourceAttrSet("prisma-postgres_connection.test", "host"),
					resource.TestCheckResourceAttrSet("prisma-postgres_connection.test", "user"),
					resource.TestCheckResourceAttrSet("prisma-postgres_connection.test", "password"),
				),
			},
		},
	})
}

func testAccConnectionResourceConfig() string {
	return `
resource "prisma-postgres_project" "test" {
  name = "test-project"
}

resource "prisma-postgres_database" "test" {
  project_id = prisma-postgres_project.test.id
  name       = "test-database"
  region     = "us-east-1"
}

resource "prisma-postgres_connection" "test" {
  database_id = prisma-postgres_database.test.id
  name        = "test-connection"
}
`
}
