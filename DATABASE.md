# Database Setup and Management

Goque supports three database backends: PostgreSQL, MySQL, and SQLite.

## Universal Commands

All database operations use the same commands regardless of the database backend.
Simply set `DB_DRIVER` in `.env.local`:

```bash
# .env.local
DB_DRIVER=postgres
DB_DSN=postgres://postgres:postgres@localhost:5432/goque?sslmode=disable

# Or for MySQL
# DB_DRIVER=mysql
# Or for SQLite
# DB_DRIVER=sqlite3
```

Then use generic commands that automatically work with your configured database:

```bash
# Show current database configuration
make db-info

# Apply migrations (works with current DB_DRIVER)
make db-up

# Check status
make db-status

# Create new migration
make db-migrate-create name="add_new_field"

# Rollback
make db-down

# Generate models (PostgreSQL only)
make db-models
```

## Quick Start by Database

### PostgreSQL (Default)

```bash
# Configure in .env.local
DB_DRIVER=postgres
DB_DSN=postgres://postgres:postgres@localhost:5432/goque?sslmode=disable

# Use universal commands
make db-info    # Show configuration
make db-up      # Apply migrations
make db-status  # Check status
make db-models  # Generate models (PostgreSQL only)
```

### MySQL

```bash
# Configure in .env.local
DB_DRIVER=mysql
DB_DSN=root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC

# Use universal commands
make db-info    # Show configuration
make db-up      # Apply migrations
make db-status  # Check status
```

Docker Compose setup:
```bash
make docker-mysql-up    # Start MySQL with Docker Compose
make docker-down        # Stop and remove all containers
```

### SQLite

```bash
# Configure in .env.local
DB_DRIVER=sqlite3
DB_DSN=./goque.db

# Use universal commands
make db-info    # Show configuration
make db-up      # Apply migrations
make db-status  # Check status
```

For in-memory database:
```bash
DB_DRIVER=sqlite3 DB_DSN=":memory:" make db-up
```

## Environment Variables

### Required
- `DB_DRIVER` - Database driver: `postgres`, `mysql`, `sqlite3`
- `DB_DSN` - Database connection string (format depends on driver)

### How DB_DSN is Determined

The `DB_DSN` variable is read from `.env.local`. If `.env.local` doesn't exist or doesn't contain `DB_DSN`, a **warning** will be displayed and the following defaults are used based on `DB_DRIVER`:

- **PostgreSQL** (`DB_DRIVER=postgres`): `postgres://postgres:postgres@localhost:5432/goque?sslmode=disable`
- **MySQL** (`DB_DRIVER=mysql`): `root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC`
- **SQLite** (`DB_DRIVER=sqlite3`): `./goque.db`

Example warning message:
```
database.mk:24: DB_DSN not found in .env.local, using defaults based on DB_DRIVER=postgres
```

### Override via Environment Variable
You can also override `DB_DSN` directly via environment variable:
```bash
DB_DSN="postgres://user:pass@localhost/mydb" make db-up
```

### Configuration File

Create `.env.local`:
```bash
# PostgreSQL
DB_DRIVER=postgres
DB_DSN=postgres://user:password@localhost:5432/dbname?sslmode=disable

# Or MySQL
DB_DRIVER=mysql
DB_DSN=root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC

# Or SQLite
DB_DRIVER=sqlite3
DB_DSN=./goque.db
```

## Migration Structure

```
migrations/
├── pg/          # PostgreSQL migrations
├── mysql/       # MySQL migrations
└── sqlite/      # SQLite migrations
```

Each directory contains version-controlled SQL migrations managed by goose.

## Available Commands

All commands work with the database configured in `DB_DRIVER`:

- `make db-info` - Show current database configuration
- `make db-migrate-create name="migration_name"` - Create new migration
- `make db-up` - Apply all pending migrations
- `make db-down` - Rollback last migration
- `make db-status` - Show migration status
- `make db-models` - Generate models (PostgreSQL only)

### Switching Between Databases
You can easily switch between databases by changing `DB_DRIVER` in `.env.local`:

```bash
# Switch to MySQL
echo "DB_DRIVER=mysql" >> .env.local
make db-info  # Verify configuration
make db-up    # Apply MySQL migrations

# Switch to SQLite
echo "DB_DRIVER=sqlite3" >> .env.local
make db-info  # Verify configuration
make db-up    # Apply SQLite migrations

# Switch back to PostgreSQL
echo "DB_DRIVER=postgres" >> .env.local
make db-info  # Verify configuration
make db-up    # Apply PostgreSQL migrations
```

Or use environment variable override:
```bash
DB_DRIVER=mysql make db-up
DB_DRIVER=sqlite3 make db-status
DB_DRIVER=postgres make db-models
```

## Docker Compose Commands

You can easily start databases using Docker Compose for local development and testing.
All database services are defined in `docker-compose.yml`.

### Start All Databases
```bash
make docker-up    # Start both PostgreSQL and MySQL
make docker-down  # Stop and remove all containers
make docker-down-volumes  # Stop and remove containers with volumes
```

### Start Individual Databases
```bash
make docker-pg-up      # Start only PostgreSQL on port 5432
make docker-mysql-up   # Start only MySQL on port 3306
```

### Monitor and Manage
```bash
make docker-ps    # Show status of containers
make docker-logs  # Show logs from all containers
```

### Service Configuration

**PostgreSQL:**
- Image: `postgres:17`
- User: `postgres`
- Password: `postgres`
- Database: `goque`
- Port: `5432`
- Volume: `postgres_data`

**MySQL:**
- Image: `mysql:8`
- User: `root`
- Password: `root`
- Database: `goque`
- Port: `3306`
- Volume: `mysql_data`

**SQLite:**
SQLite doesn't require Docker as it uses a local file database.

## Testing with Different Databases

### PostgreSQL
```bash
# Start PostgreSQL with Docker
make docker-pg-up

# Export test database connection (optional, defaults are used)
export DB_DSN="postgres://postgres:postgres@localhost:5432/goque?sslmode=disable"

# Run PostgreSQL storage tests
go test ./internal/storages/pg/task/...
```

### MySQL
```bash
# Start MySQL with Docker
make docker-mysql-up

# Export test database connection (optional, defaults are used)
export DB_DSN="root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC"

# Run MySQL storage tests
go test ./internal/storages/mysql/task/...
```

### SQLite
```bash
# SQLite tests use in-memory database by default
go test ./internal/storages/sqlite/task/...
```

## Notes

- **PostgreSQL**: Uses go-jet for type-safe queries, generates models
- **MySQL**: Uses raw SQL, no model generation needed
- **SQLite**: Uses raw SQL, great for testing with in-memory database
- All implementations share the same interface and behavior
- Choose the database that best fits your deployment requirements
