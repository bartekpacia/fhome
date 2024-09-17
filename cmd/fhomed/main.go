package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v2"
)

var config *highlevel.Config

func main() {
	loadConfig()
	app := &cli.App{
		Name:    "fhomed",
		Usage:   "Long-running daemon for F&Home Cloud",
		Version: "0.1.24",
		Authors: []*cli.Author{
			{
				Name:  "Bartek Pacia",
				Email: "barpac02@gmail.com",
			},
		},
		EnableBashCompletion: true,
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
				Action: func(c *cli.Context) error {
					format := c.String("format")
					if format == "" || format == "markdown" {
						fmt.Println(c.App.ToMarkdown())
						return nil
					}
					if format == "man" {
						fmt.Println(c.App.ToMan())
						return nil
					}
					if format == "man-with-section" {
						fmt.Println(c.App.ToManWithSection(1))
						return nil
					}
					return fmt.Errorf("invalid format '%s'", format)
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
		Before: before,
		Action: func(c *cli.Context) error {
			name := c.String("name")
			pin := c.String("pin")

			return daemon(name, pin)
		},
		CommandNotFound: func(c *cli.Context, command string) {
			log.Printf("invalid command '%s'. See 'fhomed --help'\n", command)
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
