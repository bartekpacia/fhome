package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bartekpacia/fhome/cfg"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

var (
	config cfg.Config
	logger *slog.Logger
)

func main() {
	app := &cli.App{
		Name:  "fhome",
		Usage: "Interact with smart home devices connected to F&Home",
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
		},
		CommandNotFound: func(c *cli.Context, command string) {
			log.Printf("invalid command '%s'. See 'fhome --help'\n", command)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Error("exit", slog.Any("error", err))
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

	if c.Bool("jsonl") {
		logger = slog.New(slog.HandlerOptions{Level: level}.NewJSONHandler(os.Stdout))
	} else {
		logger = slog.New(tint.Options{Level: level, TimeFormat: time.TimeOnly}.NewHandler(os.Stdout))
	}

	k := koanf.New(".")
	p := "/etc/fhome/config.toml"
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		logger.Debug("failed to load config file", slog.Any("error", err))
	} else {
		logger.Debug("loaded config file", slog.String("path", p))
	}

	homeDir, _ := os.UserHomeDir()
	p = fmt.Sprintf("%s/.config/fhome/config.toml", homeDir)
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		logger.Debug("failed to load config file", slog.Any("error", err))
	} else {
		logger.Debug("loaded config file", slog.String("path", p))
	}

	config = cfg.Config{
		Email:            k.String("FHOME_EMAIL"),
		CloudPassword:    k.String("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.String("FHOME_RESOURCE_PASSWORD"),
	}

	err := config.Verify()
	if err != nil {
		log.Fatalf("failed to load config: %v\n", err)
	}

	return nil
}
