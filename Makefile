.PHONY: docker-up docker-down build paper-artifacts

build:
	go build -o build/sovereign-node ./cmd/sovereign

docker-up:
	@echo "Booting 10-Node Sovereign Mesh..."
	docker-compose up -d --build

docker-down:
	@echo "Tearing down Sovereign Mesh..."
	docker-compose down

paper-artifacts:
	@echo "Generating IEEE publication artifacts..."
	python3 ./scripts/generate_plots.py
