.PHONY: build test docker lint helm-template

VERSION ?= dev
IMAGE ?= ghcr.io/denislemire/apsystems-prometheus-exporter

build:
	go build -ldflags="-s -w -X github.com/denislemire/apsystems-prometheus-exporter/internal/exporter.Version=$(VERSION)" -o bin/apsystems-exporter ./cmd/apsystems-exporter

test:
	go test ./...

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

helm-template:
	helm template apsystems ./helm/apsystems-exporter \
		--set apsystems.sid=EXAMPLE \
		--set apsystems.ecuId=EXAMPLE \
		--set apsystems.createSecret=true \
		--set apsystems.appId=example \
		--set apsystems.appSecret=example

lint:
	go vet ./...
