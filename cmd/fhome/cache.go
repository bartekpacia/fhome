// Package main provides the implementation of the fhome CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bartekpacia/fhome/api"
)

// cacheDir returns the path to the cache directory.
func cacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".cache", "fhome")
	err = os.MkdirAll(cacheDir, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
}

// userConfigCachePath returns the path to the user config cache file.
func userConfigCachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "user_config.json"), nil
}

// readUserConfigFromCache reads the user config from the cache file.
// If the cache file doesn't exist or is invalid, it returns nil and an error.
func readUserConfigFromCache() (*api.UserConfig, error) {
	path, err := userConfigCachePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var userConfig api.UserConfig
	err = json.Unmarshal(data, &userConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache file: %w", err)
	}

	return &userConfig, nil
}

// writeUserConfigToCache writes the user config to the cache file.
func writeUserConfigToCache(userConfig *api.UserConfig) error {
	path, err := userConfigCachePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(userConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal user config: %w", err)
	}

	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// updateCache updates the cache in a non-blocking way.
// It returns immediately and updates the cache in a goroutine.
func updateCache(ctx context.Context, createClient func() (*api.Client, error)) {
	client, err := createClient()
	if err != nil {
		slog.Error("failed to create API client: ", slog.Any("error", err))
		return
	}

	userConfig, err := client.GetUserConfig(ctx)
	if err != nil {
		slog.Error("failed to get user config for cache update", slog.Any("error", err))
		return
	}

	err = writeUserConfigToCache(userConfig)
	if err != nil {
		slog.Error("failed to write user config to cache", slog.Any("error", err))
		return
	}

	slog.Debug("updated user config cache")
}

// getUserConfig returns the user config, either from cache or from the server.
//
// If the cache exists and is valid, it returns the cached config and updates the cache in the background.
//
// If the cache doesn't exist or is invalid, it calls the provided userConfigGetter to fetch the config
// and updates the cache.
//
// The userConfigGetter is a function that retrieves the user configuration when called.
func getUserConfig(ctx context.Context, createClient func() (*api.Client, error)) (*api.UserConfig, error) {
	slog.Debug("getting user config from cache")
	userConfig, err := readUserConfigFromCache()
	if err == nil {
		slog.Debug("cache hit! Starting background update and returning")
		go updateCache(ctx, createClient)
		return userConfig, nil
	}

	slog.Debug("cache miss or error, fetching new user config", slog.Any("error", err))
	client, err := createClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create new client: %w", err)
	}
	userConfig, err = client.GetUserConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	// Write to cache
	err = writeUserConfigToCache(userConfig)
	if err != nil {
		slog.Error("failed to write user config to cache", slog.Any("error", err))
		// Continue even if cache write fails
	}

	return userConfig, nil
}
