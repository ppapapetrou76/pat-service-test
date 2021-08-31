ifndef UNIQUE_BUILD_ID
	UNIQUE_BUILD_ID=latest
endif
UNIQUE_BUILD_ID_LOWER=$(shell echo $(UNIQUE_BUILD_ID) | tr A-Z a-z)

.SILENT: help
help: ## Show this help message
	set -x
	echo "Usage: make [target] ..."
	echo ""
	echo "Available targets:"
	grep ':.* ##\ ' ${MAKEFILE_LIST} | awk '{gsub(":[^#]*##","\t"); print}' | column -t -c 2 -s $$'\t' | sort

DOCKER_IMAGE_TOOLS=healthcheck-tools:latest
DOCKER_RUN_TOOLS=docker run --rm -v "$(PWD):/app" $(DOCKER_IMAGE_TOOLS)
DOCKER_COMPOSE_TOOLS=docker-compose -p '${UNIQUE_BUILD_ID_LOWER}' -f ./docker/tools/docker-compose.yml

tools-build: ## Build docker tools image
	docker build -t $(DOCKER_IMAGE_TOOLS) -f docker/tools/Dockerfile .

test: tools-build ## Run the tests
	docker-compose version

	${DOCKER_COMPOSE_TOOLS} up -d && \
	echo '\nRunning tests in the docker container...' && \
	docker run --rm -v "$(PWD):/app" \
		--network='${UNIQUE_BUILD_ID_LOWER}_testing_network' \
		-e TEST_RDS_URL=postgres://postgres:password@db:5432/postgres?sslmode=disable \
		$(DOCKER_IMAGE_TOOLS) /bin/bash -c "go test -timeout 10m -race ./... ${ARGS}"; \
	RESULT=$$?; \
	${DOCKER_COMPOSE_TOOLS} down --volumes; \
	exit $$RESULT

lint: tools-build ## Run golangci-lint and go vet
	$(DOCKER_RUN_TOOLS) bash -c "golangci-lint version && golangci-lint run && echo 'Passed the golangci-lint checks'"

check-format: tools-build ## Check the format of the project
	$(DOCKER_RUN_TOOLS) ./scripts/check-format-go.sh
	$(DOCKER_RUN_TOOLS) ./scripts/check-format-markdown.sh

checks: check-format lint test ## Run all checks
