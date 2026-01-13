# Contributing to Terraform Provider for Prisma Postgres

Thank you for your interest in contributing to the Prisma Postgres Terraform Provider.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.24
- [Terraform](https://www.terraform.io/downloads) >= 1.0

### Setting Up the Development Environment

1. Clone the repository:

   ```bash
   git clone https://github.com/prisma/terraform-provider-prisma-postgres.git
   cd terraform-provider-prisma-postgres
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Build the provider:

   ```bash
   go build -o terraform-provider-prisma-postgres
   ```

### Running Tests

Run tests (uses HTTP mocking, no real API calls):

```bash
go test ./...
```

### Local Development

To test the provider locally:

1. Build and install:

   ```bash
   go install .
   ```

2. Create a Terraform dev override in `~/.terraformrc`:

   ```hcl
   provider_installation {
     dev_overrides {
       "prisma/prisma-postgres" = "/path/to/your/go/bin"
     }
     direct {}
   }
   ```

3. Set your Prisma Service Token:

   ```bash
   export PRISMA_SERVICE_TOKEN="your-token"
   ```

   To get your service token:
   1. Go to [console.prisma.io](https://console.prisma.io)
   2. Navigate to **Settings**
   3. Click on **Service Tokens**
   4. Create a new token or copy an existing one

4. Run Terraform (skip `terraform init` when using dev overrides):

   ```bash
   terraform plan
   terraform apply
   ```

## Making Contributions

### Reporting Bugs

Open an issue describing:
- What you expected to happen
- What actually happened
- Steps to reproduce the issue
- Terraform and provider versions

### Suggesting Features

Open an issue describing:
- The use case for the feature
- How you expect it to work

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests to ensure they pass
5. Commit your changes with a descriptive message
6. Push to your fork
7. Open a pull request against `main`

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to check for common issues

## Project Structure

```
.
├── internal/
│   ├── client/       # API client for Prisma Postgres
│   └── provider/     # Terraform provider implementation
├── examples/         # Example Terraform configurations
└── main.go           # Provider entry point
```

## License

By contributing, you agree that your contributions will be licensed under the MPL-2.0 license.
