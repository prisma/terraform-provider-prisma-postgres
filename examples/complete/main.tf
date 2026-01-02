terraform {
  required_version = ">= 1.0"

  required_providers {
    prisma-postgres = {
      source  = "prisma/prisma-postgres"
      version = "~> 0.1.0"
    }
  }
}

provider "prisma-postgres" {}

variable "project_name" {
  description = "Name of the Prisma project"
  type        = string
  default     = "my-saas-app"
}

data "prisma-postgres_regions" "available" {}

resource "prisma-postgres_project" "main" {
  name = var.project_name
}

resource "prisma-postgres_database" "production" {
  project_id = prisma-postgres_project.main.id
  name       = "production"
  region     = "us-east-1"
}

resource "prisma-postgres_database" "staging" {
  project_id = prisma-postgres_project.main.id
  name       = "staging"
  region     = prisma-postgres_database.production.region
}

resource "prisma-postgres_connection" "api" {
  database_id = prisma-postgres_database.production.id
  name        = "api-server"
}

resource "prisma-postgres_connection" "worker" {
  database_id = prisma-postgres_database.production.id
  name        = "background-worker"
}

resource "prisma-postgres_connection" "staging" {
  database_id = prisma-postgres_database.staging.id
  name        = "staging-app"
}

output "available_regions" {
  description = "List of available Prisma Postgres regions"
  value       = [for r in data.prisma-postgres_regions.available.regions : "${r.id} (${r.name})"]
}

output "project_id" {
  description = "The Prisma project ID"
  value       = prisma-postgres_project.main.id
}

output "production_database_id" {
  description = "Production database ID"
  value       = prisma-postgres_database.production.id
}

output "production_database_status" {
  description = "Production database status"
  value       = prisma-postgres_database.production.status
}

output "database_url" {
  description = "Prisma Accelerate connection string (use as DATABASE_URL)"
  value       = prisma-postgres_connection.api.connection_string
  sensitive   = true
}

output "direct_url" {
  description = "Direct PostgreSQL URL for migrations (use as DIRECT_URL)"
  value       = prisma-postgres_database.production.direct_url
  sensitive   = true
}

output "worker_database_url" {
  description = "Connection string for background workers"
  value       = prisma-postgres_connection.worker.connection_string
  sensitive   = true
}

output "staging_database_url" {
  description = "Staging environment connection string"
  value       = prisma-postgres_connection.staging.connection_string
  sensitive   = true
}

output "staging_direct_url" {
  description = "Staging direct URL for migrations"
  value       = prisma-postgres_database.staging.direct_url
  sensitive   = true
}
