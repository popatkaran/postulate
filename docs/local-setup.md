# Local Development Setup

This guide covers setting up a local development environment for the Postulate API on macOS. Linux instructions are included in a secondary section.

---

## Prerequisites

- [Homebrew](https://brew.sh) installed
- Go 1.26 or later (`go version` to verify)

---

## macOS Setup

### 1. Install PostgreSQL 16

```bash
brew install postgresql@16
```

### 2. Create roles and databases

Run the setup script from the repository root:

```bash
make db-setup
```

This creates:
- Role `postulate_dev` with password `postulate_dev`
- Database `postulate_dev` owned by `postulate_dev`
- Database `postulate_test` owned by `postulate_dev` (used for integration tests)
- `api/config.local.yaml` pre-populated with local connection values (if it does not already exist)

The script is idempotent — safe to run multiple times.

### 3. Start PostgreSQL

```bash
make db-start
```

### 4. Configure the API

Set the config file path so the API picks up your local config:

```bash
export POSTULATE_CONFIG_FILE=api/config.local.yaml
```

Open `api/config.local.yaml` and set `database.password` to `postulate_dev`.

### 5. Start the API

```bash
make run
```

### 6. Verify the API is running

```bash
curl http://localhost:8080/health
```

Expected response: `{"status":"healthy"}` with HTTP 200.

---

## Linux (Ubuntu/Debian)

```bash
sudo apt update && sudo apt install -y postgresql-16
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Create role and databases
sudo -u postgres psql -c "CREATE ROLE postulate_dev WITH LOGIN PASSWORD 'postulate_dev';"
sudo -u postgres psql -c "CREATE DATABASE postulate_dev OWNER postulate_dev;"
sudo -u postgres psql -c "CREATE DATABASE postulate_test OWNER postulate_dev;"
```

> Note: `make db-start` and `make db-stop` use `brew services` and are macOS-only.
> On Linux, use `sudo systemctl start postgresql` and `sudo systemctl stop postgresql` directly.

Then follow steps 4–6 from the macOS section above.

---

## Makefile Database Targets

| Target | Description |
|--------|-------------|
| `make db-setup` | Create roles, databases, and `api/config.local.yaml` |
| `make db-start` | Start PostgreSQL via Homebrew (macOS) |
| `make db-stop` | Stop PostgreSQL via Homebrew (macOS) |
| `make db-status` | Print current PostgreSQL service status |
| `make db-reset` | Drop and recreate `postulate_dev` and `postulate_test` (prompts for confirmation) |

---

## Troubleshooting

**PostgreSQL not running**

```
database unreachable at startup — ensure PostgreSQL is running
hint: run 'make db-start' to start the local PostgreSQL service
```

Run `make db-start` and retry.

**Role does not exist**

```
FATAL: role "postulate_dev" does not exist
```

Run `make db-setup` to create the role and databases.

**Wrong password**

```
FATAL: password authentication failed for user "postulate_dev"
```

Verify `database.password` in `api/config.local.yaml` is set to `postulate_dev`.

**Port already in use**

If port 5432 is occupied by another PostgreSQL instance, stop it first or adjust the `database.port` in your local config.
