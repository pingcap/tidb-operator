# HTTP Service

An HTTP Service to convert Kubernetes APIs into HTTP APIs, so that some users can use HTTP clients to manage TiDB clusters and backups & restores directly.

## Development

### OpenAPI

An OpenAPI Swagger file is generated in the `pbgen/oas` directory, you can view this file via some Swagger tools or update the file to [Redoc](https://redocly.github.io/redoc/) and then view in the web UI directly.

### Generate gRPC stubs and OpenAPI spec

run `make buf-generate`

### Build

run `make build`

## Run

- When running the binary out of a Kubernetes cluster, the `--kubeconfig` must be set to a KUBECONFIG file path.
- If the Kubernetes cluster does not have enough resources, set `LOCAL_RUN=true` environment variable when running the binary.
  - This will let this HTTP Service to remove the CPU & memory requests for components so that Pods can be scheduled.

## Test

There are some JSON files in the `examples` directory which can be used as the HTTP body when testing.
