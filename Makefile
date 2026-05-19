.PHONY: dev build up down logs clean

dev:
	go run ./cmd/bot

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

build:
	docker compose -f docker/docker-compose.yml build

up:
	docker compose -f docker/docker-compose.yml up -d

down:
	docker compose -f docker/docker-compose.yml down

logs:
	docker compose -f docker/docker-compose.yml logs -f bot

clean:
	rm -rf downloads cache bot.db
