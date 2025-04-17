// Package highlevel provides convenient wrappers around some common functionality
// in the [api] package.
package highlevel

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bartekpacia/fhome/api"
	"github.com/gorilla/websocket"
)

type Config struct {
	Email            string
	Password         string
	ResourcePassword string
}

// Connect returns a client that is ready to use.
func Connect(ctx context.Context, config *Config, dialer *websocket.Dialer) (*api.Client, error) {
	client, err := api.NewClient(dialer)
	if err != nil {
		slog.Error("failed to create API client", slog.Any("error", err))
		return nil, fmt.Errorf("create fhome api client: %w", err)
	}

	slog.Debug("created API client")

	err = client.OpenCloudSession(config.Email, config.Password)
	if err != nil {
		slog.Error("failed to open client session", slog.Any("error", err))
		return nil, fmt.Errorf("open client session: %w", err)
	}
	slog.Debug("opened client session", slog.String("email", config.Email))

	myResources, err := client.GetMyResources()
	if err != nil {
		slog.Error("failed to get resource", slog.Any("error", err))
		return nil, fmt.Errorf("get my resources: %w", err)
	}

	slog.Debug("got resource",
		slog.String("name", myResources.FriendlyName0),
		slog.String("id", myResources.UniqueID0),
		slog.String("type", myResources.ResourceType0),
	)

	slog.Debug("opening client to resource session")
	err = client.OpenResourceSession(ctx, config.ResourcePassword)
	if err != nil {
		slog.Error("failed to open client to resource session", slog.Any("error", err))
		return nil, fmt.Errorf("open resource session: %w", err)
	}

	slog.Debug("opened client to resource session")

	return client, nil
}

func GetConfigs(ctx context.Context, fhomeClient *api.Client) (*api.Config, error) {
	userConfig, err := fhomeClient.GetUserConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user config: %w", err)
	}
	slog.Debug("got user config",
		slog.Int("panels", len(userConfig.Panels)),
		slog.Int("cells", len(userConfig.Cells)),
	)

	systemConfig, err := fhomeClient.GetSystemConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get system config: %w", err)
	}
	slog.Debug("got system config",
		slog.Int("cells", len(systemConfig.Response.MobileDisplayProperties.Cells)),
		slog.String("source", systemConfig.Source),
	)

	apiConfig, err := api.MergeConfigs(userConfig, systemConfig)
	if err != nil {
		return nil, fmt.Errorf("merge configs: %w", err)
	}

	return apiConfig, nil
}
