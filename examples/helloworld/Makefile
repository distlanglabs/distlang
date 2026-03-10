DISTLANG := ../../bin/distlang
ROOT := ../..

.PHONY: help cli deps cloudflare-init run debug build cloudflare-deps dev deploy

	help:
	@echo "Targets:"
	@echo "  make deps   - install/check local workerd"
	@echo "  make cloudflare-init - create local Cloudflare target scaffolding"
	@echo "  make run    - run local v8 runtime"
	@echo "  make debug  - inspect parse pass output"
	@echo "  make build  - build the v8 backend"
	@echo "  make cloudflare-deps - install/check wrangler via Makefile.Cloudflare"
	@echo "  make dev    - run wrangler dev via Makefile.Cloudflare"
	@echo "  make deploy - route deploy via Makefile.Cloudflare"

cli:
	@$(MAKE) -C $(ROOT) build

deps:
	@command -v workerd >/dev/null 2>&1 && echo "workerd already installed" && exit 0 || true
	@command -v npm >/dev/null 2>&1 || (echo "npm not installed; required to install workerd" && exit 1)
	@npm install -g workerd

cloudflare-init: cli
	@$(MAKE) -f Makefile.Cloudflare init

run: cli deps
	$(DISTLANG) run index.js --v8-port=5656

debug: cli
	$(DISTLANG) debug build index.js --passes=parse

build: cli
	$(DISTLANG) build index.js

cloudflare-deps: build
	@$(MAKE) -f Makefile.Cloudflare deps

dev: build
	@$(MAKE) -f Makefile.Cloudflare dev

deploy: build
	@$(MAKE) -f Makefile.Cloudflare deploy
