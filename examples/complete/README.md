# Complete Example

This example demonstrates a complete setup with multiple databases and connections.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for local development)
- A Prisma Service Token from [console.prisma.io](https://console.prisma.io)

## Resources Created

| Resource | Description |
|----------|-------------|
| `prisma-postgres_project` | Container for organizing databases |
| `prisma-postgres_database` | Production and staging databases |
| `prisma-postgres_connection` | Separate API keys for different services |
| `prisma-postgres_regions` | Data source to list available regions |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Project: "my-saas-app"                                 │
│                                                         │
│  ┌─────────────────────┐  ┌─────────────────────┐      │
│  │  Database           │  │  Database           │      │
│  │  (production)       │  │  (staging)          │      │
│  │                     │  │                     │      │
│  │  ┌───────────────┐  │  │  ┌───────────────┐  │      │
│  │  │ Connection    │  │  │  │ Connection    │  │      │
│  │  │ (api-server)  │  │  │  │ (staging-app) │  │      │
│  │  └───────────────┘  │  │  └───────────────┘  │      │
│  │  ┌───────────────┐  │  │                     │      │
│  │  │ Connection    │  │  │                     │      │
│  │  │ (worker)      │  │  │                     │      │
│  │  └───────────────┘  │  │                     │      │
│  └─────────────────────┘  └─────────────────────┘      │
└─────────────────────────────────────────────────────────┘
```

## Quick Start (After Provider is Published)

Once the provider is published to the Terraform Registry:

```bash
# 1. Get a Prisma Service Token from https://console.prisma.io
#    Go to Settings → Service Tokens → Create Token

# 2. Set the token as an environment variable
export PRISMA_SERVICE_TOKEN="prsc_your_token_here"

# 3. Initialize and apply
cd examples/complete
terraform init
terraform plan
terraform apply

# 4. View your connection strings
terraform output database_url
terraform output direct_url
```

## Local Development (Before Publishing)

To test the provider locally before it's published to the registry:

```bash
# 1. Clone and build the provider
git clone https://github.com/prisma/terraform-provider-prisma-postgres.git
cd terraform-provider-prisma-postgres
make install

# 2. Find where Go installed the binary
go env GOBIN   # If empty, it's $(go env GOPATH)/bin

# 3. Create a Terraform dev override config
#    Replace the path below with your actual Go bin path
cat > ~/.terraformrc << 'EOF'
provider_installation {
  dev_overrides {
    "prisma/prisma-postgres" = "/path/to/your/go/bin"
  }
  direct {}
}
EOF

# 4. Set your Prisma Service Token
export PRISMA_SERVICE_TOKEN="prsc_your_token_here"

# 5. Run terraform (skip 'init' when using dev_overrides!)
cd examples/complete
terraform plan
terraform apply
```

> **Note**: With `dev_overrides`, you skip `terraform init` and go directly to `terraform plan`.

## Clean Up

```bash
# Destroy all created resources
terraform destroy

# Remove the dev override when done testing
rm ~/.terraformrc
```

## Outputs

After applying, you'll have access to these outputs:

| Output | Description |
|--------|-------------|
| `database_url` | Prisma Accelerate connection string (use as `DATABASE_URL`) |
| `direct_url` | Direct PostgreSQL URL (use as `DIRECT_URL` for migrations) |
| `worker_database_url` | Separate connection for background workers |
| `staging_database_url` | Staging environment connection |

## Use with Prisma ORM

```bash
# Export the connection strings
export DATABASE_URL=$(terraform output -raw database_url)
export DIRECT_URL=$(terraform output -raw direct_url)

# Run Prisma commands
npx prisma migrate dev
npx prisma generate
```
