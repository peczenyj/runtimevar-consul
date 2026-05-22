# runtimevar-consul

`runtimevar-consul` is a Go library providing a driver for [`gocloud.dev/runtimevar`](https://pkg.go.dev/gocloud.dev/runtimevar) backed by HashiCorp Consul KV. It allows applications to watch Consul KV keys and react to changes in real-time.

## Project Overview

- **Main Technology:** Go (>= 1.26)
- **Primary Dependency:** [`github.com/hashicorp/consul/api`](https://github.com/hashicorp/consul) and [`gocloud.dev/runtimevar`](https://gocloud.dev).
- **Architecture:** 
  - `consulvar/`: Public API and `runtimevar` URL opener registration.
  - `consulvar/internal/driver/`: Internal implementation of the `runtimevar.Driver` interface using Consul blocking queries.
  - `vendor/`: Project dependencies are vendored.
- **Automation:** [`Taskfile.yml`](./Taskfile.yml) defines common development tasks.

## Building and Running

The project uses `task` (Taskfile) as the task runner.

- **Build:** `go build ./...`
- **Unit Tests:** `task test`
- **Integration Tests:** `task test:integration` (Requires Docker for Consul test container)
- **Linting:** `task lint` (Uses `golangci-lint`)
- **Formatting:** `task format`
- **Full CI Gate:** `task ci` (Tidy + Lint + Build + Unit + Integration)

## Development Conventions

- **Branching Model:**
  - `devel`: Integration branch. All pull requests should target `devel`.
  - `main`: Release branch. Tags and releases are cut from `main`.
- **Commit Messages:** Follow [Conventional Commits](https://www.conventionalcommits.org/).
  - Used prefixes: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:`, `build:`.
  - `git-cliff` is used to generate `CHANGELOG.md`.
- **Testing:**
  - Use `github.com/stretchr/testify` for assertions.
  - Integration tests use `testcontainers-go` to spin up a real Consul instance.
- **Dependency Management:**
  - Maintain `go.mod` and `go.sum` via `task tidy`.
  - Avoid raising the minimum Go version floor (currently 1.26) unless necessary.
- **Tooling:**
  - `gotestsum` for test execution and reporting.
  - `golangci-lint` for static analysis.
  - `git-cliff` for changelog management.
