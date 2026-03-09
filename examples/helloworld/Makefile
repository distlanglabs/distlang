DISTLANG := ../../bin/distlang
ROOT := ../..

.PHONY: help cli deps cloudflare-init run debug build cloudflare-deps dev deploy

help:
	@echo "Targets:"
	@echo "  make deps   - install/check local workerd"
	@echo "  make cloudflare-init - create local target scaffolding"
	@echo "  make run    - run local v8 runtime"
	@echo "  make debug  - inspect parse pass output"
	@echo "  make build  - build v8 backend and cloudflare package"
	@echo "  make cloudflare-deps - install/check wrangler via dist/cloudflare"
	@echo "  make dev    - run wrangler dev via dist/cloudflare"
	@echo "  make deploy - route deploy via dist/cloudflare Makefile"

cli:
	@$(MAKE) -C $(ROOT) build

deps:
	@command -v workerd >/dev/null 2>&1 && echo "workerd already installed" && exit 0 || true
	@command -v npm >/dev/null 2>&1 || (echo "npm not installed; required to install workerd" && exit 1)
	@npm install -g workerd

cloudflare-init: cli
	$(DISTLANG) target init --target=cloudflare --path=.

run: cli deps
	$(DISTLANG) run index.js --v8-port=5656

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
