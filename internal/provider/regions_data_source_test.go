// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestRegionsDataSource tests the regions data source.
func TestRegionsDataSource(t *testing.T) {
	mock := newMockAPIServer()
	defer mock.Close()
	mock.SetupRegionHandlers()

	t.Setenv("PRISMA_SERVICE_TOKEN", "test-token")
	t.Setenv("PRISMA_API_BASE_URL", mock.URL())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testRegionsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prisma-postgres_regions.test", "regions.#", "3"),
					resource.TestCheckResourceAttr("data.prisma-postgres_regions.test", "regions.0.id", "us-east-1"),
				),
			},
		},
	})
}

func testRegionsDataSourceConfig() string {
	return `
data "prisma-postgres_regions" "test" {}
`
}
