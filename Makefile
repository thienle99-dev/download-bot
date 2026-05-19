.PHONY: dev build up down logs clean

dev:
	go run ./cmd/bot

# Run both Backend and Frontend Dev server (Vite on port 5173, Go on port 8080)
dev-all:
	@echo "🚀 Starting Go backend and Vite frontend dev server..."
	@trap 'kill 0' INT; (cd web && pnpm run dev) & go run ./cmd/bot

# Run Go backend and compile Svelte on changes (served directly by Go on port 8080)
dev-watch:
	@echo "🚀 Starting Go backend and Frontend auto-rebuild (Watch)..."
	@trap 'kill 0' INT; (cd web && pnpm run build --watch) & go run ./cmd/bot

web-dev:
	cd web && pnpm run dev

web-build:
	cd web && pnpm run build

build:
	docker compose -f docker/docker-compose.yml build

build-prod:
	docker build --platform linux/amd64 -f docker/Dockerfile -t chithien0909/download-bot:latest .

push: build-prod
	docker push chithien0909/download-bot:latest

up:
	docker compose -f docker/docker-compose.yml up -d

up-prod:
	docker compose -f docker/docker-compose.prod.yml up -d

down:
	docker compose -f docker/docker-compose.yml down

logs:
	docker compose -f docker/docker-compose.yml logs -f bot

clean:
	rm -rf downloads cache bot.db
