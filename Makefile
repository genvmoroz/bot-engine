include ./linting.mk

.PHONY: deps
deps:
	go mod tidy
	go mod download
	go mod vendor
	go mod verify
