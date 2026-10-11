GOLIB ?= golib

.PHONY: check check-full ci cohesion config docs-local inventory repository-check specification-check specification-check-online workflows

check: docs-local
	$(GOLIB) check --local --all

check-full: specification-check-online
	$(GOLIB) check --all

ci: config inventory cohesion repository-check specification-check workflows check

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
