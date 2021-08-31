# Template GRPC Service

Template grpc service is an internal service that - change accordingly

- [Development Tooling](#development-tooling)
- [Development Tips](#development-tips)
  - [Working with the Shared libs docker image](#downloading-the-shared-docker-image-to-run-dev-tooling)
  - [Local DB containers](#manage-local-docker-db-containers-for-development-purposes)
- [Monitoring](#monitoring)
- [Making local grpc calls](#making-local-grcp-calls)
  - [Pre-requisites](#pre-requisites)
  - [List endpoints](#get-a-list-of-available-endpoints)
  - [Port forwarding](#port-forwarding-of-a-running-env-in-k8s)
- [Dashboards](#operational-dashboards)  

## Development Tooling

Run `make` to see the list of available [make targets](Makefile). The output will look similar to the following.

```text
## Development targets

test:                 Runs the tests
test-local:           Runs the tests locally ( no docker container )
add-package:          Adds a package
ensure-package:       Installs required dependencies
lint:                 Runs linters
format:               Formats code
check-format:         Checks the format of go code
generate:             Generates mocks files and template files from proto files
check-up-to-date:     Checks the generated code is up-to-date
start-db:             Starts a postgres DB docker container under the name `db` exposing the port 5432 to the host.
stop-db:              Stops the postgres DB docker container named `db`.
start:                Starts the service locally ( single docker container )

## Build targets

go-build:             Builds the go binary
docker-build:         Builds the docker app image
docker-clean:         Removes the latest docker app image

```

There are three main categories.

The development targets contain tooling that can be used in day-to-day development activities such as running tests,
formatting code, running linters etc.

All targets run inside a docker container.

Note: You might need to authenticate the Docker CLI to access Sliide AWS ECR registry before using make commands, see [this section](#downloading-the-shared-docker-image-to-run-dev-tooling).

## Development tips

Please make sure that before submitting a PR you have run locally the following:

```shell
make pre-commit
```

### Downloading the shared docker image to run dev tooling

A guide for downloading the shared docker image can be found [here](https://sliide.atlassian.net/wiki/spaces/BE/pages/2018803790/).

### Manage local Docker DB containers for development purposes

Sometimes a DB container is needed to run the template-grpc-service  
using the favorite IDE for debug and tracing purposes. There are a couple of make targets that come to rescue.

#### Start a local DB docker container

Run ```make start-db``` to start a DB container exposing the postgres DB to the host on port 5432

Once the container is up and running, the following env variable need to be set in order to run the service.

- `RDS_URL`=`postgres://postgres:tests@localhost:5432/postgres?sslmode=disable`

#### Stop the local DB docker container

Run ```make stop-db``` to stop the running db docker container.

## Monitoring

Health check endpoint:

```sh
$ curl http://localhost:2112/healthcheck
{"checks":[{"state":"healthy","output":"Daemon is serving","name":"http server","duration":306171},{"state":"healthy","output":"Skip check because of no tables given","name":"template-grpc database","duration":211476}],"duration":408819,"runtime":{"host":"Patrokloss-MBP","go_version":"go1.16.5","service":"template-grpc-service","environment":"dev","version":""},"duration_in_seconds":0.000408819,"is_healthy":true,"is_degraded":false}
```

Service ready endpoint, designed for k8s controller's checking:

```sh
$ curl -i http://localhost:2112/ready
HTTP/1.1 200 OK
Date: Sun, 02 Feb 2020 20:20:20 GMT
Content-Length: 0
```

Prometheus metrics endpoint:

```sh
$ curl http://localhost:2112/metrics
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 7.0154e-05
go_gc_duration_seconds{quantile="0.25"} 7.6315e-05
go_gc_duration_seconds{quantile="0.5"} 8.368e-05
go_gc_duration_seconds{quantile="0.75"} 0.000119571
go_gc_duration_seconds{quantile="1"} 0.000156717
go_gc_duration_seconds_sum 0.000750696
go_gc_duration_seconds_count 8
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 13
...
```

Profiling check endpoint:

```sh
open "http://localhost:6060/debug/pprof/"
```

## Making local grcp calls

To locally validate that the grpc service is working as expected you can follow the guide below.

### Pre-requisites

We need a CLI to access the grpc endpoints. One great option is to use `grpcurl`
To install it locally, if you are on a Mac machine simply run `brew install grpcurl` or follow the [installation
instructions](https://github.com/fullstorydev/grpcurl#installation)

### Get a list of available endpoints

_Bear in mind that we don't support TLS locally, so we need to use the `-plaintext` option in every `grpcurl` command
we are executing.

Once you have started the server locally run

```shell
grpcurl -plaintext localhost:8080 list
```

Update this section after implementing the service endpoints

You should see an output very similar to this:

```text
grpc.examples.echo.Echo
grpc.reflection.v1alpha.ServerReflection
```

### Show the available `rpc`

Update this section after implementing the service endpoints

### Port forwarding of a running env in K8s

TIP: sometimes we need to do some validation against a live environment (dev or staging). If you have K8s access you
can run the following command to forward all the traffic of port `8080` of the template-grpc-service to your local port `8080`

```shell
kubectl -n dev port-forward $(kubectl -n dev get pods |grep template-grpc-service |head -n1 |awk '{print $1}') 8080:8080
```

## Operational Dashboards

The following dashboards are available to monitor operational status.

- [Grafana dashboard] : Metrics about CPU, Goroutines etc.
- [Kibana dashboard] : Logging reports and graphs.
