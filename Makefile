.DEFAULT_GOAL := compile

PROTECTED_TRANSFORM_PROXY_PORT ?= 1111

CMD_DIR = $(CURDIR)/cmd

compile:
	go build -o $(CMD_DIR)/protected-transform-proxy $(CMD_DIR)/protected-transform-proxy.go

build-image:
	docker build --build-arg GIT_SHA=$$(git rev-parse HEAD) -t quemot-dev/protected-transform-proxy .

build-dev:
	DISABLE_AUTH=1 docker compose -f ./docker-compose.yml build

run-dev:
	DISABLE_AUTH=1 docker compose -f ./docker-compose.yml up

.PHONY: compile build-image build-dev run-dev
