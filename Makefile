# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get

# Main package name
MAIN_PACKAGE = main

# Output binary name
BINARY_NAME = plandex

# Check the PLANDEX_ENVIRONMENT environment variable, reassign the BINARY_NAME if necessary
ifeq ($(PLANDEX_ENV),development)
BINARY_NAME = plandex-dev
endif

# create a dev cmd that runs a shell script
dev:
	@cd app/scripts && ./dev.sh

# Build target
build:
	@$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

# Clean target
clean:
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)

# Test target
test: render
	@$(GOTEST) -v ./...

render:
	@cd app/scripts && go run render_config.go

# Get dependencies
deps:
	$(GOGET) -v ./...

# Default target
default: build

.PHONY: all render build clean test deps