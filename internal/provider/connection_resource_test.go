// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestConnectionResource tests the connection resource lifecycle.
func TestConnectionResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()
	mock.SetupDatabaseHandlers()
	mock.SetupConnectionHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testConnectionResourceConfig(),
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

func testConnectionResourceConfig() string {
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
