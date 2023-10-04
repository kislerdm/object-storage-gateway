# Blob storage gateway

The HTTP gateway to a cluster of Minio Object Storage instances.

## Endpoints

The gateway supports two operations:
- Object Write
- Object Read

See the endpoints definition in the [spec file](apispec.yaml).

## Problems

- [ ] How to implement sticky load balancing algorithm? 
  - Caching: ObjectID -> nodeID
  - Hash function
- [ ] How to implement cluster rebalancing algorithm to ensure that objects writen to a node, will be read from it even 
if an additional node was added to the cluster?
- [ ] How to handle big objects over 10-32Mb?

## How to run

### Prerequisites

- Docker 23+
- docker-compose 

### Requirements 

- The Minio cluster and the gateway must be deployed to an environment with the Docker daemon
- The cluster and the gateway must share the network
- The gateway mush have access to the Docker socket `/var/run/docker.sock`

Run to setup: 

```
docker-compose up --build
```

## License

The codebase present in the repository is distributed under the [MIT license](LICENSE).

The images and graphical material is distributed under the [CC BY-NC-SA 4.0 DEED](https://creativecommons.org/licenses/by-nc-sa/4.0/) license.
