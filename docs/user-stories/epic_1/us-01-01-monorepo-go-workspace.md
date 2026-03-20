# US-01-01 — Monorepo and Go Workspace Initialisation

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform engineer**, I need the Postulate monorepo and Go workspace structure initialised so that all modules — API, CLI, SDK, and plugins — have a consistent, buildable home with shared tooling and enforced conventions from day one.

---

## 2. Background

The monorepo uses Go workspaces (`go.work`) to manage multiple modules within a single repository without requiring module versioning overhead for internal dependencies. The API and CLI share types and interfaces from a common `sdk` module. Official plugins are co-located under `plugins/` so they are tested against the API in the same CI run.

This story has no runtime behaviour. Its output is the structural skeleton all other stories build within.

---

## 3. Acceptance Criteria

1. The repository root contains a valid `go.work` file that includes all workspace modules.
2. `go work build ./...` and `go work test ./...` complete without error from the repository root.
3. The following module structure exists and each module contains a valid `go.mod`:
   ```
   postulate/
     api/              # module: github.com/postulate/api
     cli/              # module: github.com/postulate/cli
     sdk/              # module: github.com/postulate/sdk
     plugins/
       platform-standards/   # module: github.com/postulate/plugins/platform-standards
     infra/            # Terraform and Kubernetes manifests — no Go module
   ```
4. A root-level `Makefile` exists with the following targets operational:
   - `make build` — builds all Go modules
   - `make test` — runs all tests across the workspace
   - `make lint` — runs `golangci-lint` across all modules
   - `make tidy` — runs `go mod tidy` across all modules
5. A `.golangci.yml` configuration file exists at the repository root with the following linters enabled as a minimum: `errcheck`, `govet`, `staticcheck`, `unused`, `gofmt`, `goimports`.
6. A root-level `.gitignore` covers Go build artefacts, IDE files, `.env` files, and binary outputs.
7. A root-level `.editorconfig` enforces: UTF-8 encoding, LF line endings, 4-space indentation for Go files, tab indentation for Makefiles, and a trailing newline.
8. A `CONTRIBUTING.md` at the repository root documents the module structure, how to add a new module, and how to run the full test suite.

---

## 4. Tasks

### Task 1 — Initialise Git repository and root scaffolding
- Initialise git repository with an initial commit
- Create `.gitignore` covering Go, IDE, binary, and secret file patterns
- Create `.editorconfig` with the settings defined in acceptance criteria
- Create root-level `README.md` with a one-paragraph project description and a link to architecture documentation

### Task 2 — Initialise Go modules
- Create `api/` directory with `go.mod` declaring module `github.com/postulate/api`, Go 1.26
- Create `cli/` directory with `go.mod` declaring module `github.com/postulate/cli`, Go 1.26
- Create `sdk/` directory with `go.mod` declaring module `github.com/postulate/sdk`, Go 1.26
- Create `plugins/platform-standards/` with `go.mod` declaring module `github.com/postulate/plugins/platform-standards`, Go 1.26
- Create `go.work` at repository root referencing all four modules

### Task 3 — Initialise Go workspace
- Run `go work sync` to verify the workspace resolves correctly
- Add `go.work.sum` to version control
- Verify `go work build ./...` produces no errors

### Task 4 — Makefile
- Create root `Makefile` with `build`, `test`, `lint`, `tidy`, and `help` targets
- `lint` target must invoke `golangci-lint run ./...` from each module directory
- `test` target must pass `-race` flag to catch data races from the outset
- `help` target must print available targets with a one-line description each

### Task 5 — Linter configuration
- Create `.golangci.yml` at repository root
- Enable linters: `errcheck`, `govet`, `staticcheck`, `unused`, `gofmt`, `goimports`, `gocritic`, `gosec`
- Configure `goimports` to enforce the import grouping convention: stdlib, external, internal
- Set `issues.max-issues-per-linter` to 0 — report all issues
- Verify `make lint` passes against the empty module stubs

### Task 6 — CONTRIBUTING.md
- Document the module structure and the purpose of each module
- Document how to add a new official plugin module to the workspace
- Document how to run the full test suite
- Document the linter configuration and how to run lint locally
- Document the branching and PR conventions

---

## 5. Definition of Done

- All tasks completed
- `go work build ./...` passes from repository root
- `make lint` passes with zero issues
- `make test` passes (no tests yet — pass by default on empty modules)
- PR reviewed and approved
- All acceptance criteria verified by reviewer
