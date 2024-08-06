.PHONY: deps
deps:
	go mod tidy
	go mod verify

vulnerabilities_lookup:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck -test ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
		run --allow-parallel-runners -c .golangci.yml

.PHONY: gci
gci:
	go run github.com/luw2007/gci@latest \
		write . --skip-generated
