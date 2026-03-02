// Package internal provides internal utilities for the fhome CLI binaries.
package internal

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

// Load reads fhome configuration from well-known config file paths and environment variables.
// Later sources override earlier ones.
func Load() *highlevel.Config {
	k := koanf.New(".")

	p := "/etc/fhome/config.toml"
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	homeDir, _ := os.UserHomeDir()
	p = fmt.Sprintf("%s/.config/fhome/config.toml", homeDir)
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	if err := k.Load(env.Provider("", ".", nil), nil); err != nil {
		slog.Debug("failed to load environment variables", slog.Any("error", err))
	} else {
		slog.Debug("loaded configuration from environment variables")
	}

	return &highlevel.Config{
		Email:            k.MustString("FHOME_EMAIL"),
		Password:         k.MustString("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.MustString("FHOME_RESOURCE_PASSWORD"),
	}
}
