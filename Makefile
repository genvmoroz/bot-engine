include ./scripts/linting.mk

.PHONY: deps
deps:
	go mod tidy
	go mod verify
