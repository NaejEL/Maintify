# Maintify Makefile - Docker and Dev Workflow

# --- General Commands ---
.PHONY: help all build clean reset

help:
	@echo ""
	@echo "Maintify Makefile - Available Commands"
	@echo ""
	@echo "  General:"
	@echo "    make help           Show this help message"
	@echo "    make all            Build everything (core, builder)"
	@echo "    make build          Alias for 'make all'"
	@echo "    make clean          Remove all containers, volumes, and networks (safe reset)"
	@echo "    make reset          Remove all containers, volumes, then rebuild and start"
	@echo ""
	@echo "  Core/Builder:"
	@echo "    make core           Build the core service"
	@echo "    make builder        Build the builder service"
	@echo ""
	@echo "  Docker Compose Workflow:"
	@echo "    make up             Build and start all services"
	@echo "    make down           Stop all services"
	@echo "    make restart        Restart all services"
	@echo "    make logs           Tail logs for all services"
	@echo "    make shell          Open a shell in the core container"
	@echo "    make ps             Show running containers"
	@echo "    make status         Show running containers (alias for ps)"
	@echo ""
	@echo "  Database Utilities:"
	@echo "    make db-backup      Backup the Postgres database to db_backup.sql"
	@echo "    make db-restore     Restore the Postgres database from db_backup.sql"
	@echo ""
	@echo "  RBAC Management:"
	@echo "    make rbac-migrate   Run RBAC database migrations (in Docker)"
	@echo "    make rbac-admin     Create system administrator user (in Docker)"
	@echo "    make rbac-org       Create new organization (in Docker)"
	@echo "    make rbac-status    Show migration status (in Docker)"
	@echo "    make rbac-users     List all users (in Docker)"
	@echo "    make rbac-orgs      List all organizations (in Docker)"
	@echo ""
	@echo "  Testing (100% Docker-based):"
	@echo "    make test           Run complete test suite in Docker"
	@echo "    make test-unit      Run unit tests only in Docker"
	@echo "    make test-integration Run integration tests only in Docker"
	@echo "    make test-static    Run static analysis only in Docker"
	@echo "    make test-security  Run security scans only in Docker"
	@echo "    make test-quick     Run quick tests (no security) in Docker"
	@echo "    make test-dashboard Start test results dashboard"
	@echo "    make test-clean     Clean test environment"
	@echo ""
	@echo "  Notes:"
	@echo "    All RBAC commands run inside Docker containers."
	@echo "    No local Go installation required."

# --- Build Targets ---
.PHONY: all build core builder

all: build

build: core builder

# --- Clean/Reset Targets ---
.PHONY: clean reset

clean:
	docker-compose down --remove-orphans --volumes
	docker system prune -f
	docker network prune -f

reset:
	docker-compose down --remove-orphans --volumes
	docker network rm maintify_default || true
	docker-compose up --build

# --- Service Build Targets ---

core:
	docker-compose build core

builder:
	docker-compose build builder

# --- Docker Compose Workflow ---
.PHONY: up down restart logs shell ps status

up:
	docker-compose up --build

down:
	docker-compose down

restart:
	$(MAKE) down
	$(MAKE) up

logs:
	docker-compose logs -f --tail=100

shell:
	docker-compose exec core sh

ps:
	docker-compose ps

status:
	docker-compose ps

# --- Database Utilities ---
.PHONY: db-backup db-restore

db-backup:
	docker-compose exec db pg_dump -U $${DB_USER:-maintify} $${DB_NAME:-maintify} > db_backup.sql

db-restore:
	docker-compose exec -T db psql -U $${DB_USER:-maintify} $${DB_NAME:-maintify} < db_backup.sql

# --- RBAC Management ---
.PHONY: rbac-migrate rbac-admin rbac-org rbac-status rbac-users rbac-orgs

rbac-migrate:
	@echo "Running RBAC migrations..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go migrate

rbac-admin:
	@echo "Creating system administrator..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go create-admin

rbac-org:
	@echo "Creating organization..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go create-org

rbac-status:
	@echo "Checking migration status..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go migration-status

rbac-users:
	@echo "Listing users..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go list-users

rbac-orgs:
	@echo "Listing organizations..."
	docker run --rm -it --network maintify_default \
		-e DB_HOST=db -e DB_PORT=5432 \
		-e DB_USER=$${DB_USER:-maintify} \
		-e DB_PASSWORD=$${DB_PASSWORD:-maintify} \
		-e DB_NAME=$${DB_NAME:-maintify} \
		-v "$(PWD):/app" -w /app/core golang:1.24 \
		go run cmd/rbac-cli/main.go list-orgs

# --- Testing Commands (100% Docker-based) ---
.PHONY: test test-unit test-integration test-static test-security test-quick test-dashboard test-clean test-build

test:
	@echo "Running complete test suite in Docker..."
	docker-compose -f tests/docker-compose.test.yml run --rm static-analysis
	docker-compose -f tests/docker-compose.test.yml run --rm unit-tests
	docker-compose -f tests/docker-compose.test.yml run --rm integration-tests
	docker-compose -f tests/docker-compose.test.yml run --rm security-scan
	docker-compose -f tests/docker-compose.test.yml up -d test-dashboard
	@echo "All tests completed! View results at http://localhost:8080"

test-unit:
	@echo "Running unit tests in Docker..."
	docker-compose -f tests/docker-compose.test.yml run --rm unit-tests

test-integration:
	@echo "Running integration tests in Docker..."
	docker-compose -f tests/docker-compose.test.yml up -d test-db test-redis
	@echo "Waiting for services..."
	@sleep 10
	docker-compose -f tests/docker-compose.test.yml run --rm integration-tests
	docker-compose -f tests/docker-compose.test.yml stop test-db test-redis

test-static:
	@echo "Running static analysis in Docker..."
	docker-compose -f tests/docker-compose.test.yml up -d sonarqube
	@echo "Waiting for SonarQube..."
	@sleep 15
	docker-compose -f tests/docker-compose.test.yml run --rm static-analysis

test-security:
	@echo "Running security scans in Docker..."
	docker-compose -f tests/docker-compose.test.yml run --rm security-scan

test-quick:
	@echo "Running quick tests in Docker (no security scan)..."
	docker-compose -f tests/docker-compose.test.yml run --rm unit-tests
	docker-compose -f tests/docker-compose.test.yml up -d test-db test-redis
	@sleep 10
	docker-compose -f tests/docker-compose.test.yml run --rm integration-tests
	docker-compose -f tests/docker-compose.test.yml stop test-db test-redis

test-dashboard:
	@echo "Starting test results dashboard..."
	docker-compose -f tests/docker-compose.test.yml up -d test-dashboard
	@echo "Dashboard available at http://localhost:8080"

test-clean:
	@echo "Cleaning up test environment..."
	docker-compose -f tests/docker-compose.test.yml down -v
	docker volume prune -f

test-build:
	@echo "Building test containers..."
	docker-compose -f tests/docker-compose.test.yml build
