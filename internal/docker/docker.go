package docker

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	gateway "github.com/kislerdm/minio-gateway"
)

func NewClient() (*ClientAdapter, error) {
	cl, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}

	const defaultGWContainerIdentifier = "gateway-container"

	gwContainers, err := cl.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: defaultGWContainerIdentifier,
		}),
	})
	if err != nil {
		return nil, err
	}

	if len(gwContainers) == 0 {
		return nil, errors.New("the gateway docker container cannot be identified")
	}

	gwContainer := gwContainers[0]

	info, err := cl.ContainerInspect(context.Background(), gwContainer.ID)

	// Note that it's assumed that gateway network is transparent to all storage nodes
	var referenceNetworkID string
	for _, network := range info.NetworkSettings.Networks {
		referenceNetworkID = network.IPAddress
		break
	}
	if referenceNetworkID == "" {
		referenceNetworkID = info.NetworkSettings.DefaultNetworkSettings.IPAddress
	}

	return &ClientAdapter{
		referenceNetworkID: referenceNetworkID,
		Client:             cl,
	}, nil
}

type ClientAdapter struct {
	referenceNetworkID string

	*client.Client
}

func (c ClientAdapter) Read(ctx context.Context, nameIdentifier string) (map[string]gateway.MimioConnectionDetails, error) {
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: nameIdentifier,
		}),
	})
	if err != nil {
		return nil, err
	}

	o := map[string]gateway.MimioConnectionDetails{}
	for _, container := range containers {
		if container.State == statusOK {
			id := container.ID

			info, err := c.ContainerInspect(ctx, id)
			if err != nil {
				return nil, err
			}

			connectionDetails := gateway.MimioConnectionDetails{}
			for _, network := range info.NetworkSettings.Networks {
				if network.NetworkID == c.referenceNetworkID {
					connectionDetails.IPAddress = network.IPAddress
				}
			}

			if connectionDetails.IPAddress == "" {
				return nil, errors.New("cannot find ip address of the node " + container.Names[0])
			}

			connectionDetails.AccessKeyID, connectionDetails.SecretAccessKey = readAccessCredentialsFromEnv(info.Config.Env)
			o[id] = connectionDetails
		}
	}

	return nil, err
}

func readAccessCredentialsFromEnv(envVars []string) (accessKeyID, secretAccessKey string) {
	for _, kvPair := range envVars {
		// stop when credentials found
		if accessKeyID != "" && secretAccessKey != "" {
			break
		}

		els := strings.SplitN(kvPair, "=", 2)

		switch els[0] {
		case "MINIO_SECRET_KEY":
			accessKeyID = els[1]
		case "MINIO_ACCESS_KEY":
			secretAccessKey = els[1]
		}
	}

	return
}

const statusOK = "running"
