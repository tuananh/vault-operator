build:
	CGO_ENABLED=0 go build -o bin/vault-operator -ldflags "-s -w" .

run: build
	./bin/vault-operator
		
docker:
	docker build -t tuananh/vault-operator .

codegen: ## Generate code. Must be run if changes are made to ./pkg/apis/...
	controller-gen \
		object:headerFile="hack/boilerplate.go.txt" \
		crd \
		paths="./pkg/apis/..." \
		output:crd:artifacts:config=config/crd
	hack/boilerplate.sh

validate:
	golangci-lint --timeout 5m run

validate-ci: codegen
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
	./hack/tools.sh