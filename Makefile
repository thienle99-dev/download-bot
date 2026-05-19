.PHONY: dev build up down logs clean

dev:
	go run ./cmd/bot

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
