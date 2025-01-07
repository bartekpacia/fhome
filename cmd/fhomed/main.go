package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	webapi "github.com/bartekpacia/fhome/cmd/fhomed/api"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	app := &cli.Command{
		Name:                  "fhomed",
		Usage:                 "Long-running daemon for F&Home Cloud",
		Authors:               []any{"Bartek Pacia <barpac02@gmail.com>"},
		Version:               version,
		EnableShellCompletion: true,
		HideHelpCommand:       true,
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
					switch format {
					case "", "markdown":
						content, err := docs.ToMarkdown(cmd)
						if err != nil {
							return fmt.Errorf("generate documentation in markdown: %v", err)
						}
						fmt.Println(content)
					case "man":
						content, err := docs.ToMan(cmd)
						if err != nil {
							return fmt.Errorf("generate documentation in man: %v", err)
						}
						fmt.Println(content)
					case "man-with-section":
						content, err := docs.ToManWithSection(cmd, 1)
						if err != nil {
							return fmt.Errorf("generate documentation in man with section 1: %v", err)
						}
						fmt.Println(content)
					default:
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
			&cli.BoolFlag{
				Name:  "homekit",
				Usage: "Enable HomeKit bridge",
			},
			&cli.StringFlag{
				Name:  "homekit-name",
				Usage: "name of the HomeKit bridge accessory. Only makes sense when --homekit is set",
				Value: "fhomed",
			},
			&cli.StringFlag{
				Name:  "homekit-pin",
				Usage: "PIN of the HomeKit bridge accessory. Only makes sense when --homekit is set",
				Value: "00102003",
			},
			&cli.BoolFlag{
				Name:  "api",
				Usage: "Run a web server with a simple API. Requires --api-passphrase",
			},
			&cli.StringFlag{
				Name:  "api-passphrase",
				Usage: "Passphrase to access the API. Only makes sense when coupled with --api",
				Value: "",
			},
			&cli.IntFlag{
				Name:  "api-port",
				Usage: "Port to run API on. Only makes sense when coupled with --api",
				Value: 9001,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
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

			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.String("homekit-name")
			pin := cmd.String("homekit-pin")

			apiPort := cmd.Int("api-port")
			apiPassphrase := cmd.String("api-passphrase")
			if cmd.Bool("api") {
				if apiPassphrase == "" {
					return fmt.Errorf("--api-passphrase is required when using --api")
				}
			}

			config := loadConfig()

			if !cmd.Bool("homekit") && !cmd.Bool("api") {
				return fmt.Errorf("no modules enabled")
			}

			apiClient, err := highlevel.Connect(ctx, config, nil)
			if err != nil {
				return fmt.Errorf("failed to create api apiClient: %v", err)
			}

			apiConfig, err := highlevel.GetConfigs(ctx, apiClient)
			if err != nil {
				return fmt.Errorf("failed to get api configs: %v", err)
			}

			errs := make(chan error)
			if cmd.Bool("homekit") {
				go func() {
					err := homekitSyncer(ctx, apiClient, apiConfig, name, pin)
					slog.Debug("homekit syncer exited", slog.Any("error", err))
					errs <- err
				}()
			}

			if cmd.Bool("api") {
				go func() {
					webSrv := webapi.New(apiClient, apiConfig)
					err := webSrv.Run(ctx, int(apiPort), apiPassphrase)
					slog.Debug("api exited", slog.Any("error", err))
					errs <- err
				}()
			}

			return <-errs
		},
		CommandNotFound: func(ctx context.Context, cmd *cli.Command, command string) {
			log.Printf("invalid command '%s'. See 'fhomed --help'\n", command)
		},
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := app.Run(ctx, os.Args)
	if err != nil {
		slog.Error("exiting because app.Run returned an error", slog.Any("error", err))
		os.Exit(1)
	}
}

func loadConfig() *highlevel.Config {
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

	return &highlevel.Config{
		Email:            k.MustString("FHOME_EMAIL"),
		Password:         k.MustString("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.MustString("FHOME_RESOURCE_PASSWORD"),
	}
}

func homekitSyncer(ctx context.Context, fhomeClient *api.Client, apiConfig *api.Config, name, pin string) error {
	slog.Debug("starting module: homekit syncer")

	// HomeKit -> F&Home
	//
	// Here we listen to events from HomeKit and convert them to API calls to
	// F&Home to keep the state in sync.
	homekitClient := &homekit.Client{
		PIN:  pin,
		Name: name,
		OnLightbulbUpdate: func(ID int, on bool) {
			value := api.ValueToggle
			attrs := []slog.Attr{
				slog.Int("object_id", ID),
				slog.String("value", value),
				slog.String("callback", "OnLightbulbUpdate"),
			}

			err := fhomeClient.SendEvent(ctx, ID, value)
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
				slog.LogAttrs(context.TODO(), slog.LevelError, "failed to send event", attrs...)
				os.Exit(1)
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "sent event", attrs...)
			}
		},
		OnLEDUpdate: func(ID int, brightness int) {
			value := api.MapLighting(brightness)
			attrs := []slog.Attr{
				slog.Int("object_id", ID),
				slog.String("value", value),
				slog.String("callback", "OnLEDUpdate"),
			}

			err := fhomeClient.SendEvent(ctx, ID, value)
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
				slog.LogAttrs(context.TODO(), slog.LevelError, "failed to send event", attrs...)
				os.Exit(1)
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "sent event", attrs...)
			}
		},
		OnGarageDoorUpdate: func(ID int) {
			value := api.ValueToggle
			attrs := []slog.Attr{
				slog.Int("object_id", ID),
				slog.String("value", value),
				slog.String("callback", "OnGarageDoorUpdate"),
			}

			err := fhomeClient.SendEvent(ctx, ID, value)
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
				slog.LogAttrs(context.TODO(), slog.LevelError, "failed to send event", attrs...)
				os.Exit(1)
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "sent event", attrs...)
			}
		},
		OnThermostatUpdate: func(ID int, temperature float64) {
			value := api.EncodeTemperature(temperature)
			attrs := []slog.Attr{
				slog.Int("object_id", ID),
				slog.String("value", value),
				slog.String("callback", "OnGarageDoorUpdate"),
			}

			err := fhomeClient.SendEvent(ctx, ID, value)
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
				slog.LogAttrs(context.TODO(), slog.LevelError, "failed to send event", attrs...)
				os.Exit(1)
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "sent event", attrs...)
			}
		},
	}

	home, err := homekitClient.SetUp(apiConfig)
	if err != nil {
		slog.Error("failed to set up homekit", slog.Any("error", err))
		return err
	}

	// F&Home -> HomeKit
	//
	// In this loop, we listen to events from F&Home and send updates to HomeKit
	// to keep the state in sync.
	for {
		msg, err := fhomeClient.ReadMessage(ctx, api.ActionStatusTouchesChanged, "")
		if err != nil {
			if errors.Is(err, api.ErrClientDone) {
				slog.Info("client is done, stopping the ReadMessage loop")
				return nil
			}

			slog.Warn("got a message but it is an error. Ignoring.", slog.Any("error", err))
			continue
			// return err
		}

		var resp api.StatusTouchesChangedResponse

		err = json.Unmarshal(msg.Raw, &resp)
		if err != nil {
			slog.Error("failed to unmarshal message", slog.Any("error", err))
			return err
		}

		if len(resp.Response.CellValues) == 0 {
			continue
		}

		cellValue := resp.Response.CellValues[0]
		err = highlevel.PrintCellData(&cellValue, apiConfig)
		if err != nil {
			slog.Error("failed to print cell data", slog.Any("error", err))
			return err
		}

		// handle lightbulb
		{
			accessory := home.Lightbulbs[cellValue.IntID()]
			if accessory != nil {
				switch cellValue.ValueStr {
				case "100%":
					accessory.Lightbulb.On.SetValue(true)
				case "0%":
					accessory.Lightbulb.On.SetValue(false)
				}
			}
		}

		// handle LEDs
		{
			accessory := home.ColoredLightbulbs[cellValue.IntID()]
			if accessory != nil {
				newValue, err := api.RemapLighting(cellValue.Value)
				if err != nil {
					slog.Error("failed to remap lightning value",
						slog.Any("error", err),
						slog.String("value", cellValue.Value),
						slog.Int("object_id", cellValue.IntID()),
					)
				}

				accessory.Lightbulb.On.SetValue(newValue > 0)
				err = accessory.Lightbulb.Brightness.SetValue(newValue)
				if err != nil {
					slog.Error("failed to set brightness",
						slog.Any("error", err),
						slog.Int("value", newValue),
						slog.Int("object_id", cellValue.IntID()),
					)
				}
			}
		}

		// handle thermostats
		{
			accessory := home.Thermostats[cellValue.IntID()]
			if accessory != nil {
				newValue, err := api.DecodeTemperatureValue(cellValue.Value)
				if err != nil {
					slog.Error("failed to remap temperature",
						slog.Any("error", err),
						slog.String("value", cellValue.Value),
						slog.Int("object_id", cellValue.IntID()),
					)
				}

				accessory.Thermostat.TargetTemperature.SetValue(newValue)
			}
		}
	}
}
