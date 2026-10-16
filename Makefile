.PHONY: test build lint-chart image

test:
	go test ./...

build:
	go build -o bin/trips-api ./cmd/trips-api

lint-chart:
	helm lint charts/trips-api
	helm template trips-api charts/trips-api >/dev/null
	helm template trips-api charts/trips-api --set ingress.enabled=true >/dev/null
	helm template trips-api charts/trips-api --set postgresql.enabled=false --set database.host=db.example >/dev/null

image:
	docker build -t trips-api:local .
