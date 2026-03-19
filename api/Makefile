.PHONY: migrate-up run dev sqlc

migrate-up:
	go run ./cmd/migrate

run:
	go run .

dev:
	air

sqlc:
	sqlc generate
