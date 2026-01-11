# Repository Guidelines

## Project Structure & Module Organization
- `main.go` is the API entrypoint (Gin HTTP server).
- `cmd/` contains CLI tools (e.g., `init_admin`, `dbinspect`).
- `controllers/`, `routers/`, `middleware/` define HTTP handlers, route wiring, and middleware.
- `services/` holds business logic and background workers.
- `models/` and `db/` contain Gorm models and database setup (MySQL + ClickHouse).
- `config/` reads `config.yaml` and CLI flags; `utils/`, `dns/` provide helpers.
- `scripts/` has maintenance/debug scripts (Go, Python, PowerShell).
- `../common` is a local module dependency via `go.mod` `replace`.

## Build, Test, and Development Commands
- `go run .` starts the API server (reads `config.yaml` by default).
- `go run . -config config.yaml -port 8080 -db "dsn"` overrides config via flags.
- `go build ./...` compiles all packages.
- `go run ./cmd/init_admin <user> <pass> [email]` creates or updates an admin user.
- `go run ./cmd/dbinspect` inspects database/schema values.
- `go test ./...` runs unit tests across packages.

## Coding Style & Naming Conventions
- Format Go code with `gofmt` before committing.
- Follow Go naming: `MixedCaps` for exported identifiers, `lowerCamel` for local.
- Use lowercase file names, typically with underscores (e.g., `task_service.go`).

## Testing Guidelines
- Tests use Go’s `testing` package with `testify/assert`.
- Test files follow `*_test.go` naming (see `services/task_service_test.go`).
- Some tests require a configured DB (`config.yaml`); they skip when `db.DB` is nil.

## Commit & Pull Request Guidelines
- No Git history is available in this checkout; use short, imperative commit messages.
- PRs should include a concise summary, test results (or reason not run), and any
  config or schema changes.

## Configuration & Security Notes
- `config.yaml` contains credentials and tokens; keep secrets out of version control.
- Use `-config` to point to a local config file and avoid sharing sensitive values.
