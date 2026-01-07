# Prisma Postgres Terraform Provider

Terraform provider for managing [Prisma Postgres](https://www.prisma.io/postgres) resources.

[![Go Report Card](https://goreportcard.com/badge/github.com/prisma/terraform-provider-prisma-postgres)](https://goreportcard.com/report/github.com/prisma/terraform-provider-prisma-postgres)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL%202.0-blue.svg)](https://opensource.org/licenses/MPL-2.0)

## Features

- **Projects** — Create and manage Prisma Postgres projects
- **Databases** — Deploy databases to specific regions with direct PostgreSQL access
- **Connections** — Generate API keys with Prisma Accelerate connection strings
- **Regions** — Query available deployment regions

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- Prisma Service Token ([create one here](https://console.prisma.io))

## Installation

### From Terraform Registry

```hcl
terraform {
  required_providers {
    prisma-postgres = {
      source  = "prisma/prisma-postgres"
      version = "~> 0.1.0"
    }
  }
}
```

### Local Development

```bash
git clone https://github.com/prisma/terraform-provider-prisma-postgres.git
cd terraform-provider-prisma-postgres
make install
```

## Quick Start

### 1. Set your service token

```bash
export PRISMA_SERVICE_TOKEN="your-service-token"
```

### 2. Create your Terraform configuration

```hcl
terraform {
  required_providers {
    prisma-postgres = {
      source  = "prisma/prisma-postgres"
      version = "~> 0.1.0"
    }
  }
}

provider "prisma-postgres" {}

resource "prisma-postgres_project" "main" {
  name = "my-app"
}

resource "prisma-postgres_database" "production" {
  project_id = prisma-postgres_project.main.id
  name       = "production"
  region     = "us-east-1"
}

resource "prisma-postgres_connection" "api" {
  database_id = prisma-postgres_database.production.id
  name        = "api-key"
}

output "connection_string" {
  value     = prisma-postgres_connection.api.connection_string
  sensitive = true
}

output "direct_url" {
  value     = prisma-postgres_database.production.direct_url
  sensitive = true
}
```

### 3. Deploy

```bash
terraform init
terraform apply
```

## Provider Configuration

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_token` | string | No | Prisma service token. Can also be set via `PRISMA_SERVICE_TOKEN` environment variable. |

## Resources

### prisma-postgres_project

Manages a Prisma Postgres project.

```hcl
resource "prisma-postgres_project" "example" {
  name = "my-project"
}
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | The name of the project. |

| Attribute | Description |
|-----------|-------------|
| `id` | The unique project ID. |
| `created_at` | ISO 8601 timestamp of creation. |

### prisma-postgres_database

Manages a Prisma Postgres database.

```hcl
resource "prisma-postgres_database" "example" {
  project_id = prisma-postgres_project.main.id
  name       = "production"
  region     = "us-east-1"
}
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `project_id` | string | Yes | The ID of the parent project. |
| `name` | string | Yes | The database name. |
| `region` | string | No | Deployment region. Default: `us-east-1`. |

| Attribute | Sensitive | Description |
|-----------|-----------|-------------|
| `id` | No | The unique database ID. |
| `status` | No | Current status (`provisioning`, `ready`, `failure`). |
| `connection_string` | Yes | Prisma Accelerate connection string. |
| `direct_url` | Yes | Direct PostgreSQL URL. |
| `direct_host` | No | Direct PostgreSQL host. |
| `direct_user` | No | Direct PostgreSQL username. |
| `direct_password` | Yes | Direct PostgreSQL password. |

### prisma-postgres_connection

Manages a database connection (API key).

```hcl
resource "prisma-postgres_connection" "example" {
  database_id = prisma-postgres_database.main.id
  name        = "api-key"
}
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `database_id` | string | Yes | The ID of the parent database. |
| `name` | string | Yes | The connection name. |

| Attribute | Sensitive | Description |
|-----------|-----------|-------------|
| `id` | No | The unique connection ID. |
| `connection_string` | Yes | Prisma Accelerate connection string. |
| `host` | No | Database host. |
| `user` | No | Database username. |
| `password` | Yes | Database password. |

## Data Sources

### prisma-postgres_regions

Lists available deployment regions.

```hcl
data "prisma-postgres_regions" "available" {}

output "regions" {
  value = [for r in data.prisma-postgres_regions.available.regions : r.id]
}
```

## Available Regions

| Region ID | Location |
|-----------|----------|
| `us-east-1` | US East (N. Virginia) |
| `us-west-1` | US West (N. California) |
| `eu-west-3` | Europe (Paris) |
| `eu-central-1` | Europe (Frankfurt) |
| `ap-northeast-1` | Asia Pacific (Tokyo) |
| `ap-southeast-1` | Asia Pacific (Singapore) |

## Importing Resources

```bash
terraform import prisma-postgres_project.example <project-id>
terraform import prisma-postgres_database.example <database-id>
terraform import prisma-postgres_connection.example <database-id>,<connection-id>
```

> **Note:** Credentials are only available at creation time and cannot be recovered after import.

## Use with Prisma ORM

```bash
export DATABASE_URL=$(terraform output -raw connection_string)
export DIRECT_URL=$(terraform output -raw direct_url)
```

```prisma
datasource db {
  provider  = "postgresql"
  url       = env("DATABASE_URL")
  directUrl = env("DIRECT_URL")
}
```

## Development

```bash
make build       # Build the provider
make install     # Install locally
make test        # Run unit tests
TF_ACC=1 make test  # Run all tests (uses mocking, no token needed)
```

## License

MPL-2.0 — See [LICENSE](LICENSE) for details.
