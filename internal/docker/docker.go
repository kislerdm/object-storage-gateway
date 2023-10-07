package docker

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func NewClient() (*Client, error) {
	// TODO: add non-lazy initialisation to ensure that
	//  the process runs as "inside" container and shares the network.
	c, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}
	return &Client{c}, err
}

type Client struct {
	*client.Client
}

func (c *Client) Find(ctx context.Context, instanceNameFilter string) (map[string]struct{}, error) {
	const statusOK = "running"
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: instanceNameFilter,
		}),
	})
	if err != nil {
		return nil, err
	}

	var o = make(map[string]struct{}, len(containers))
	for _, container := range containers {
		if container.State == statusOK {
			o[container.ID] = struct{}{}
		}
	}
	return o, err
}

func (c *Client) Read(ctx context.Context, id string) (ipAddress, accessKeyID, secretAccessKey string, err error) {
	info, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return
	}

	settings := info.NetworkSettings
	if settings == nil {
		err = errors.New("cannot find network configuration for the instance " + id)
		return
	}

	for _, network := range settings.Networks {
		ipAddress = network.IPAddress

		// select the first ip
		// TODO: identify the Gateway process and use it to select common network
		if ipAddress != "" {
			break
		}
	}

	// fallback: if the docker process runs in the default network
	if ipAddress == "" {
		ipAddress = settings.DefaultNetworkSettings.IPAddress
	}

	accessKeyID, secretAccessKey = readAccessCredentialsFromEnv(info.Config.Env)

	return
}

func readAccessCredentialsFromEnv(envVars []string) (accessKeyID, secretAccessKey string) {
	for _, kvPair := range envVars {
		// stop scanning env variable when the credentials are found
		if accessKeyID != "" && secretAccessKey != "" {
			break
		}

		els := strings.SplitN(kvPair, "=", 2)

		switch els[0] {
		case "MINIO_SECRET_KEY":
			secretAccessKey = els[1]
		case "MINIO_ACCESS_KEY":
			accessKeyID = els[1]
		}
	}

	return
}
