# -------------------------------------
# Shared Configuration
# -------------------------------------
# This file contains common configuration variables shared across all Makefiles.
# It is automatically included by Makefile and database.mk.


# GOBIN - Directory for installed Go binaries (goose, golangci-lint, mockgen, etc.)
# Default: ./bin in the project root
GOBIN			?= $(PWD)/bin

# ENV_CONFIG_FILE - Path to environment configuration file
# Used by database.mk to read DB_DRIVER and DB_DSN
# Default: .env.local in the project root
ENV_CONFIG_FILE ?= $(PWD)/.env.local

# Ensure GOBIN directory exists
$(shell mkdir -p $(GOBIN))
