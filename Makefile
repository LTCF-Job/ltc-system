.PHONY: test test-e2e test-web-e2e build run migrate-up migrate-down clean web-dev web-build web-typecheck docker-up docker-down docker-build

test:
	cd apps/api && go test -v ./...

test-e2e:
	node tests/e2e/e2e.spec.js

test-web-e2e:
	cd apps/web && npm run test:e2e

build:
	cd apps/api && go build -o bin/server.exe ./cmd/server
	cd apps/api && go build -o bin/migrate.exe ./cmd/migrate

run:
	cd apps/api && go run ./cmd/server

migrate-up:
	cd apps/api && go run ./cmd/migrate up

migrate-down:
	cd apps/api && go run ./cmd/migrate down

clean:
	rm -rf apps/api/bin apps/web/dist

web-dev:
	cd apps/web && npm run dev

web-build:
	cd apps/web && npm run build

web-typecheck:
	cd apps/web && npm run type-check

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build
