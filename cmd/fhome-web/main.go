package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/bartekpacia/fhome/cmd/fhome-web/webserver"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/bartekpacia/fhome/internal"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	app := &cli.Command{
		Name:            "fhome-web",
		Usage:           "Web server for F&Home device preview",
		Authors:         []any{"Bartek Pacia <barpac02@gmail.com>"},
		Version:         version,
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output logs in JSON Lines format",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "show debug logs (can also be enabled with FHOME_DEBUG env var)",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port to listen on",
				Value: 9001,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			var level slog.Level
			if cmd.Bool("debug") || os.Getenv("FHOME_DEBUG") != "" {
				level = slog.LevelDebug
			} else {
				level = slog.LevelInfo
			}

			if cmd.Bool("json") {
				handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
				slog.SetDefault(slog.New(handler))
			} else {
				handler := tint.NewHandler(os.Stdout, &tint.Options{Level: level, TimeFormat: time.TimeOnly})
				slog.SetDefault(slog.New(handler))
			}

			return ctx, nil
		},
		Action: run,
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := app.Run(ctx, os.Args)
	if err != nil {
		slog.Error("exit", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	config := internal.Load()
	port := int(cmd.Int("port"))

	apiClient, err := highlevel.Connect(ctx, config, nil)
	if err != nil {
		return fmt.Errorf("connect to fhome: %v", err)
	}

	apiConfig, err := highlevel.GetConfigs(ctx, apiClient)
	if err != nil {
		return fmt.Errorf("get configs: %v", err)
	}

	slog.Info("connected to F&Home",
		slog.Int("panels", len(apiConfig.Panels)),
		slog.Int("cells", len(apiConfig.Cells())),
	)

	return webserver.Run(ctx, apiClient, apiConfig, config.Email, port)
}
