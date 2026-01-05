// Copyright (c) Prisma Data, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/prisma/terraform-provider-prisma-postgres/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ConnectionResource{}
	_ resource.ResourceWithConfigure   = &ConnectionResource{}
	_ resource.ResourceWithImportState = &ConnectionResource{}
)

// ConnectionResource defines the resource implementation.
type ConnectionResource struct {
	client *client.Client
}

// ConnectionResourceModel describes the resource data model.
type ConnectionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	DatabaseID       types.String `tfsdk:"database_id"`
	Name             types.String `tfsdk:"name"`
	CreatedAt        types.String `tfsdk:"created_at"`
	ConnectionString types.String `tfsdk:"connection_string"` // Accelerate URL
	Host             types.String `tfsdk:"host"`
	User             types.String `tfsdk:"user"`
	Password         types.String `tfsdk:"password"`
}

// NewConnectionResource creates a new connection resource.
func NewConnectionResource() resource.Resource {
	return &ConnectionResource{}
}

// Metadata returns the resource type name.
func (r *ConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection"
}

// Schema defines the schema for the resource.
func (r *ConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Prisma Postgres database connection (API key).",
		MarkdownDescription: `
Manages a Prisma Postgres database connection (API key).

Connections provide credentials to access a database. Each connection has its own
credentials and connection strings.

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

resource "prisma-postgres_connection" "api" {
  database_id = prisma-postgres_database.example.id
  name        = "api-key"
}

# Use the connection string in your application
output "database_url" {
  value     = prisma-postgres_connection.api.connection_string
  sensitive = true
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.StringAttribute{
				Description: "The ID of the database this connection belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the connection.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the connection was created.",
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
			"host": schema.StringAttribute{
				Description: "The database host.",
				Computed:    true,
			},
			"user": schema.StringAttribute{
				Description: "The database user.",
				Computed:    true,
			},
			"password": schema.StringAttribute{
				Description: "The database password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Prisma connection", map[string]any{
		"database_id": plan.DatabaseID.ValueString(),
		"name":        plan.Name.ValueString(),
	})

	connection, err := r.client.CreateConnection(
		ctx,
		plan.DatabaseID.ValueString(),
		plan.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating connection",
			"Could not create connection, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(connection.ID)
	plan.CreatedAt = types.StringValue(connection.CreatedAt)

	plan.ConnectionString = types.StringValue(connection.ConnectionString)
	plan.Host = types.StringValue(connection.Host)
	plan.User = types.StringValue(connection.User)
	plan.Password = types.StringValue(connection.Pass)

	tflog.Trace(ctx, "Created Prisma connection", map[string]any{
		"id":   connection.ID,
		"name": connection.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Prisma connection", map[string]any{
		"id":          state.ID.ValueString(),
		"database_id": state.DatabaseID.ValueString(),
	})

	// The API doesn't have a GET /connections/{id} endpoint,
	// so we list all connections for the database and find ours
	connections, err := r.client.ListConnections(ctx, state.DatabaseID.ValueString())
	if err != nil {
		// If database doesn't exist, connection is gone too
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Database not found, removing connection from state", map[string]any{
				"id":          state.ID.ValueString(),
				"database_id": state.DatabaseID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading connection",
			"Could not list connections for database "+state.DatabaseID.ValueString()+": "+err.Error(),
		)
		return
	}

	var found bool
	for _, conn := range connections {
		if conn.ID == state.ID.ValueString() {
			found = true
			state.Name = types.StringValue(conn.Name)
			state.CreatedAt = types.StringValue(conn.CreatedAt)
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Connection not found, removing from state", map[string]any{
			"id": state.ID.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	// Credentials are only returned on create, not on GET - preserved in state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Prisma connections cannot be updated. Changes to attributes require replacing the resource.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Prisma connection", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteConnection(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Connection already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting connection",
			"Could not delete connection ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

// ImportState imports the resource state.
// Import ID format: database_id,connection_id.
func (r *ConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: database_id,connection_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}
