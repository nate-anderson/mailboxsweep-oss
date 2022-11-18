.PHONY: up
up:
	docker-compose -f ./.docker-dev/docker-compose.yml up -d

.PHONY: down
down:
	docker-compose -f ./.docker-dev/docker-compose.yml down