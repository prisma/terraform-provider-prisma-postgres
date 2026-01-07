// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestProjectResource tests the project resource lifecycle.
func TestProjectResource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupProjectHandlers()

	// Set environment for the test.
	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testProjectResourceConfig("test-project"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prisma-postgres_project.test", "name", "test-project"),
					resource.TestCheckResourceAttrSet("prisma-postgres_project.test", "id"),
					resource.TestCheckResourceAttrSet("prisma-postgres_project.test", "created_at"),
				),
			},
		},
	})
}

func testProjectResourceConfig(name string) string {
	return `
resource "prisma-postgres_project" "test" {
  name = "` + name + `"
}
`
}
