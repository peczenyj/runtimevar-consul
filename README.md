# runtimevar-contrib

[![tag](https://img.shields.io/github/tag/peczenyj/runtimevar-contrib.svg)](https://github.com/peczenyj/runtimevar-contrib/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.26-%23007d9c)
[![GoDoc](https://pkg.go.dev/badge/github.com/peczenyj/runtimevar-contrib)](http://pkg.go.dev/github.com/peczenyj/runtimevar-contrib)
[![ci](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/ci.yml/badge.svg)](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/peczenyj/runtimevar-contrib/graph/badge.svg?token=9y6f3vGgpr)](https://codecov.io/gh/peczenyj/runtimevar-contrib)
[![Report card](https://goreportcard.com/badge/github.com/peczenyj/runtimevar-contrib)](https://goreportcard.com/report/github.com/peczenyj/runtimevar-contrib)
[![CodeQL](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/peczenyj/runtimevar-contrib/actions/workflows/dependency-review.yml)
[![License](https://img.shields.io/github/license/peczenyj/runtimevar-contrib)](./LICENSE)

Third-party drivers for [`gocloud.dev/runtimevar`](https://pkg.go.dev/gocloud.dev/runtimevar).

| Driver | Backend | Import |
|---|---|---|
| [`consulvar`](./consulvar) | HashiCorp Consul KV | `github.com/peczenyj/runtimevar-contrib/consulvar` |

## Quick start — `consulvar`

```go
import (
    "context"

    _ "github.com/peczenyj/runtimevar-contrib/consulvar" // registers the consul:// scheme
    "gocloud.dev/runtimevar"
)

func main() {
    ctx := context.Background()
    v, err := runtimevar.OpenVariable(ctx, "consul://services/auth/db_url?decoder=string")
    if err != nil { panic(err) }
    defer v.Close()

    snap, err := v.Latest(ctx)
    if err != nil { panic(err) }
    println(snap.Value.(string))
}
```

The opener reads Consul connection info from the standard environment variables (`CONSUL_HTTP_ADDR`, `CONSUL_HTTP_TOKEN`, …) via `api.DefaultConfig()`. For more control, construct a `*consul/api.Client` yourself and call `consulvar.OpenVariable(client, key, opts)`.

## Development

Install the toolchain:

```bash
go install gotest.tools/gotestsum@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
# task: https://taskfile.dev/installation/
# git-cliff: https://git-cliff.org/docs/installation
```

Requires Go 1.26 or newer (set by the `go` directive in `go.mod`; the dependencies pull the floor up to it).

Common commands (the `Makefile` delegates to `task`):

```bash
task ci                 # full pre-push gate: tidy + lint + build + tests on the minimum Go
task test               # unit tests
task test:integration   # unit + integration (requires Docker)
task lint
task format
task changelog:unreleased
```

Run `task ci` before pushing — it mirrors the GitHub Actions matrix and pins the toolchain to the minimum supported Go, so a dependency that raises the version floor fails locally instead of only in CI.

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
