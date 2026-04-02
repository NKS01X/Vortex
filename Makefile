# Makefile for Vortex Load Balancer
.PHONY: build run clean docker-build docker-run up down logs

APP_NAME = vortex
DOCKER_IMAGE = vortex-lb
DOCKER_TAG = latest

## LOCAL DEV TARGETS ##

build:
	@echo "Building Vortex binary..."
	go build -o $(APP_NAME) main.go

run: build
	@echo "Running Vortex..."
	./$(APP_NAME)

clean:
	@echo "Cleaning up..."
	rm -f $(APP_NAME)
	-docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

## STANDALONE DOCKER TARGETS ##

docker-build:
	@echo "Building Docker image..."
	docker build --target prod -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	@echo "Running Vortex in standalone Docker container..."
	docker run -p 8000:8000 --name $(APP_NAME) --rm $(DOCKER_IMAGE):$(DOCKER_TAG)

## OBSERVABILITY CLUSTER TARGETS ##

up:
	@echo "Spinning up Vortex Cluster (Standard Start)..."
	TARGET=prod docker compose up -d

dev:
	@echo "Spinning up Vortex Cluster for Development (Hot-Reloading with Air)..."
	TARGET=dev docker compose up -d --build

rebuild:
	@echo "Rebuilding and starting Vortex Cluster..."
	TARGET=prod docker compose up -d --build

down:
	@echo "Tearing down the Vortex Cluster..."
	docker compose down

logs:
	@echo "Fetching logs for Vortex Cluster..."
	docker compose logs -f
