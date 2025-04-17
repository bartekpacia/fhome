package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	// Maybe slog setup has to happen outside of Before(), because then it's not run during ShellComplete?
	var logLevel slog.Level
	if os.Getenv("FHOME_DEBUG") != "" {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	_ = slog.New(handler)
	// slog.SetDefault(logger)

	app := &cli.Command{
		Name:                  "fhome",
		Usage:                 "Interact with smart home devices connected to F&Home",
		Authors:               []any{"Bartek Pacia <barpac02@gmail.com>"},
		Version:               version,
		EnableShellCompletion: true,
		HideHelpCommand:       true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output logs in JSON Lines format",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "show debug logs (can also be enabled with FHOME_DEBUG env var)",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd.Bool("debug") {
				logLevel = slog.LevelDebug
			}

			if cmd.Bool("json") {
				opts := slog.HandlerOptions{Level: logLevel}
				handler := slog.NewJSONHandler(os.Stdout, &opts)
				logger := slog.New(handler)
				slog.SetDefault(logger)
			} else {
				opts := tint.Options{Level: logLevel, TimeFormat: time.TimeOnly}
				handler := tint.NewHandler(os.Stdout, &opts)
				logger := slog.New(handler)
				slog.SetDefault(logger)
			}

			return ctx, nil
		},
		Commands: []*cli.Command{
			&configCommand,
			&eventCommand,
			&objectCommand,
		},
		CommandNotFound: func(ctx context.Context, cmd *cli.Command, command string) {
			log.Printf("invalid command '%s'. See 'fhome --help'\n", command)
		},
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := app.Run(ctx, os.Args)
	if err != nil {
		slog.Error("exit", slog.Any("error", err))
		os.Exit(1)
	}
}

func loadConfig() *highlevel.Config {
	k := koanf.New(".")

	// Attempt to load configuration from /etc/fhome/config.toml
	p := "/etc/fhome/config.toml"
	err := k.Load(file.Provider(p), toml.Parser())
	if err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	// Attempt to load configuration from ~/.config/fhome/config.toml
	homeDir, _ := os.UserHomeDir()
	p = fmt.Sprintf("%s/.config/fhome/config.toml", homeDir)
	err = k.Load(file.Provider(p), toml.Parser())
	if err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	// Attempt to load configuration from environment variables.
	// This will override any values from config files.
	err = k.Load(env.Provider("", ".", nil), nil)
	if err != nil {
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
