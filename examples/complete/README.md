# Complete Example

This example demonstrates a complete setup with multiple databases and connections.

## Resources

| Resource | Description |
|---------|-------------|
| `prisma-postgres_regions` | Data source to list available deployment regions |
| `prisma-postgres_project` | Container for organizing databases |
| `prisma-postgres_database` | Production and staging databases |
| `prisma-postgres_connection` | Separate API keys for different services |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  prisma-postgres_project.main ("my-saas-app")                        │
│                                                              │
│  ┌──────────────────────┐  ┌──────────────────────┐   │
│  │  prisma-postgres_database        │  │  prisma-postgres_database        │   │
│  │  (production)        │  │  (staging)           │   │
│  │                      │  │                      │   │
│  │  ┌────────────────┐  │  │  ┌────────────────┐ │   │
│  │  │ prisma-postgres_connection │  │  │ prisma-postgres_connection │ │   │
│  │  │ (api-server)   │  │  │  │ (staging-app)   │ │   │
│  │  └────────────────┘  │  │  └────────────────┘ │   │
│  │  ┌────────────────┐  │  │                      │   │
│  │  │ prisma-postgres_connection │  │  │                      │   │
│  │  │ (worker)        │  │  │                      │   │
│  │  └────────────────┘  │  │                      │   │
│  └──────────────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Usage

```bash
terraform init
terraform plan
terraform apply
```
