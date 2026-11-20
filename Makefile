IMAGE ?= registry.sion2k.ru/shturval/shturval-alert-wall:latest
.PHONY: test lint build docker-build helm-template run smoke-test
test:
	cd web && npm install && npm test
	cd web && npm run build
	go test ./...
lint:
	go vet ./...
build: test
	go build -o bin/server ./cmd/server
docker-build:
	docker build -t $(IMAGE) .
helm-template:
	helm template alert-wall charts/shturval-alert-wall
run:
	SOURCE_MODE=fake FAKE_ALERTS_PATH=config/samples/fake-alerts.json go run ./cmd/server
smoke-test:
	./scripts/smoke-test.sh