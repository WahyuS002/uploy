.PHONY: migrate-up run dev sqlc oapi generate

migrate-up:
	go run ./cmd/migrate

run:
	go run .

dev:
	air

sqlc:
	sqlc generate

oapi:
	oapi-codegen -config oapi-codegen.yaml openapi.yaml

generate: sqlc oapi
