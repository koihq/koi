.PHONY: compose-up compose-down ui-build ensure-ui run test dev mocks

compose-up:
	docker compose up -d --wait

compose-down:
	docker compose down

ui-build:
	pnpm --dir ui build

ensure-ui:
	@test -f pkg/web/dist/index.html || $(MAKE) ui-build

run: ensure-ui
	go run ./cmd/koi

test: ensure-ui
	go test ./...

dev: compose-up ensure-ui
	@trap 'kill $$(jobs -p) 2>/dev/null; docker compose stop' EXIT; \
		go tool air & \
		pnpm --dir ui dev & \
		wait

mocks:
	go tool mockery
