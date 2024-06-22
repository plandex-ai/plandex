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

#### Evals and Providers ####


#! No cache is used to ensure that the latest changes are reflected in the eval
# TODO: Implement eval all
eval:
	@cd test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS)) && promptfoo eval --no-cache

view-eval:
	@cd test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS)) && promptfoo view

gen-eval:
	@$(GOCMD) run app/scripts/cmd/gen/gen.go test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS))

gen-provider:
	@$(GOCMD) run app/scripts/cmd/provider/gen_provider.go

#### End Evals and Providers ####

# Get dependencies
deps:
	$(GOGET) -v ./...

# Default target
default: build

# Usage
help:
	@echo "Usage:"
	@echo "  make dev - to run the development scripts"
	@echo "  make eval <directory_name> - to run the promptfoo eval command on a specific directory"
	@echo "  make view-eval - to view the promptfoo eval output"
	@echo "  make gen-eval <directory_name> - to create a new promptfoo eval directory structure"
	@echo "  make gen-provider - to create a new promptfoo provider file from the promptfoo diretory structure"
	@echo "  make clean - to remove generated files and directories"
	@echo "  make help - to display this help message"

# Prevents make from interpreting the arguments as targets
%:
	@:

.PHONY: all render build clean test deps