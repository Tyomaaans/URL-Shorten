.PHONY: swagger build test vet

swagger:
	swag init -g doc.go -d ./internal/url-shortener,./pkg -o ./docs/url-shortener --ot go,json,yaml --instanceName URLShortener --parseDependency --parseInternal --parseGoList

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...
