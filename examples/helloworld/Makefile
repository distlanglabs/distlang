DISTLANG := ../../bin/distlang
ROOT := ../..

.PHONY: help cli target-init run debug build deploy

help:
	@echo "Targets:"
	@echo "  make target-init - create local target scaffolding"
	@echo "  make run    - run worker locally on port 5656"
	@echo "  make debug  - inspect parse pass output"
	@echo "  make build  - build goja + cloudflare artifacts"
	@echo "  make deploy - deploy to cloudflare"

cli:
	@$(MAKE) -C $(ROOT) build

target-init: cli
	$(DISTLANG) target init --target=cloudflare --path=.

run: cli
	$(DISTLANG) run index.js --port=5656

debug: cli
	$(DISTLANG) debug build index.js --passes=parse

build: cli
	$(DISTLANG) build index.js

deploy: cli
	$(DISTLANG) deploy index.js --target=cloudflare
