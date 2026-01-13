// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package provider implements the Prisma Postgres Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &PrismaProvider{}

// PrismaProvider defines the provider implementation.
type PrismaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running unit
	// testing.
	version string
}

// PrismaProviderModel describes the provider data model.
type PrismaProviderModel struct {
	ServiceToken types.String `tfsdk:"service_token"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PrismaProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *PrismaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "prisma-postgres"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *PrismaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Prisma Postgres resources.",
		MarkdownDescription: `
The Prisma provider allows you to manage [Prisma Postgres](https://www.prisma.io/postgres) resources.

## Authentication

The provider requires a Prisma service token for authentication. You can provide it via:

1. The ` + "`service_token`" + ` provider attribute
2. The ` + "`PRISMA_SERVICE_TOKEN`" + ` environment variable

Generate a service token from the [Prisma Console](https://console.prisma.io).
`,
		Attributes: map[string]schema.Attribute{
			"service_token": schema.StringAttribute{
				Description: "Prisma service token for API authentication. " +
					"Can also be set via the PRISMA_SERVICE_TOKEN environment variable.",
				MarkdownDescription: "Prisma service token for API authentication. " +
					"Can also be set via the `PRISMA_SERVICE_TOKEN` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a Prisma API client for data sources and resources.
func (p *PrismaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Prisma provider")

	var config PrismaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceToken := os.Getenv("PRISMA_SERVICE_TOKEN")
	if !config.ServiceToken.IsNull() {
		serviceToken = config.ServiceToken.ValueString()
	}

	if serviceToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("service_token"),
			"Missing Prisma Service Token",
			"The provider cannot create the Prisma API client without a service token. "+
				"Set the service_token value in the provider configuration or use the "+
				"PRISMA_SERVICE_TOKEN environment variable.",
		)
		return
	}

	// Allow overriding the base URL for testing.
	baseURL := os.Getenv("PRISMA_API_BASE_URL")

	apiClient := client.NewClient(client.Config{
		ServiceToken: serviceToken,
		UserAgent:    "terraform-provider-prisma-postgres/" + p.version,
		BaseURL:      baseURL,
	})

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient

	tflog.Info(ctx, "Configured Prisma provider", map[string]any{"version": p.version})
}

// Resources defines the resources implemented in the provider.
func (p *PrismaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewDatabaseResource,
		NewConnectionResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *PrismaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRegionsDataSource,
	}
}
