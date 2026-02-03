.PHONY: all build run clean test install

# Binary name
BINARY_NAME=usermgr
BUILD_DIR=build
CMD_DIR=cmd/usermgr

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy

install:
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	$(GOBUILD) -o /usr/local/bin/$(BINARY_NAME) ./$(CMD_DIR)

# Docker targets
docker-build:
	docker build -t devbase:latest .

docker-run:
	docker run -d -p 8080:8080 -p 2222:2222 --name devbase devbase:latest

docker-stop:
	docker stop devbase || true
	docker rm devbase || true
