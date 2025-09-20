# Sotoon Go SDK

This repository contains the Sotoon Go SDK and a complete generation pipeline which produces typed clients and helpers from the public OpenAPI specification.

The generator downloads the OpenAPI spec, splits it into per-tag sub-APIs, generates Go clients and types, creates per-service handlers (only if missing), and builds a top-level `SDK` wrapper that wires everything together.

## Repository Structure

- `sdk/`
  - `sdk.go` — Top-level SDK wrapper that aggregates all generated service handlers. This file is auto-generated on every run and will be overwritten.
  - `constants/` — Shared constants usable by the SDK. You can edit these.
  - `interceptors/` — HTTP interceptor middleware (auth, logging, retry, etc.). You can edit these.
  - `core/` — One folder per service (derived from OpenAPI tags). Each folder contains:
    - `client.gen.go` — Auto-generated client. Always overwritten.
    - `types.gen.go` — Auto-generated types. Always overwritten.
    - `handler.go` — Lightweight, human-friendly wrapper around the generated client with interceptor support. Created by the generator only if it does not already exist, so you can customize it safely.

- `generator/`
  - `scripts/`
    - `run.sh` — Orchestrates the full generation pipeline (download spec, split, generate SDK).
    - `create-subapis.sh` — Splits the OpenAPI spec into per-tag sub-APIs using `jq`.
    - `filtering.sh` — Utilities used by `create-subapis.sh` to filter endpoints by tag.
    - `create-sdk.sh` — Generates Go code from each sub-API via `oapi-codegen`, then generates handlers and the top-level `sdk.go`.
    - `generate-handler.go` — Creates `handler.go` from a template only if it does not already exist.
    - `generate-sdk.go` — Generates `sdk/sdk.go` from a template by discovering service modules under `sdk/core/`.
  - `templates/`
    - `handler.go.tmpl` — Template used for new service handlers.
    - `sdk.go.tmpl` — Template used for the top-level SDK wrapper.
  - `configs/`
    - `openapi.json` — Downloaded OpenAPI specification (created by the generator).
    - `sub/` — Per-tag filtered OpenAPI JSON files (created by the generator).

- `makefile` — Provides `make generate` to run the full pipeline.
- `go.mod`, `go.sum` — Go module files.

## What You Can Modify

- `sdk/constants/` — Yes, editable.
- `sdk/interceptors/` — Yes, editable.
- `sdk/core/<service>/handler.go` — Yes, editable. The generator will NOT overwrite an existing handler. If the file is missing, it will be created from the template.

## What Is Auto-Generated (Do Not Edit Manually)

- `sdk/core/<service>/client.gen.go` — Always overwritten.
- `sdk/core/<service>/types.gen.go` — Always overwritten.
- `sdk/sdk.go` — Always overwritten (regenerated each run to include all services).
- `generator/configs/openapi.json` — Downloaded each run.
- `generator/configs/sub/*.json` — Recreated each run.

## Requirements

- Go 1.21+
- `jq` (used for JSON filtering)
- `curl` (to download the OpenAPI spec)
- `oapi-codegen` (Go generator):

```bash
go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
```

Ensure your `GOPATH/bin` is on your `PATH` so `oapi-codegen` is available in the shell.

## Generate the SDK

The simplest way is via make:

```bash
make generate
```

This will:

1. Download the latest OpenAPI spec to `generator/configs/openapi.json`.
2. Split the spec into per-tag sub-APIs under `generator/configs/sub/`.
3. Generate clients and types with `oapi-codegen` under `sdk/core/<service>/`.
4. Create `sdk/core/<service>/handler.go` if it does not exist yet.
5. Generate/overwrite the top-level `sdk/sdk.go` wrapper.
6. Run `go fmt` over the repository.

## Adding a New Service

- Add a new tag in the OpenAPI spec (or when the upstream API gains a new tag).
- Run `make generate`.
- The generator will detect the new tag, create `sdk/core/<service>/` with generated `client.gen.go`, `types.gen.go`, and will create `handler.go` only if missing.
- Customize the new handler if needed.

## Notes

- Handlers are intentionally not overwritten to preserve your custom logic. If you need to re-create a handler from the template, delete the existing `handler.go` and run `make generate` again.
- The top-level SDK wrapper `sdk/sdk.go` is regenerated on every run to ensure it reflects the current set of services.
