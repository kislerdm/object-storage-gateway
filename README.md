# Object storage gateway

The codebase defines the `sateway` to distribute Read and Write operations among the object storage instances.

### Shortcuts

* [Demo](#demo)
* [How to run](#how-to-run)
* [Useful Commands](#commands)

## Demo

The following instructions explain how to provision the local environment to deploy gateway as a restful webserver.

### How to run

#### Prerequisites

- Docker 23+
- docker-compose

Run to provision a setup with three blob storage [Minio](https://min.io/) instances and a gateway instance:

```
docker-compose up --build
```

<details>
<summary><strong>Env Variables Configurations</strong></summary>

The gateway process can be configured using the environment variables listed in the table.

| Variable Name              | Definition                         | Default                      |
|:---------------------------|:-----------------------------------|:-----------------------------|
| STORAGE_INSTANCES_SELECTOR | Selector to identify storage nodes | "amazin-object-storage-node" |
| PORT                       | Port for the webserver to listen   | 3000                         |
| LOG_DEBUG                  | Logger's debug verbosity level     | true                         |

</details>

### Test scenario

_Given_ that the local environment was [provisioned](#how-to-run) successfully,

_when_ the [script](e2e-test/e2e-tests.sh) is executed,

_then_ three end-to-end/round-trip upload+download tests are expected to succeed.

The following test files are used:

- The text file with a dummy text sample "foo bar baz";
- The `LICENSE` file from the OpenTofu v1.6.0-alpha1 release;
-

The [`tofu.zip` file](https://github.com/opentofu/opentofu/releases/download/v1.6.0-alpha1/tofu_1.6.0-alpha1_darwin_arm64.zip)
from the OpenTofu v1.6.0-alpha1 release.

Run the command to execute the tests:

```commandline
make e2etests
```

**Note**: the execution requires `bash`, `curl`, `wc`, `grep` and `diff`.

### Endpoints

The gateway can be called from the host machine on http://localhost:3000.
See the API contract in the [spec file](internal/restfulhandler/apispec.yaml).

## How it works

The gateway webserver is a reverse proxy which also functions as a cluster load balancer. Its modus operandi can be described as follows:

1. Read and validate HTTP request.
2. Return the response with the error message if the request is invalid. Find additional details in the [API spec file](internal/restfulhandler/apispec.yaml).
3. Read the list of object storage instances available in the cluster using a "service discovery" mechanism. In the [demo](#demo) example,
   the gateway calls the Docker daemon over HTTP to obtain the list of available Minio instances.
4. Communicate to the storage cluster node:

- When a _read_ request is received, the gateway attempts to fetch the requested data by sequentially sending the "read command" to each discovered instance over the network.
  Data will be proxied to the user as soon as the "read command" returns the "found" status. An error message will be returned if no requested data is found or the read operation fails.

- When a _write_ request is received, the gateway "scans" the cluster by sequentially sending the "find command" to each discovered instance over the network. Provided data will overwrite existing object upon discovery.
  If the data is not found, a new object will be created, and the data will be written to the instance selected based on the `objectID` provided by the user. The HTTP status code 201 shall be expected if the write operation succeeds,
  other an error message will be returned.

### Module Design

```mermaid
---
title: The code diagram of the gateway module architecture
---
classDiagram
    class Gateway {
        // pkg/gateway/gateway.go

        -storageInstancesSelector string
        -storageBucket string
        -storageInstancesFinder StorageInstancesFinder
        -storageConnectionDetailsReader StorageConnectionDetailsReader
        -newStorageConnectionFn StorageConnectionFactory
        +Logger              *slog.Logger

        +Read(ctx context.Context, id string) io.ReadCloser, bool, error
        +Write(ctx context.Context, id string, reader io.Reader) error
    }

    class StorageConnectionDetailsReader {
        // pkg/gateway/gateway.go
        <<Interface>>
        Read(ctx context.Context, id string) string, string, string, error
    }

    class StorageInstancesFinder {
        // pkg/gateway/gateway.go
        <<Interface>>
        Find(ctx context.Context, instanceNameFilter string) map[string]struct, error
    }

    class StorageController {
        // pkg/gateway/gateway.go
        <<interface>>
        Read(ctx context.Context, bucketName, objectName string) io.ReadCloser, bool, error
        Write(ctx context.Context, bucketName, objectName string, reader io.Reader) error
        Detected(ctx context.Context, bucketName, objectName string) bool, error
    }

    class StorageConnectionFactory {
        // pkg/gateway/gateway.go
        <<interface>>
        func(endpoint, accessKeyID, secretAccessKey string) StorageController, error
    }

    class Handler {
        // internal/restfulhandler/handler.go
        -rw readWriter
        -commonRoutePrefix string
        -logger            *slog.Logger
        +ServeHTTP(w http.ResponseWriter, r *http.Request)
        -logError(r *http.Request, statusCode int, msg string)
        -knownRoute(p string) bool
        -readObjectID(p string) string
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

StorageConnectionFactory"1"-->"N"StorageController

minioClient --|> StorageController
dockerClient --|> StorageConnectionDetailsReader
dockerClient --|> StorageInstancesFinder

NewClient --|>StorageConnectionFactory

Gateway *--dockerClient
Gateway *--NewClient

Handler <|-- Gateway
```

## How to extend

The gateway module can be extended to use different storage and "service discovery" backends:

- a new service discovery client is required to implement the interface `StorageConnectionFinder`;
- a new storage backed client is required to implement the interface `ObjectReadWriteFinder`.

Find a code snippet example below.

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/kislerdm/minio-gateway/pkg/gateway"
)

func NewStorageConnection(ipAddress, accessKeyID, secretAccessKey string) (gateway.ObjectReadWriteFinder, error) {
	panic("implement me")
	// Definition of the logic to initialise your storage backend.
}

type myServiceDiscoveryClient struct {
	// Attributes of your ServiceDiscoveryClient
}

func (m myServiceDiscoveryClient) Find(ctx context.Context, instanceNameFilter string) (map[string]struct{}, error) {
	panic("implement me")
	// Definition of the logic to find storage instances.
}

func (m myServiceDiscoveryClient) Read(ctx context.Context, id string) (
	ipAddress, accessKeyID, secretAccessKey string, err error,
) {
	panic("implement me")
	// Definition of the logic to retrieve the details required to connected to the storage instance.
}

func main() {
	const (
		storageInstanceSelector = "my-storage-instance"
		storageBucket           = "mybucket"
	)

	gw, err := gateway.New(storageInstanceSelector, storageBucket, &myServiceDiscoveryClient{}, NewStorageConnection,
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})),
	)
	if err != nil {
		log.Fatalln(err)
	}

	if err := gw.Write(context.Background(), "foo", strings.NewReader("bar")); err != nil {
		log.Fatalln(err)
	}
}
```

## How to contribute

**Requirements**:

- Go 1.21
- Docker and docker compose
- gnuMake/cmake

1. Submit the GitHub issue with a bug report or feature request
2. Open PR with a code proposal for the code modification, or extension. 

Please make sure that the following commands succeed before pushing changes. 

- Run unit tests:

```commandline
make tests
```

- Run linters:

```commandline
make lint
```

## License

The codebase present in the repository is distributed under the [MIT license](LICENSE).

The images and graphical material is distributed under
the [CC BY-NC-SA 4.0 DEED](https://creativecommons.org/licenses/by-nc-sa/4.0/) license.

## Disclaimer

The project was developed as a solution addressing the problem
described [here](https://github.com/spacelift-io/homework-object-storage).
