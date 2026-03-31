# Makefile for Vortex Load Balancer
.PHONY: build run clean docker-build docker-run

APP_NAME = vortex
DOCKER_IMAGE = vortex-lb
DOCKER_TAG = latest

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

docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	@echo "Running Vortex in Docker container..."
	docker run -p 8000:8000 --name $(APP_NAME) --rm $(DOCKER_IMAGE):$(DOCKER_TAG)
