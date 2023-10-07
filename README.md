# Minio Blob Storage Gateway

The codebase defines the `Gateway` to distribute Read and Write operations among the Minio Object Storage instances.

## Roadmap

- [ ] Release v0.0.1
    - [ ] Add readme section about the module usage
    - [ ] Add architecture diagram
    - [ ] Update apispec.yaml
    - [ ] Add github action to run tests upon push and PR
    - [ ] Update changelog
- [ ] Release v0.0.2
  - [ ] Fix: add support for big files, >~10Mb
  - [ ] Cache connections
  - [ ] Refactor to simplify codebase
- [ ] Add linter

## Gateway Deployed as Restful WebServer

### Endpoints

See the endpoints definition in the [spec file](pkg/gateway/restfulhandler/apispec.yaml).

## How to run 

### Prerequisites

- Docker 23+
- docker-compose 

### Requirements 

- The Minio Cluster Instances and the Gateway must run as Docker processes
- The Gateway must share the network with the Minio Cluster
- The Gateway mush have access to the socket `/var/run/docker.sock` to communicate to the Docker daemon over HTTP

Run to setup: 

```
docker-compose up --build
```

## Problems

- [x] How to implement sticky load balancing algorithm?
  - [x] In-memory caching: ObjectID -> nodeID
  - Hash function
- [x] How to implement cluster rebalancing algorithm to ensure that objects writen to a node, will be read from it even
  if an additional node was added to the cluster? -> in-memory caching implemented
- [ ] How to handle big objects over 10-32Mb?

## License

The codebase present in the repository is distributed under the [MIT license](LICENSE).

The images and graphical material is distributed under the [CC BY-NC-SA 4.0 DEED](https://creativecommons.org/licenses/by-nc-sa/4.0/) license.
