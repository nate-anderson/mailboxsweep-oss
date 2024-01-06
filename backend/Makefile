.PHONY: up
up:
	docker compose -f ./.docker-dev/docker-compose.yml up -d --wait

.PHONY: up_ci
	docker compose -f ./.docker-dev/docker-compose.yml up -d --wait || true

.PHONY: down
down:
	docker compose -f ./.docker-dev/docker-compose.yml down

.PHONY: install_openapi_gen
install_openapi_gen:
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest

# HTTP API server codegen stuff
HTTP_SERVER_SPEC_PATH = ../openapi.yml
HTTP_SERVER_GEN_PATH = http_codegen.yml

.PHONY: install_openapi_generator
install_openapi_generator:
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest	

.PHONY: http_scaffold
http_scaffold:
	oapi-codegen -generate chi-server,types -config $(HTTP_SERVER_GEN_PATH) $(HTTP_SERVER_SPEC_PATH)

.PHONY: migrate_db
migrate_db:
	go run ./cmd/migrate dynamo

.PHONY: migrate_s3
migrate_s3:
	go run ./cmd/migrate s3

.PHONY: migrate_sqs
migrate_sqs:
	go run ./cmd/migrate sqs

.PHONY: migrate
migrate: migrate_db migrate_s3 migrate_sqs
	
.PHONY: seed
seed:
	go run ./cmd/seed

.PHONY: e2e_seed
e2e_seed:
	go run ./cmd/e2e_seed

.PHONY: e2e_seed_ci
e2e_seed_ci:
	go run ./cmd/e2e_seed --envFile=.env.e2e

.PHONY: build_docker_server
build_docker_server:
	cd .. && docker build -f ./backend/Dockerfile.server -t mailboxsweep/mailboxsweep .

.PHONY: build_docker_worker
build_docker_worker:
	cd .. && docker build -f ./backend/Dockerfile.worker -t mailboxsweep/worker .

# Build an image to run migrations
.PHONY: build_docker_migrate
build_docker_migrate:
	cd .. && docker build -f ./backend/Dockerfile.migrate -t mailboxsweep/migrate .

.PHONY: start_e2e_server
start_e2e_server:
	go run ./cmd/server/ --envFile=".env.e2e" --ssmHost=localhost --ssmPort=4566 &

# Maddy Email Test Server Image Directives
# Config
MADDY_VERSION = 0.7
MADDY_DATA_DIR = ./.docker-dev/volumes/maddy
MADDY_VOLUMES = $(MADDY_DATA_DIR):/data
MADDY_TESTUSER = testuser@mailboxsweep.local
MADDY_TESTPASS = pass123
MADDY_VERSION_COMMIT = de756c8dc52b69efa511c8340aa29af76c407f93
MADDY_CONFIG_DEFAULT_URL = https://raw.githubusercontent.com/foxcpp/maddy/$(MADDY_VERSION_COMMIT)/maddy.conf

.PHONY: maddy_make_tls_dir
maddy_make_tls_dir:
	mkdir -p $(MADDY_DATA_DIR)/tls

.PHONY: maddy_init_config
maddy_init_config:
	cp ./.docker-dev/maddy.conf ./.docker-dev/volumes/maddy/maddy.conf

.PHONY: maddy_create_user_smtp
maddy_create_user_smtp:
	docker run --rm -i -v $(MADDY_VOLUMES) -- foxcpp/maddy:$(MADDY_VERSION) creds create --password=$(MADDY_TESTPASS) testuser@mailboxsweep.local

.PHONY: maddy_create_user_imap
maddy_create_user_imap:
	docker run --rm -i -v $(MADDY_VOLUMES) -- foxcpp/maddy:$(MADDY_VERSION) imap-acct create testuser@mailboxsweep.local

.PHONY: maddy_creds
maddy_creds: maddy_create_user_smtp maddy_create_user_imap

.PHONY: maddy_gen_certs
maddy_gen_certs:
	openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -subj "/C=US/ST=NV/L=Las Vegas/O=Technology/OU=Development/CN=mailboxsweep.local" -keyout $(MADDY_DATA_DIR)/tls/privkey.pem -out $(MADDY_DATA_DIR)/tls/fullchain.pem

.PHONY: maddy_init_setup
maddy_init_setup: maddy_make_tls_dir maddy_init_config maddy_gen_certs up maddy_creds

.PHONY: maddy_init_setup_ci
maddy_init_setup: maddy_make_tls_dir maddy_init_config maddy_gen_certs up_ci maddy_creds

.PHONY: up_pentest
up_pentest:
	docker compose -f ./.docker-dev/docker-compose-pentest.yml up -d --wait