ifndef UNIQUE_BUILD_ID
	UNIQUE_BUILD_ID=latest
endif

ifndef JENKINS_HOME
	DOCKER_IMAGE_TOOLS=410715645895.dkr.ecr.us-east-1.amazonaws.com/go-shared-tools
	DOCKER_RUN_TOOLS=docker run --rm -v "$(PWD):/app" --net=host -v /var/run/docker.sock:/var/run/docker.sock $(DOCKER_IMAGE_TOOLS)
else
	# We are running in the Jenkins' docker-container,
	# which mean doesn't require launching another environment for the operations.
	DOCKER_RUN_TOOLS=
endif

PROJECT_NAME=template-grpc-service
DOCKER_IMAGE_APP=$(PROJECT_NAME):$(UNIQUE_BUILD_ID)

include scripts/Makefile.help
include scripts/Makefile.dev
include scripts/Makefile.build
