# Building DevForge

## Requirements

- Go 1.24+
- `bin/dpf` for media-related features

CGO is no longer required because the database stack was removed.

## Build

```bash
go build ./...
```

## Test

```bash
go test ./...
```

## Release bundles

```bash
bash scripts/package_release_bundle.sh --version $(cat VERSION) --target-os linux --target-arch amd64
bash scripts/package_release_bundle.sh --version $(cat VERSION) --target-os darwin --target-arch arm64
```
