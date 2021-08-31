# Must use the digests for the security reason (immutable) in the product environment
# see https://success.docker.com/article/images-tagging-vs-digests
# golang:1.16.5
ARG GOLANG_DIGEST=sha256:360bc82ac2b24e9ab6e5867eebac780920b92175bb2e9e1952dce15571699baa
# alpine 3.13.1
ARG ALPINE_DIGEST=sha256:3747d4eb5e7f0825d54c8e80452f1e245e24bd715972c919d189a62da97af2ae

FROM golang@${GOLANG_DIGEST} as builder

ARG PIPELINE_VERSION

RUN mkdir /app && chmod a+rwx /app
COPY . /app
WORKDIR /app

RUN go version

RUN export GIT_REVISION=$(git rev-parse HEAD) && \
    export GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD) && \
    export CGO_ENABLED=0 && \
    export GOOS=linux && \
    printenv | sort && \
    go build -mod=vendor -tags netgo \
    -ldflags '\
    -X "main.Version='"${PIPELINE_VERSION}"'" \
    -X "main.GitRevision='"${GIT_REVISION}"'" \
    -X "main.GitBranch='"${GIT_BRANCH}"'" \
    ' -o main

FROM alpine@${ALPINE_DIGEST}

WORKDIR /app

COPY --from=builder "/app/main" "/app/"

# Install certificates, otherwise get an error `x509: certificate signed by unknown authority`
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN adduser --disabled-password --uid 1234 www-data
USER www-data

# Main
EXPOSE 8080

# Profiling and Monitoring
EXPOSE 6060 2112

CMD ["/bin/sh", "-c", "/app/main"]
