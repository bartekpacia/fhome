package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"golang.org/x/exp/slog"
)

func daemon(name, pin string) error {
	client, err := api.NewClient()
	if err != nil {
		slog.Error("failed to create api client", slog.Any("err", err))
		return err
	}

	err = client.OpenCloudSession(k.String("FHOME_EMAIL"), k.String("FHOME_CLOUD_PASSWORD"))
	if err != nil {
		slog.Error("failed to open client session", slog.Any("error", err))
		return err
	} else {
		slog.Info("opened client session", slog.String("email", k.String("FHOME_EMAIL")))
	}

	myResources, err := client.GetMyResources()
	if err != nil {
		slog.Error("failed to get my resources", slog.Any("error", err))
		return err
	} else {
		slog.Info("got resource",
			slog.String("name", myResources.FriendlyName0),
			slog.String("id", myResources.UniqueID0),
			slog.String("type", myResources.ResourceType0),
		)
	}

	err = client.OpenResourceSession(k.String("FHOME_RESOURCE_PASSWORD"))
	if err != nil {
		slog.Error("failed to open client to resource session", slog.Any("error", err))
		return err
	}

	slog.Info("opened client to resource session")

	userConfig, err := client.GetUserConfig()
	if err != nil {
		slog.Error("failed to get user config", slog.Any("error", err))
		return err
	} else {
		slog.Info("got user config",
			slog.Int("panels", len(userConfig.Panels)),
			slog.Int("cells", len(userConfig.Cells)),
		)
	}

	systemConfig, err := client.GetSystemConfig()
	if err != nil {
		slog.Error("failed to get system config", slog.Any("error", err))
		return err
	} else {
		slog.Info("got system config",
			slog.Int("cells", len(systemConfig.Response.MobileDisplayProperties.Cells)),
			slog.String("source", systemConfig.Source),
		)
	}

	config, err := api.MergeConfigs(userConfig, systemConfig)
	if err != nil {
		slog.Error("failed to merge configs", slog.Any("error", err))
		return err
	}

	go serviceListener(client)

	// Here we listen to HomeKit events and convert them to API calls to F&Home
	// to keep the state in sync.
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

			err := client.SendEvent(ID, value)
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

			err := client.SendEvent(ID, value)
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

			err := client.SendEvent(ID, value)
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

			err = client.SendEvent(ID, value)
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
				slog.LogAttrs(context.TODO(), slog.LevelError, "failed to send event", attrs...)
				os.Exit(1)
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "sent event", attrs...)
			}
		},
	}

	home, err := homekitClient.SetUp(config)
	if err != nil {
		slog.Error("failed to set up homekit", slog.Any("error", err))
		return err
	}

	// In this loop, we listen to events from F&Home and send updates to HomeKit
	// to keep the state in sync.
	for {
		msg, err := client.ReadMessage(api.ActionStatusTouchesChanged, "")
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
		printCellData(&cellValue, config)

		// handle lightbulb
		{
			accessory := home.Lightbulbs[cellValue.IntID()]
			if accessory != nil {
				if cellValue.ValueStr == "100%" {
					accessory.Lightbulb.On.SetValue(true)
				} else if cellValue.ValueStr == "0%" {
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
