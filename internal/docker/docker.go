package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}
	return &Client{c}, err
}

type Client struct {
	*client.Client
}

func (c *Client) Scan(ctx context.Context, serviceLabelFilter string) (map[string]string, error) {
	const statusOK = "running"
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: serviceLabelFilter,
		}),
	})
	if err != nil {
		return nil, err
	}

	var o = make(map[string]string)
	for _, container := range containers {
		if container.State == statusOK {
			if container.NetworkSettings != nil {
				for _, network := range container.NetworkSettings.Networks {
					o[container.ID] = network.IPAddress
					// use the first found IP for now
					break
				}
			}
		}
	}
	return o, err
}

func (c *Client) Read(ctx context.Context, instanceID string) (string, string, error) {
	info, err := c.ContainerInspect(ctx, instanceID)
	if err != nil {
		return "", "", err
	}

	accessKeyID, secretAccessKey := readAccessCredentialsFromEnv(info.Config.Env)

	return accessKeyID, secretAccessKey, nil
}

func readAccessCredentialsFromEnv(envVars []string) (accessKeyID, secretAccessKey string) {
	for _, kvPair := range envVars {
		// stop scanning env variable when the credentials are found
		if accessKeyID != "" && secretAccessKey != "" {
			break
		}

		const cntElements = 2
		els := strings.SplitN(kvPair, "=", cntElements)

		switch els[0] {
		case "MINIO_SECRET_KEY":
			secretAccessKey = els[1]
		case "MINIO_ACCESS_KEY":
			accessKeyID = els[1]
		}
	}

	return
}
