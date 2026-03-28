package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cmd/fhome-homekit/homekit"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/bartekpacia/fhome/internal"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	app := &cli.Command{
		Name:            "fhome-homekit",
		Usage:           "HomeKit bridge for F&Home",
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
			&cli.StringFlag{
				Name:  "homekit-name",
				Usage: "name of the HomeKit bridge accessory",
				Value: "fhome-homekit",
			},
			&cli.StringFlag{
				Name:  "homekit-pin",
				Usage: "PIN of the HomeKit bridge accessory",
				Value: "00102003",
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
	name := cmd.String("homekit-name")
	pin := cmd.String("homekit-pin")

	config := internal.Load()

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

	return homekitSyncer(ctx, apiClient, apiConfig, name, pin)
}

func homekitSyncer(ctx context.Context, fhomeClient *api.Client, apiConfig *api.Config, name, pin string) error {
	slog.Debug("starting homekit syncer")

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
				slog.String("callback", "OnThermostatUpdate"),
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
			slog.Error("failed to read message", slog.Any("error", err))
			return err
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
