// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithConfigure   = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
)

// DatabaseResource defines the resource implementation.
type DatabaseResource struct {
	client *client.Client
}

// DatabaseResourceModel describes the resource data model.
type DatabaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	Name             types.String `tfsdk:"name"`
	Region           types.String `tfsdk:"region"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	ConnectionString types.String `tfsdk:"connection_string"` // Accelerate URL
	DirectURL        types.String `tfsdk:"direct_url"`        // Direct PostgreSQL URL
	DirectHost       types.String `tfsdk:"direct_host"`
	DirectUser       types.String `tfsdk:"direct_user"`
	DirectPassword   types.String `tfsdk:"direct_password"`
}

// NewDatabaseResource creates a new database resource.
func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

// Metadata returns the resource type name.
func (r *DatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the resource.
func (r *DatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Prisma Postgres database within a project.",
		MarkdownDescription: `
Manages a Prisma Postgres database within a project.

Each project can contain multiple databases. Databases are deployed to a specific region.

## Example Usage

` + "```hcl" + `
resource "prisma-postgres_project" "example" {
  name = "my-project"
}

resource "prisma-postgres_database" "example" {
  project_id = prisma-postgres_project.example.id
  name       = "production"
  region     = "us-east-1"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the database.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project this database belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region where the database is deployed (e.g., us-east-1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("us-east-1"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the database.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the database was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_string": schema.StringAttribute{
				Description: "The Prisma Accelerate connection string (prisma+postgres://...).",
				Computed:    true,
				Sensitive:   true,
			},
			"direct_url": schema.StringAttribute{
				Description: "The direct PostgreSQL connection URL (postgresql://user:pass@host:5432/db).",
				Computed:    true,
				Sensitive:   true,
			},
			"direct_host": schema.StringAttribute{
				Description: "The direct PostgreSQL host.",
				Computed:    true,
			},
			"direct_user": schema.StringAttribute{
				Description: "The direct PostgreSQL user.",
				Computed:    true,
			},
			"direct_password": schema.StringAttribute{
				Description: "The direct PostgreSQL password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Prisma database", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"name":       plan.Name.ValueString(),
		"region":     plan.Region.ValueString(),
	})

	database, err := r.client.CreateDatabase(
		ctx,
		plan.ProjectID.ValueString(),
		plan.Name.ValueString(),
		plan.Region.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			"Could not create database, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(database.ID)
	plan.Status = types.StringValue(database.Status)
	plan.CreatedAt = types.StringValue(database.CreatedAt)

	plan.ConnectionString = types.StringValue(database.ConnectionString)
	if database.DirectConnection != nil {
		plan.DirectHost = types.StringValue(database.DirectConnection.Host)
		plan.DirectUser = types.StringValue(database.DirectConnection.User)
		plan.DirectPassword = types.StringValue(database.DirectConnection.Pass)
		if database.DirectConnection.Host != "" {
			directURL := fmt.Sprintf("postgresql://%s:%s@%s:5432/postgres",
				database.DirectConnection.User,
				database.DirectConnection.Pass,
				database.DirectConnection.Host)
			plan.DirectURL = types.StringValue(directURL)
		} else {
			plan.DirectURL = types.StringValue("")
		}
	} else {
		plan.DirectHost = types.StringValue("")
		plan.DirectUser = types.StringValue("")
		plan.DirectPassword = types.StringValue("")
		plan.DirectURL = types.StringValue("")
	}

	if database.Region != nil {
		plan.Region = types.StringValue(database.Region.ID)
	}

	tflog.Trace(ctx, "Created Prisma database", map[string]any{
		"id":   database.ID,
		"name": database.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Prisma database", map[string]any{
		"id": state.ID.ValueString(),
	})

	database, err := r.client.GetDatabase(ctx, state.ID.ValueString())
	if err != nil {
		// Check if resource was deleted outside of Terraform
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Database not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading database",
			"Could not read database ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(database.Name)
	state.Status = types.StringValue(database.Status)
	state.CreatedAt = types.StringValue(database.CreatedAt)

	if database.Project != nil {
		state.ProjectID = types.StringValue(database.Project.ID)
	}

	if database.Region != nil {
		state.Region = types.StringValue(database.Region.ID)
	}

	// Credentials are only returned on create, not on GET - preserved in state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Prisma databases cannot be updated. Changes to attributes require replacing the resource.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Prisma database", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteDatabase(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Database already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting database",
			"Could not delete database ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

// ImportState imports the resource state.
func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
