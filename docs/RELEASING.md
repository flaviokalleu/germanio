# Releasing Germanio

Germanio uses [Semantic Versioning](https://semver.org/) and is **pre-1.0**: a minor version
(0.7 → 0.8) may change the language; a patch version (0.7.0 → 0.7.1) only fixes bugs. Do not
announce 1.0 until the language is stable enough to promise compatibility (see
[`docs/gep/`](gep/README.md)).

## Checklist

1. `master` is green in CI and locally: `gofmt -l .` is empty, `go vet ./...`,
   `go test ./...` and `go test -race ./runtime/... ./compiler/... ./tooling/...` pass.
2. The examples pass: `ge check` and `ge fmt --check` on `examples/` (CI does it).
3. [`CHANGELOG.md`](../CHANGELOG.md): move the *Unreleased* items under the new version with
   today's date, and update the comparison links at the bottom.
4. Update the development version in [`versao/versao.go`](../versao/versao.go) to the next
   `-dev` after tagging (the tag itself sets the release version at link time).
5. Tag and push:

   ```bash
   git tag -a v0.7.0 -m "Germanio 0.7.0"
   git push origin v0.7.0
   ```

6. The [release workflow](../.github/workflows/release.yml) runs GoReleaser
   ([`.goreleaser.yml`](../.goreleaser.yml)): it runs the tests, builds `ge` and `germanio`
   for Linux, macOS and Windows (amd64 and arm64) with the version and commit embedded,
   packs them with the README, LICENSE and CHANGELOG, writes `checksums.txt` (SHA-256) and
   publishes the GitHub release (marked pre-release while the version has a suffix).
7. Check the release page: the archives, `checksums.txt`, and that `ge --version` of a
   downloaded binary prints the tag's version.

## Try it locally first

```bash
go run github.com/goreleaser/goreleaser/v2@latest check
go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean   # builds into dist/, publishes nothing
```

## Benchmarks

When a release changes the parser, the resolver, the runtime or the server, run
`scripts/bench.sh` on a quiet machine and compare with the previous release using
`benchstat` ([`bench/README.md`](../bench/README.md)). A significant regression is recorded
and decided before the release, never shipped silently.
