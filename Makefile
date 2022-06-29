build:
	CGO_ENABLED=0 go build -o bin/vault-operator -ldflags "-s -w" .

docker:
	docker build -t tuananh/vault-operator .
	
generate:
	go generate

validate:
	golangci-lint --timeout 5m run

validate-ci: generate
	go mod tidy
	if [ -n "$$(git status --porcelain)" ]; then \
		git status --porcelain; \
		echo "Encountered dirty repo!"; \
		exit 1 \
	;fi

test:
	go test ./...

goreleaser:
	goreleaser build --snapshot --single-target --rm-dist

setup-ci-env:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.46.2