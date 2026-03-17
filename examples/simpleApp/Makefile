DISTLANG := ../../bin/distlang
ROOT := ../..

.PHONY: help cli deps cloudflare-init run run-set1 run-set2 debug build cloudflare-deps dev dev-set1 dev-set2 deploy

help:
	@echo "Targets:"
	@echo "  make deps           - install/check local workerd"
	@echo "  make cloudflare-init - create local Cloudflare target scaffolding"
	@echo "  make run            - run both handler sets locally (5656 + 5657)"
	@echo "  make run-set1       - run only handlerSet1 locally"
	@echo "  make run-set2       - run only handlerSet2 locally"
	@echo "  make debug          - inspect parse pass output"
	@echo "  make build          - build both v8 handler set artifacts"
	@echo "  make cloudflare-deps - install/check wrangler for both Cloudflare worker dirs"
	@echo "  make dev            - run wrangler dev for handlerSet1"
	@echo "  make dev-set2       - run wrangler dev for handlerSet2"
	@echo "  make deploy         - deploy both handler sets via distlang deploy"

cli:
	@$(MAKE) -C $(ROOT) build

deps:
	@if command -v workerd >/dev/null 2>&1; then \
		echo "workerd already installed"; \
	else \
		command -v npm >/dev/null 2>&1 || (echo "npm not installed; required to install workerd" && exit 1); \
		npm install -g workerd; \
	fi

cloudflare-init: cli
	@$(MAKE) -f Makefile.Cloudflare init

run: cli deps
	$(DISTLANG) run index.js --set=all --port1=5656 --port2=5657

run-set1: cli deps
	$(DISTLANG) run index.js --set=handlerSet1 --port1=5656

run-set2: cli deps
	$(DISTLANG) run index.js --set=handlerSet2 --port2=5657

debug: cli
	$(DISTLANG) debug build index.js --passes=parse

build: cli
	$(DISTLANG) build index.js

cloudflare-deps: build
	@$(MAKE) -f Makefile.Cloudflare deps

dev: build
	@$(MAKE) -f Makefile.Cloudflare dev

dev-set1: build
	@$(MAKE) -f Makefile.Cloudflare dev-set1

dev-set2: build
	@$(MAKE) -f Makefile.Cloudflare dev-set2

deploy: build
	@$(MAKE) -f Makefile.Cloudflare deploy
