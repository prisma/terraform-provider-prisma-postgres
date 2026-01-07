// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestDatabaseResource tests the database resource lifecycle.
func TestDatabaseResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()
	mock.SetupDatabaseHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDatabaseResourceConfig(),
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

func testDatabaseResourceConfig() string {
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
