.PHONY: all build test verify experiments plots docker-build docker-up docker-down clean

all: verify

build:
	go build -o bin/sovereign-node ./cmd/server

test:
	go test -race ./testing/simulation/...

verify:
	go mod tidy && ./deploy/scripts/verify.sh

experiments:
	go run ./cmd/experiment/runner.go

plots:
	python3 ./scripts/generate_plots.py

paper-artifacts: experiments plots
	@echo "Research artifacts generated successfully in results/"

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/ results/json/* results/plots/*
