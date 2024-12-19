package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v2"
)

func main() {
	config := loadConfig()

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
			&cli.BoolFlag{
				Name:  "dbstream",
				Usage: "Stream data from F&Home to database",
			},
			&cli.BoolFlag{
				Name:  "homekit",
				Usage: "Enable HomeKit bridge",
			},
			&cli.BoolFlag{
				Name:  "webserver",
				Usage: "Enable web server with simple website preview",
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
		},
		Before: before,
		Action: func(c *cli.Context) error {
			enableWebServer := c.Bool("webserver")
			enableHomekit := c.Bool("homekit")

			if !enableWebServer && !enableHomekit {
				return fmt.Errorf("no features enabled")
			}

			//if enableWebServer {
			//	go webserver.Start(fhomeClient, apiConfig, config.Email)
			//}

			if enableHomekit {
				name := c.String("name")
				pin := c.String("pin")
				go homekitSyncer(config, name, pin)
			}

			return nil
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
		Email:            k.String("FHOME_EMAIL"),
		Password:         k.String("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: k.String("FHOME_RESOURCE_PASSWORD"),
	}
}

func homekitSyncer(config *highlevel.Config, name, pin string) error {
	fhomeClient, err := highlevel.Connect(config, nil)
	if err != nil {
		return fmt.Errorf("failed to create api client: %v", err)
	}

	userConfig, err := fhomeClient.GetUserConfig()
	if err != nil {
		slog.Error("failed to get user config", slog.Any("error", err))
		return err
	}
	slog.Info("got user config",
		slog.Int("panels", len(userConfig.Panels)),
		slog.Int("cells", len(userConfig.Cells)),
	)

	systemConfig, err := fhomeClient.GetSystemConfig()
	if err != nil {
		slog.Error("failed to get system config", slog.Any("error", err))
		return err
	}
	slog.Info("got system config",
		slog.Int("cells", len(systemConfig.Response.MobileDisplayProperties.Cells)),
		slog.String("source", systemConfig.Source),
	)

	apiConfig, err := api.MergeConfigs(userConfig, systemConfig)
	if err != nil {
		slog.Error("failed to merge configs", slog.Any("error", err))
		return err
	}

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

			err := fhomeClient.SendEvent(ID, value)
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

			err := fhomeClient.SendEvent(ID, value)
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

			err := fhomeClient.SendEvent(ID, value)
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

			err = fhomeClient.SendEvent(ID, value)
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
		msg, err := fhomeClient.ReadMessage(api.ActionStatusTouchesChanged, "")
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
		printCellData(&cellValue, apiConfig)

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

func mustGetenv(varname string) string {
	value := os.Getenv(varname)
	if value == "" {
		slog.Error(varname + " env var is empty or not set")
		os.Exit(1)
	}
	return value
}

// printCellData prints the values of its arguments into a JSON object.
func printCellData(cellValue *api.CellValue, cfg *api.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	// Find panel ID of the cell
	var panelName string
	for _, panel := range cfg.Panels {
		for _, c := range panel.Cells {
			if c.ID == cell.ID {
				panelName = panel.Name
				break
			}
		}
	}

	slog.Debug("object state changed",
		slog.Int("id", cell.ID),
		slog.String("panel", panelName),
		slog.String("name", cell.Name),
		slog.String("desc", cell.Desc),
		slog.String("display_type", string(cellValue.DisplayType)),
		slog.String("value", cellValue.Value),
		slog.String("value_str", cellValue.ValueStr),
	)
	return nil
}
