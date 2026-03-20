# US-01-10 — Multi-Stage Non-Root Dockerfile

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need the Postulate API to be packaged as a minimal, secure container image so that it can be deployed consistently across environments and satisfies the same security baseline that Postulate enforces in the microservices it generates.

---

## 2. Background

Postulate enforces non-root containers and read-only root filesystems as Tier 1 standards in every generated microservice. The Postulate API itself must conform to these same standards — the platform must eat its own cooking.

A multi-stage Dockerfile is used to keep the final image minimal. The build stage compiles the Go binary. The final stage copies only the binary into a minimal base image — no compiler, no source code, no build tooling.

The Dockerfile must also include a `HEALTHCHECK` instruction so that the Docker daemon (and Docker Compose in local development) can determine container health independently of Kubernetes.

---

## 3. Acceptance Criteria

1. The Dockerfile uses a multi-stage build:
   - **Build stage** — uses the official `golang:1.26-alpine` image, compiles the binary with `CGO_ENABLED=0` and the version linker flags from the Makefile.
   - **Final stage** — uses `gcr.io/distroless/static-debian12` as the base image. No shell. No package manager. Binary only.
2. The final image runs as a non-root user. The `USER` directive is set in the final stage.
3. The final image includes a `HEALTHCHECK` instruction that calls `GET /health` on the configured port with a 5-second timeout, 10-second interval, and 3 retries before marking unhealthy.
4. The image builds successfully with `docker build`.
5. The running container passes `docker inspect --format='{{.State.Health.Status}}'` as `healthy` within 30 seconds of start.
6. The binary in the final image is statically linked — `ldd` against the binary produces `not a dynamic executable`.
7. A `.dockerignore` file excludes: `vendor/`, `*.md`, `docs/`, `infra/`, `.git/`, `*_test.go`, and any file matching `*.local`.
8. A `docker-compose.yml` in `api/` enables local development with: the API service, a volume for the config file, and port mapping for `8080`.
9. `docker scout` or `trivy` image scan produces no `CRITICAL` severity CVEs in the final image.

---

## 4. Tasks

### Task 1 — Create the Dockerfile
- Create `api/Dockerfile`
- Stage 1 (`builder`):
  - Base: `golang:1.26-alpine`
  - Set `WORKDIR /build`
  - Copy `go.mod`, `go.sum` and run `go mod download` before copying source (layer cache optimisation)
  - Copy source and run `go build` with:
    - `CGO_ENABLED=0`
    - `GOOS=linux`
    - `GOARCH=amd64`
    - `-ldflags` injecting version, commit, and build time (matching Makefile `build` target)
    - Output binary to `/build/postulate-api`
- Stage 2 (`final`):
  - Base: `gcr.io/distroless/static-debian12`
  - Copy binary from builder stage
  - Set `USER nonroot:nonroot`
  - Set `HEALTHCHECK` per acceptance criteria
  - `ENTRYPOINT ["/postulate-api"]`

### Task 2 — Create .dockerignore
- Create `api/.dockerignore`
- Exclude: `.git`, `**/*_test.go`, `docs/`, `infra/`, `*.md`, `*.local`, `config.yaml` (config is mounted at runtime, not baked in)

### Task 3 — Create docker-compose.yml for local development
- Create `api/docker-compose.yml`
- Define `api` service using the Dockerfile
- Map port `8080:8080`
- Mount `./config.yaml:/etc/postulate/config.yaml:ro`
- Set `POSTULATE_CONFIG_FILE=/etc/postulate/config.yaml` environment variable
- Define a `healthcheck` matching the Dockerfile `HEALTHCHECK` instruction

### Task 4 — Add Docker targets to Makefile
- Add `docker-build` target: builds the image and tags it as `postulate-api:local`
- Add `docker-run` target: runs `docker-compose up` from the `api/` directory
- Add `docker-scan` target: runs `trivy image postulate-api:local` and fails on `CRITICAL` findings

### Task 5 — Verify non-root and static binary
- Document in `CONTRIBUTING.md` how to verify:
  - Non-root: `docker run --rm --entrypoint whoami postulate-api:local` must output `nonroot`
  - Static binary: `docker run --rm --entrypoint ldd postulate-api:local /postulate-api` must output `not a dynamic executable`

### Task 6 — CI pipeline step
- Add a `docker-build` step to the CI pipeline configuration
- Add image scan step using `trivy` after build
- Fail the pipeline on any `CRITICAL` CVE finding

---

## 5. Definition of Done

- All tasks completed
- `docker build` completes without error
- Container starts, becomes healthy, and serves `GET /health` returning `200`
- Container runs as non-root user verified
- Binary is statically linked verified
- Image scan produces no `CRITICAL` CVEs
- `docker-compose up` starts the API in local development
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
