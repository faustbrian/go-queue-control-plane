GOLIB ?= golib

.PHONY: source-docs-check check check-full ci cohesion config docs-local inventory repository-check specification-check specification-check-online workflows

check: source-docs-check docs-local
	$(GOLIB) check --local --all

check-full: source-docs-check specification-check-online
	$(GOLIB) check --all

ci: source-docs-check config inventory cohesion repository-check specification-check workflows check

cohesion:
	$(GOLIB) cohesion check

config:
	$(GOLIB) config validate

inventory:
	$(GOLIB) inventory

repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check

specification-check-online:
	$(GOLIB) specification check --online

docs-local:
	bash scripts/check-docs.sh --local

workflows:
	$(GOLIB) workflows check

source-docs-check:
	$(GOLIB) source-docs check
