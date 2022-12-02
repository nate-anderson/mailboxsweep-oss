.PHONY: up
up:
	docker-compose -f ./.docker-dev/docker-compose.yml up -d

.PHONY: down
down:
	docker-compose -f ./.docker-dev/docker-compose.yml down

.PHONY: install_openapi_gen
install_openapi_gen:
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest

# Greenmail testcontainer stuff
GREENMAIL_RELEASE_HASH = 7dcc78cdfa258e57e2f931f4c610be55ae2c567a
GREENMAIL_SPEC_URL = https://raw.githubusercontent.com/greenmail-mail-test/greenmail/$(GREENMAIL_RELEASE_HASH)/greenmail-standalone/src/main/resources/greenmail-openapi.yml
GREENMAIL_SPEC_LOCAL_PATH = pkg/clients/greenmail/openapi.yml
GREENMAIL_CLIENT_GEN_PATH = pkg/clients/greenmail/greenmail.gen.go
GREENMAIL_OAPI_CODEGEN_CONFIG_PATH = pkg/clients/greenmail/codegen.yml

.PHONY: greenmail_fetch_spec
greenmail_fetch_spec:
	curl $(GREENMAIL_SPEC_URL) --output $(GREENMAIL_SPEC_LOCAL_PATH)

.PHONY: greenmail_gen_client
greenmail_gen_client:
	oapi-codegen -config $(GREENMAIL_OAPI_CODEGEN_CONFIG_PATH) $(GREENMAIL_SPEC_LOCAL_PATH)

# HTTP API server codegen stuff
HTTP_SERVER_SPEC_PATH = openapi.yml
HTTP_SERVER_GEN_PATH = http_codegen.yml

.PHONY: http_scaffold
http_scaffold:
	oapi-codegen -generate chi-server,types -config $(HTTP_SERVER_GEN_PATH) $(HTTP_SERVER_SPEC_PATH)