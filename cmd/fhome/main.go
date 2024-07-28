package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/bartekpacia/fhome/internal"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v2"
)

var config *internal.Config

func main() {
	loadConfig()
	app := &cli.App{
		Name:                 "fhome",
		Usage:                "Interact with smart home devices connected to F&Home",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output logs in JSON Lines format",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "show debug logs",
			},
		},
		Before: before,
		Commands: []*cli.Command{
			&configCommand,
			&eventCommand,
			&objectCommand,
			&experimentCommand,
		},
		CommandNotFound: func(c *cli.Context, command string) {
			log.Printf("invalid command '%s'. See 'fhome --help'\n", command)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		slog.Error("exit", slog.Any("error", err))
		os.Exit(1)
	}
}

func before(c *cli.Context) error {
	var level slog.Level
	if c.Bool("debug") {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	if c.Bool("json") {
		opts := slog.HandlerOptions{Level: level}
		handler := slog.NewJSONHandler(os.Stdout, &opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	} else {
		opts := tint.Options{Level: level, TimeFormat: time.TimeOnly}
		handler := tint.NewHandler(os.Stdout, &opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	}

	return nil
}

func loadConfig() {
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

	config = &internal.Config{
		Email:            k.String("FHOME_EMAIL"),
		Password:         k.String("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.String("FHOME_RESOURCE_PASSWORD"),
	}
}
