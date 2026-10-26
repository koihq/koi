.PHONY: compose-up compose-down ui-build ensure-ui run test dev mocks openapi format go-format-check

OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.22.0

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
