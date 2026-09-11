.PHONY: compose-up compose-down run test

compose-up:
	docker compose up -d

compose-down:
	docker compose down

run:
	go run ./cmd/koi

test:
	go test ./...
