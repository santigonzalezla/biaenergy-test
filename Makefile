include .env
export

MIGRATIONS_DIR := backend/db/migrations

.PHONY: db-up db-down db-reset db-psql migrate-new migrate-up migrate-down migrate-status sqlc dev

#database
db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-reset:
	docker compose down -v && docker compose up -d postgres

db-psql:
	docker exec -it biaenergy-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

#migrations
migrate-new:
	goose -s -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

#codegeneration
sqlc:
	cd backend && sqlc generate

#dev
dev:
	cd backend && air
