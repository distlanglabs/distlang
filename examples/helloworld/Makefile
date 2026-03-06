DISTLANG := ../../bin/distlang
ROOT := ../..

.PHONY: help cli cloudflare-init run debug build cloudflare-deps dev deploy

help:
	@echo "Targets:"
	@echo "  make cloudflare-init - create local target scaffolding"
	@echo "  make run    - run worker locally on port 5656"
	@echo "  make debug  - inspect parse pass output"
	@echo "  make build  - build goja + cloudflare artifacts"
	@echo "  make cloudflare-deps - install/check wrangler via dist/cloudflare"
	@echo "  make dev    - run wrangler dev via dist/cloudflare"
	@echo "  make deploy - route deploy via dist/cloudflare Makefile"

cli:
	@$(MAKE) -C $(ROOT) build

cloudflare-init: cli
	$(DISTLANG) target init --target=cloudflare --path=.

run: cli
	$(DISTLANG) run index.js --port=5656

debug: cli
	$(DISTLANG) debug build index.js --passes=parse

build: cli
	$(DISTLANG) build index.js

cloudflare-deps: build
	$(MAKE) -C dist/cloudflare deps

dev: build
	$(MAKE) -C dist/cloudflare run

deploy: build
	@test -f targets/cloudflare/cloudflare.env || (echo "missing targets/cloudflare/cloudflare.env; run 'make cloudflare-init' first" && exit 1)
	@set -a; . targets/cloudflare/cloudflare.env; set +a; $(MAKE) -C dist/cloudflare publish
