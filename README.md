# Blob Storage Gateway

The codebase defines the `Gateway` to distribute Read and Write operations among the Minio Object Storage instances.

## Module Design

```mermaid
---
title: The Gateway design diagram
---
classDiagram
    class Gateway {
        // pkg/gateway/gateway.go

        -cfg                 *Config
        -logger              *slog.Logger

        +Read(ctx context.Context, id string) io.ReadCloser, bool, error
        +Write(ctx context.Context, id string, reader io.Reader) error
    }


    class Config {
        // pkg/gateway/config.go
        +StorageInstancesSelector string
        +DefaultBucket string
        +StorageInstancesFinder StorageInstancesFinder
        +StorageConnectionDetailsReader StorageConnectionDetailsReader
        +NewStorageConnectionFn StorageConnectionFactory
        +Logger                         *slog.Logger
    }

    class StorageConnectionDetailsReader {
        // pkg/gateway/config.go
        <<Interface>>
        Read(ctx context.Context, id string) string, string, string, error
    }

    class StorageInstancesFinder {
        // pkg/gateway/config.go
        <<Interface>>
        Find(ctx context.Context, instanceNameFilter string) map[string]struct, error
    }

    class StorageController {
        // pkg/gateway/config.go
        <<interface>>
        Read(ctx context.Context, bucketName, objectName string) io.ReadCloser, bool, error
        Write(ctx context.Context, bucketName, objectName string, reader io.Reader) error
        Detected(ctx context.Context, bucketName, objectName string) bool, error
    }

    class StorageConnectionFactory {
        // pkg/gateway/config.go
        <<interface>>
        func(endpoint, accessKeyID, secretAccessKey string) StorageController, error
    }

    class Handler {
        // pkg/restfulhandler/handler.go
        -rw readWriter
        -commonRoutePrefix string
        -logger            *slog.Logger
        +ServeHTTP(w http.ResponseWriter, r *http.Request)
        -logError(r *http.Request, statusCode int, msg string)
        -knownRoute(p string) bool
        -readObjectID(p string) string
    }

    class readWriter {
        // pkg/restfulhandler/handler.go
        <<interface>>
        +Read()
        +Write()
    }

    class dockerClient {
// internal/docker/docker.go
*"github.com/docker/docker".Client
}

class minioClient {
// internal/minio/minio.go
*"github.com/minio/minio-go/v7".Client
}

class NewClient {
// internal/minio/minio.go
func"internal/minio.NewClient"
    }

StorageConnectionFactory"1"-->"*"StorageController

minioClient --|> StorageController
dockerClient --|> StorageConnectionDetailsReader
dockerClient --|> StorageInstancesFinder

NewClient --|>StorageConnectionFactory

Config *--dockerClient
Config *--NewClient

Handler <|-- readWriter
Gateway --|> readWriter
Gateway *-- Config
Handler *-- Config: Logger
```

## Gateway Deployed as Restful HTTP WebServer

### Endpoints

See the endpoints definition in the [spec file](pkg/restfulhandler/apispec.yaml).

## How to run

### Prerequisites

- Docker 23+
- docker-compose

### Requirements

- The Minio Cluster Instances and the Gateway must run as Docker processes.
- The Gateway must share the network with the Minio Cluster.
- The Gateway mush have access to the socket `/var/run/docker.sock` to communicate to the Docker daemon over HTTP.

Run to provision a setup with three Minio instance and a Gateway instance:

```
docker-compose up --build
```

## Commands

_Requires_ gnuMake/cmake

- See help:

```commandline
make help
```

- Run unit tests:

```commandline
make tests
```

- Run e2e tests:

```commandline
make e2etests
```

**Note**: the command requires `curl` and `wc`.

- Run linters:

```commandline
make lint
```

**Note**: the command requires `Docker`.

## Env Variables Configurations

The Gateway process can be configured using the environment variables listed in the table.

| Variable Name              | Definition                         | Default                      |
|:---------------------------|:-----------------------------------|:-----------------------------|
| STORAGE_INSTANCES_SELECTOR | Selector to identify storage nodes | "amazin-object-storage-node" |
| PORT                       | Port for the webserver to listen   | 3000                         |
| LOG_DEBUG                  | Logger's debug verbosity level     | true                         |

## License

The codebase present in the repository is distributed under the [MIT license](LICENSE).

The images and graphical material is distributed under
the [CC BY-NC-SA 4.0 DEED](https://creativecommons.org/licenses/by-nc-sa/4.0/) license.

## Disclaimer

The project was developed as a solution addressing the problem
described [here](https://github.com/spacelift-io/homework-object-storage).
