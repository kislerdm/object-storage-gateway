# Minio Blob Storage Gateway

The codebase defines the `Gateway` to distribute Read and Write operations among the Minio Object Storage instances.

## Roadmap

- [ ] Release v0.0.1
    - [x] Update apispec.yaml
    - [x] Add github action to run tests upon push and PR
    - [ ] Add architecture diagram
    - [ ] Update changelog
- [ ] Release v0.0.2
    - [ ] Fix: add support for big files, >~10Mb
    - [ ] Cache connections
    - [ ] Refactor to simplify codebase
- [ ] Add linter

## Gateway Deployed as Restful HTTP WebServer

### Endpoints

See the endpoints definition in the [spec file](pkg/gateway/restfulhandler/apispec.yaml).

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

## Env Variables Configurations

The Gateway process can be configured using the environment variables listed in the table.

| Variable Name              | Definition                         | Default                      |
|:---------------------------|:-----------------------------------|:-----------------------------|
| STORAGE_INSTANCES_SELECTOR | Selector to identify storage nodes | "amazin-object-storage-node" |
| PORT                       | Port for the webserver to listen   | 3000                         |
| LOG_DEBUG                  |                                    | true                         |

## Problems/ToDo

- [ ] How to handle big objects over 10-32Mb?
  - Example 21Mb input is written as 11.5Mb. Potential reason: transport layer because the content-length is 
    indicated as
```commandline
{"time":"2023-10-08T00:02:44.207206884Z","level":"DEBUG","source":{"function":"github.com/kislerdm/minio-gateway/pkg/gateway/restfulhandler.Handler.ServeHTTP","file":"/app/pkg/gateway/restfulhandler/rest.go","line":49},"msg":"request","webserver":{"path":"/object/4","method":"PUT","content-length":12034212,"headers":"Accept=*/*,Content-Length=12034212,Content-Type=application/x-www-form-urlencoded,Expect=100-continue,User-Agent=curl/8.3.0"}}
```
  Proposed Solution: use multi-form upload.
- [ ] How to preserve formatting of text files?
- [ ] How to ensure the files content is not corrupted? zip archives seem to be corrupt - mime type problem?
- [ ] How to lock Write operation to avoid data duplication and uncertainty. Example. Two simultaneous write request
  with the same ObjectID = 'foo'. The object with that ID was not present in the cluster before the requests. Without
  the lock, two objects with the same ID may end up in two different nodes.

## License

The codebase present in the repository is distributed under the [MIT license](LICENSE).

The images and graphical material is distributed under
the [CC BY-NC-SA 4.0 DEED](https://creativecommons.org/licenses/by-nc-sa/4.0/) license.
