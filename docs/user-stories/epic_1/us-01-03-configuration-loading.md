# US-01-03 — Configuration Loading and Startup Validation

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform engineer**, I need the API server to load its configuration from a YAML file with environment variable overrides so that the server can be configured consistently across local development, CI, and production environments without code changes.

---

## 2. Background

Configuration must support two sources in priority order: environment variables take precedence over the YAML file. This supports containerised deployment (where environment variables are the standard injection mechanism) while allowing a developer-friendly config file for local development.

The server must fail fast at startup if required configuration is absent or invalid. A server that starts with missing critical configuration and fails later during a request is harder to diagnose than one that refuses to start with a clear error message.

No secrets — database credentials, API keys, AI model keys — are stored in the config file or as plain environment variables. Secret management is addressed in later Epics. This story covers structural and operational configuration only.

---

## 3. Acceptance Criteria

1. The server loads configuration from a YAML file at a path specified by the `POSTULATE_CONFIG_FILE` environment variable. If this variable is absent, it defaults to `./config.yaml`.
2. Any configuration value in the YAML file can be overridden by a corresponding environment variable. The environment variable naming convention is `POSTULATE_` prefix followed by the uppercased, underscore-separated key path (e.g., `server.port` → `POSTULATE_SERVER_PORT`).
3. The server fails to start and exits with a non-zero exit code if any of the following required configuration values are absent or invalid:
   - `server.port` — must be a valid port number (1–65535)
   - `server.environment` — must be one of `development`, `staging`, `production`
   - `server.shutdown_timeout_seconds` — must be a positive integer
   - `observability.service_id` — must be a non-empty string
4. On successful startup, the server logs a summary of the loaded configuration at `INFO` level. Sensitive values must not appear in this log summary — if a field is designated sensitive, log its key with value `[redacted]`.
5. A `config.example.yaml` file exists in the `api/` directory documenting all available configuration keys with their types, default values, and descriptions.
6. The configuration struct is fully validated before the server starts — validation errors are collected and reported together, not one at a time.
7. Unit tests cover: successful load from file, environment variable override, missing required field, invalid field value, and sensitive field redaction in log output.

---

## 4. Configuration Schema

The following is the full configuration schema for this Epic. Later Epics will extend this schema.

```yaml
# config.example.yaml

server:
  port: 8080                          # required — integer 1-65535
  environment: development            # required — development | staging | production
  shutdown_timeout_seconds: 30        # required — positive integer

observability:
  service_id: postulate-api           # required — non-empty string
  instance_id: ""                     # optional — populated from hostname if absent
  otlp_endpoint: ""                   # optional — OTLP gRPC endpoint for trace/metric export
  log_level: info                     # optional — debug | info | warn | error (default: info)
```

---

## 5. Tasks

### Task 1 — Add configuration dependencies
- Add `gopkg.in/yaml.v3` to `api/go.mod` for YAML parsing
- Run `go mod tidy` in `api/`

### Task 2 — Define the full configuration struct
- Create `api/internal/config/config.go`
- Define `Config` as the root struct containing `Server ServerConfig` and `Observability ObservabilityConfig`
- Define `ServerConfig` struct: `Port int`, `Environment string`, `ShutdownTimeoutSeconds int`
- Define `ObservabilityConfig` struct: `ServiceID string`, `InstanceID string`, `OTLPEndpoint string`, `LogLevel string`
- All fields must have `yaml` struct tags

### Task 3 — Implement the loader
- Create `api/internal/config/loader.go`
- Implement `Load(configFilePath string) (*Config, error)` function
- Read and parse the YAML file at the given path
- After YAML parsing, apply environment variable overrides using the `POSTULATE_` prefix convention
- If `POSTULATE_CONFIG_FILE` is set and the file does not exist, return a descriptive error
- If the default `./config.yaml` does not exist, proceed with an empty config (environment variables only)

### Task 4 — Implement validation
- Create `api/internal/config/validation.go`
- Implement `Validate(cfg *Config) error` function
- Collect all validation errors into a single `error` using a custom `ValidationErrors` type that implements `error`
- Validate all required fields and value constraints defined in acceptance criteria
- Validation errors must name the failing field and describe the constraint violated

### Task 5 — Implement sensitive field redaction
- Create `api/internal/config/redact.go`
- Implement `LogSafe(cfg *Config) map[string]any` function that returns a map safe for logging
- Any field tagged as sensitive must appear in the map with value `"[redacted]"`
- For this Epic, no fields are sensitive — the structure must exist for later Epics to use

### Task 6 — Wire into main entrypoint
- Update `api/cmd/api/main.go` to call `config.Load()` and `config.Validate()` at startup
- On validation failure, log all errors and exit with code 1
- On success, log the safe configuration summary at `INFO` level

### Task 7 — Create example configuration file
- Create `api/config.example.yaml` documenting all configuration keys per the schema in Section 4

### Task 8 — Unit tests
- Create `api/internal/config/loader_test.go`
- Test: successful load from a valid YAML file
- Test: environment variable override takes precedence over file value
- Test: missing config file at default path proceeds without error
- Test: missing config file at explicitly specified path returns error
- Create `api/internal/config/validation_test.go`
- Test: valid config passes validation
- Test: missing required field produces a named validation error
- Test: invalid port value produces a validation error
- Test: invalid environment value produces a validation error
- Test: all validation errors are returned together, not just the first

---

## 6. Definition of Done

- All tasks completed
- Server starts with a valid config file and logs the configuration summary
- Server exits with code 1 and prints all validation errors when configuration is invalid
- Environment variable override verified manually and in tests
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
