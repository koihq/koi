.PHONY: compose-up compose-down compose-image ui-build site-dev site-build ensure-ui run test dev seed-demo seed-demo-reset migration mocks openapi format go-format-check

OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.22.0

compose-up:
	docker compose up -d --wait

compose-down:
	docker compose down

compose-image:
	docker compose --profile image up --build -d --wait

ui-build:
	pnpm --dir ui build

site-dev:
	pnpm --dir ui site:dev

site-build:
	pnpm --dir ui site:build

ensure-ui:
	@test -f pkg/web/dist/index.html || $(MAKE) ui-build

run: ensure-ui
	go run ./cmd/koi

test: ensure-ui
	go test ./...

seed-demo: ensure-ui
	go run ./cmd/koi admin seed-demo

seed-demo-reset: ensure-ui
	go run ./cmd/koi admin seed-demo --reset

# New schema change: make migration name=add_brand_notes. Never edit a released migration.
migration:
	@test -n "$(name)" || { echo 'usage: make migration name=add_something'; exit 1; }
	go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir pkg/database/migrations -s create $(name) sql

format:
	pnpm --dir ui check:fix
	gofmt -w cmd internal pkg

go-format-check:
	@files="$$(gofmt -l cmd internal pkg)" || exit $$?; \
		if [ -n "$$files" ]; then printf '%s\n' "$$files"; exit 1; fi

dev: compose-up ensure-ui
	@trap 'kill $$(jobs -p) 2>/dev/null; docker compose stop' EXIT; \
		go tool air & \
		pnpm --dir ui dev & \
		wait

mocks:
	go tool mockery

openapi:
	docker run --rm -v "$(CURDIR):/local" $(OPENAPI_GENERATOR_IMAGE) generate \
		-i /local/docs/openapi.yaml \
		-g typescript-fetch \
		-o /local/ui/src/api/generated \
		--additional-properties=supportsES6=true,typescriptThreePlus=true,withInterfaces=true,modelPropertyNaming=camelCase,enumPropertyNaming=UPPERCASE,useSingleRequestParameter=true
	@rm -rf ui/src/api/generated/docs
	@python3 -c "from pathlib import Path; \
root = Path('ui/src/api/generated'); \
[p.write_text('// @ts-nocheck\n' + t) for p in root.rglob('*.ts') if not (t := p.read_text()).startswith('// @ts-nocheck')]"
