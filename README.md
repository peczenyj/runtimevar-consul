# runtimevar-contrib

Third-party drivers for [`gocloud.dev/runtimevar`](https://pkg.go.dev/gocloud.dev/runtimevar).

| Driver | Backend | Import |
|---|---|---|
| [`consulvar`](./consulvar) | HashiCorp Consul KV | `github.com/peczenyj/runtimevar-contrib/consulvar` |

## Quick start — `consulvar`

```go
import (
    "context"

    "github.com/peczenyj/runtimevar-contrib/consulvar"
    _ "github.com/peczenyj/runtimevar-contrib/consulvar" // register the consul:// scheme
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

Common commands (the `Makefile` delegates to `task`):

```bash
task test               # unit tests
task test:integration   # unit + integration (requires Docker)
task lint
task format
task changelog:unreleased
```

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
