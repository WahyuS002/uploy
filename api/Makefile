.PHONY: migrate-up migrate-down run dev sqlc oapi generate

migrate-up:
	go run ./cmd/migrate

migrate-down:
	go run ./cmd/migrate down

run:
	go run .

dev:
	air

sqlc:
	sqlc generate

oapi:
	oapi-codegen -config oapi-codegen.yaml openapi.yaml

generate: sqlc oapi
