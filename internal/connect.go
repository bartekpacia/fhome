// Package internal contains internal logic of the command line tools.
package internal

import (
	"log/slog"

	"github.com/bartekpacia/fhome/api"
)

type Config struct {
	Email            string
	Password         string
	ResourcePassword string
}

// Connect returns a client that is ready to use.
func Connect(config *Config) (*api.Client, error) {
	client, err := api.NewClient(nil)
	if err != nil {
		slog.Error("failed to create API client", slog.Any("error", err))
		return nil, err
	} else {
		slog.Debug("created API client")
	}

	err = client.OpenCloudSession(config.Email, config.Password)
	if err != nil {
		slog.Error("failed to open client session", slog.Any("error", err))
		return nil, err
	} else {
		slog.Debug("opened client session", slog.String("email", config.Email))
	}

	myResources, err := client.GetMyResources()
	if err != nil {
		slog.Error("failed to get resource", slog.Any("error", err))
		return nil, err
	} else {
		slog.Debug("got resource",
			slog.String("name", myResources.FriendlyName0),
			slog.String("id", myResources.UniqueID0),
			slog.String("type", myResources.ResourceType0),
		)
	}

	err = client.OpenResourceSession(config.ResourcePassword)
	if err != nil {
		slog.Error("failed to open client to resource session", slog.Any("error", err))
		return nil, err
	} else {
		slog.Debug("opened client to resource session")
	}

	return client, nil
}
