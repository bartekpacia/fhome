package main

import (
	"context"
	"fmt"
	docs "github.com/urfave/cli-docs/v3"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

var config *highlevel.Config

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	loadConfig()
	var app *cli.Command
	app = &cli.Command{
		Name:    "fhomed",
		Usage:   "Long-running daemon for F&Home Cloud",
		Version: version,
		Authors: []any{
			"Bartek Pacia <barpac02@gmail.com>",
		},
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "docs",
				Usage:  "Print documentation in various formats",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:   "format",
						Usage:  "output format [markdown, man, or man-with-section]",
						Hidden: true,
						Value:  "markdown",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					format := cmd.String("format")
					if format == "" || format == "markdown" {
						content, err := docs.ToMarkdown(cmd)
						if err != nil {
							return fmt.Errorf("generate documentation in markdown: %v", err)
						}
						fmt.Println(content)
					} else if format == "man" {
						content, err := docs.ToMan(cmd)
						if err != nil {
							return fmt.Errorf("generate documentation in man: %v", err)
						}
						fmt.Println(content)
					} else if format == "man-with-section" {
						content, err := docs.ToManWithSection(cmd, 1)
						if err != nil {
							return fmt.Errorf("generate documentation in man with section 1: %v", err)
						}
						fmt.Println(content)
					} else {
						return fmt.Errorf("invalid documentation format %#v", format)
					}
					return nil
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output logs in JSON Lines format",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "show debug logs",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "name of the HomeKit bridge accessory",
				Value: "fhomed",
			},
			&cli.StringFlag{
				Name:  "pin",
				Usage: "PIN of the HomeKit bridge accessory",
				Value: "00102003",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) error {
			var level slog.Level
			if cmd.Bool("debug") {
				level = slog.LevelDebug
			} else {
				level = slog.LevelInfo
			}

			if cmd.Bool("json") {
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.String("name")
			pin := cmd.String("pin")

			return daemon(name, pin)
		},
		CommandNotFound: func(ctx context.Context, cmd *cli.Command, command string) {
			log.Printf("invalid command '%s'. See 'fhomed --help'\n", command)
		},
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := app.Run(ctx, os.Args)
	if err != nil {
		slog.Error("exit", slog.Any("error", err))
		os.Exit(1)
	}
}

func loadConfig() {
	k := koanf.New(".")
	p := "/etc/fhomed/config.toml"
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	homeDir, _ := os.UserHomeDir()
	p = fmt.Sprintf("%s/.config/fhomed/config.toml", homeDir)
	if err := k.Load(file.Provider(p), toml.Parser()); err != nil {
		slog.Debug("failed to load config file", slog.Any("error", err))
	} else {
		slog.Debug("loaded config file", slog.String("path", p))
	}

	config = &highlevel.Config{
		Email:            k.String("FHOME_EMAIL"),
		Password:         k.String("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.String("FHOME_RESOURCE_PASSWORD"),
	}
}
