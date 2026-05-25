# runtimevar-consul

[![tag](https://img.shields.io/github/tag/peczenyj/runtimevar-consul.svg)](https://github.com/peczenyj/runtimevar-consul/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.25-%23007d9c)
[![GoDoc](https://pkg.go.dev/badge/github.com/peczenyj/runtimevar-consul)](http://pkg.go.dev/github.com/peczenyj/runtimevar-consul)
[![ci](https://github.com/peczenyj/runtimevar-consul/actions/workflows/ci.yml/badge.svg)](https://github.com/peczenyj/runtimevar-consul/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/peczenyj/runtimevar-consul/graph/badge.svg?token=9y6f3vGgpr)](https://codecov.io/gh/peczenyj/runtimevar-consul)
[![Report card](https://goreportcard.com/badge/github.com/peczenyj/runtimevar-consul)](https://goreportcard.com/report/github.com/peczenyj/runtimevar-consul)
[![CodeQL](https://github.com/peczenyj/runtimevar-consul/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/peczenyj/runtimevar-consul/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/peczenyj/runtimevar-consul/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/peczenyj/runtimevar-consul/actions/workflows/dependency-review.yml)
[![License](https://img.shields.io/github/license/peczenyj/runtimevar-consul)](./LICENSE)

Third-party driver for [`gocloud.dev/runtimevar`](https://pkg.go.dev/gocloud.dev/runtimevar) to read from Consul KV.

| Driver | Backend | Import |
|---|---|---|
| [`consulvar`](./consulvar) | HashiCorp Consul KV | `github.com/peczenyj/runtimevar-consul/consulvar` |

## Quick start — `consulvar`

```go
import (
    "context"
    "fmt"

    _ "github.com/peczenyj/runtimevar-consul/consulvar" // registers the consul:// scheme
    "gocloud.dev/runtimevar"
)

func main() {
    ctx := context.Background()
    v, err := runtimevar.OpenVariable(ctx, "consul://services/auth/db_url?decoder=string")
    if err != nil { panic(err) }
    defer v.Close()

    snap, err := v.Latest(ctx)
    if err != nil { panic(err) }
    fmt.Println(snap.Value.(string))
}
```

The opener reads Consul connection info from the standard environment variables (`CONSUL_HTTP_ADDR`, `CONSUL_HTTP_TOKEN`, …) via `api.DefaultConfig()`. For more control, construct a `*consul/api.Client` yourself and call `consulvar.OpenVariable(client, key, opts)`.

## Development
Install the toolchain:

```bash
go install gotest.tools/gotestsum@latest
# golangci-lint v2 — see https://golangci-lint.run/welcome/install/#local-installation
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2
go install github.com/vektra/mockery/v2@latest # only needed to regenerate mocks
# task:      https://taskfile.dev/installation/
# git-cliff: https://git-cliff.org/docs/installation
```

Requires Go 1.25 or newer (the `go` directive in `go.mod`).

Common commands (the `Makefile` delegates to `task`):

```bash
task ci                 # full pre-push gate: tidy:check + lint + build + unit + integration
task test               # unit tests
task test:integration   # unit + integration (requires Docker)
task lint
task format
task changelog:unreleased
```

Run `task ci` before pushing — it mirrors the GitHub Actions checks. Both CI and `task ci` run on whatever Go toolchain is installed; the minimum supported version is recorded by the `go` directive in `go.mod` and is not otherwise enforced.

## Commit messages

This repo follows [Conventional Commits](https://www.conventionalcommits.org/) so `git cliff` can generate Keep a Changelog sections. Use `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:`, `build:` prefixes.

## Branching

- `devel` is the integration branch. All PRs target `devel`.
- `main` holds release commits only; tags `vX.Y.Z` are cut on `main`.

## Releasing

1. On `devel`: `task changelog:unreleased` to sanity-check the upcoming section.
2. Open a PR `devel` → `main`.
3. Merge with a merge commit titled `release: vX.Y.Z`.
4. On `main`: `task changelog` to regenerate the full `CHANGELOG.md`; commit it.
5. Tag `vX.Y.Z` on `main`; push the tag.
6. The `release.yml` workflow publishes the GitHub Release from `git cliff --latest`.
7. Fast-forward `main` back into `devel`.

## License

MIT — see [LICENSE](./LICENSE).
