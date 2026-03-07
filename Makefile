test:
	go test ./... -v

run:
	go run main.go

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

install-lint:
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.60.0
	@echo "golangci-lint installed successfully!"

buf-gen:
	cd presentation/grpc && buf dep update && buf generate

.PHONY: test run lint lint-fix install-lint buf-gen