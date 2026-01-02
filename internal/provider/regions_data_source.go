// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &RegionsDataSource{}
	_ datasource.DataSourceWithConfigure = &RegionsDataSource{}
)

// RegionsDataSource defines the data source implementation.
type RegionsDataSource struct {
	client *client.Client
}

// RegionsDataSourceModel describes the data source data model.
type RegionsDataSourceModel struct {
	Regions []RegionModel `tfsdk:"regions"`
}

// RegionModel describes a single region.
type RegionModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

// NewRegionsDataSource creates a new regions data source.
func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

// Metadata returns the data source type name.
func (d *RegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

// Schema defines the schema for the data source.
func (d *RegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available Prisma Postgres regions.",
		MarkdownDescription: `
Lists all available Prisma Postgres regions where databases can be deployed.

## Example Usage

` + "```hcl" + `
data "prisma-postgres_regions" "available" {}

output "regions" {
  value = data.prisma-postgres_regions.available.regions
}

# Use a specific region
resource "prisma-postgres_database" "example" {
  project_id = prisma-postgres_project.example.id
  name       = "production"
  region     = data.prisma-postgres_regions.available.regions[0].id
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Description: "List of available regions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The region identifier (e.g., us-east-1).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The region name.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The region status (available or unavailable).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *RegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading Prisma regions")

	regions, err := d.client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading regions",
			"Could not read regions: "+err.Error(),
		)
		return
	}

	var state RegionsDataSourceModel
	for _, region := range regions {
		state.Regions = append(state.Regions, RegionModel{
			ID:     types.StringValue(region.ID),
			Name:   types.StringValue(region.Name),
			Status: types.StringValue(region.Status),
		})
	}

	tflog.Trace(ctx, "Read Prisma regions", map[string]any{
		"count": len(regions),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
