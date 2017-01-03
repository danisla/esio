SHELL := /bin/bash

APP_TITLE := Elasticsearch Snapshot Index Orchestrator
APP_DESC := Orchestrates the recovery and cleanup of Elasticsearch index snapshots.
APP_VER := 1.0.0
APP_SCHEME := http
APP_CONSUMES := application/com.github.danisla.esio.v1+json
APP_PRODUCES := application/com.github.danisla.esio.v1+json
APP_MODEL := Esio
APP_HOST := 127.0.0.1
APP_PORT := 8000

APP_CMD := esio-server

SPEC := swagger.yml
JSON_SPEC := $(subst .yml,.json,$(SPEC))

ES_HOST ?= http://localhost:9200
INDEX_RESOLUTION ?= day
REPO_PATTERN ?= test/test-%Y_%m/test-v1-%j

PID := .server.PID

all: run

run: compile
	$(APP_CMD) --host $(APP_HOST) --port $(APP_PORT) \
		--es-host $(ES_HOST) \
		--resolution $(INDEX_RESOLUTION) \
		--repo-pattern $(REPO_PATTERN)

compile: validate
	@if [[ "$${GOGET:-true}" == "true" ]]; then echo "go get ./..." ; go get ./...; else echo "Skipping go get"; fi
	go install ./cmd/$(APP_CMD)

$(SPEC):
	swagger init spec \
	  --title "$(APP_TITLE)" \
	  --description "$(APP_DESC)" \
	  --version $(APP_VER) \
	  --scheme $(APP_SCHEME) \
	  --consumes $(APP_CONSUMES) \
	  --produces $(APP_PRODUCES)

$(JSON_SPEC): $(SPEC)
	swagger generate spec -i $^ -o $@

docs: $(SPEC)
	swagger serve --host 127.0.0.1 --port 8001 -F swagger $(SPEC)

validate: $(SPEC)
	swagger validate $(SPEC)

gen: validate
	swagger generate server -A $(APP_MODEL) -f $(SPEC)

start-server: $(PID)

wait-server:
	@while [[ ! `curl -sf http://$(APP_HOST):$(APP_PORT)/healthz` ]]; do sleep 5; done ; \
	echo "Server is running, pid: `cat $(PID)`"

$(PID):
	make run & echo $$! > $@

stop-server: $(PID)
	-kill `cat $<`
	rm -f $<

restart-server:
	make stop-server
	make start-server
	make wait-server

clean: stop-server stop-elastic

include elastic.mk
include tests.mk
