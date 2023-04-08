package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cfg"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/lmittmann/tint"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

var (
	config cfg.Config
	logger *slog.Logger
)

var (
	PIN  string
	Name string
)

func init() {
	log.SetFlags(0)

	logger = slog.New(tint.Options{TimeFormat: time.TimeOnly}.NewHandler(os.Stderr))

	flag.StringVar(&PIN, "pin", "00102003", "accessory PIN")
	flag.StringVar(&Name, "name", "fhomed", "accessory name")

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/fhomed/")
	viper.AddConfigPath("/etc/fhomed/")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("failed to read in config: %v\n", err)
		}
	}

	config = cfg.Config{
		Email:            viper.GetString("FHOME_EMAIL"),
		CloudPassword:    viper.GetString("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: viper.GetString("FHOME_RESOURCE_PASSWORD"),
	}

	err := config.Verify()
	if err != nil {
		log.Fatalf("failed to load env: %v\n", err)
	}
}

func main() {
	flag.Parse()

	client, err := api.NewClient()
	if err != nil {
		logger.Error("failed to create api client", slog.Any("err", err))
		os.Exit(1)
	}

	err = client.OpenCloudSession(config.Email, config.CloudPassword)
	if err != nil {
		logger.Error("failed to open client session", slog.Any("error", err))
		os.Exit(1)
	} else {
		logger.Info("opened client session", slog.String("email", config.Email))
	}

	myResources, err := client.GetMyResources()
	if err != nil {
		logger.Error("failed to get my resources", slog.Any("error", err))
		os.Exit(1)
	} else {
		logger.Info("got resource",
			slog.String("name", myResources.FriendlyName0),
			slog.String("id", myResources.UniqueID0),
			slog.String("type", myResources.ResourceType0),
		)
	}

	err = client.OpenResourceSession(config.ResourcePassword)
	if err != nil {
		logger.Error("failed to open client to resource session", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("opened client to resource session")

	userConfig, err := client.GetUserConfig()
	if err != nil {
		logger.Error("failed to get user config", slog.Any("error", err))
		os.Exit(1)
	} else {
		logger.Info("got user config",
			slog.Int("panels", len(userConfig.Panels)),
			slog.Int("cells", len(userConfig.Cells)),
		)
	}

	systemConfig, err := client.GetSystemConfig()
	if err != nil {
		logger.Error("failed to get system config", slog.Any("error", err))
		os.Exit(1)
	} else {
		logger.Info("got system config",
			slog.Int("cells", len(systemConfig.Response.MobileDisplayProperties.Cells)),
			slog.String("source", systemConfig.Source),
		)
	}

	config, err := api.MergeConfigs(userConfig, systemConfig)
	if err != nil {
		logger.Error("failed to merge configs", slog.Any("error", err))
		os.Exit(1)
	}

	go serviceListener(client)

	// Here we listen to HomeKit events and convert them to API calls to F&Home
	// to keep the state in sync.
	homekitClient := &homekit.Client{
		PIN:  PIN,
		Name: Name,
		OnLightbulbUpdate: func(ID int, on bool) {
			err := client.SendEvent(ID, api.ValueToggle)
			if err != nil {
				logger.Error("failed to send event",
					slog.Any("error", err),
					slog.Int("object_id", ID),
					slog.String("value", api.ValueToggle),
				)
				os.Exit(1)
			}
		},
		OnLEDUpdate: func(ID int, brightness int) {
			value := api.MapLighting(brightness)
			err := client.SendEvent(ID, value)
			if err != nil {
				logger.Error("failed to send event",
					slog.Any("error", err),
					slog.Int("object_id", ID),
					slog.String("value", value),
				)
				os.Exit(1)
			}
		},
		OnGarageDoorUpdate: func(ID int) {
			err := client.SendEvent(ID, api.ValueToggle)
			if err != nil {
				logger.Error("failed to send event",
					slog.Any("error", err),
					slog.Int("object_id", ID),
					slog.String("value", api.ValueToggle),
				)
				os.Exit(1)
			}
		},
		OnThermostatUpdate: func(ID int, temperature float64) {
			value := api.EncodeTemperature(temperature)
			err = client.SendEvent(ID, value)
			if err != nil {
				logger.Error("failed to send event",
					slog.Any("error", err),
					slog.Int("object_id", ID),
					slog.String("value", value),
				)
				os.Exit(1)
			}
		},
	}

	home, err := homekitClient.SetUp(config)
	if err != nil {
		logger.Error("failed to set up homekit", slog.Any("error", err))
		os.Exit(1)
	}

	// In this loop, we listen to events from F&Home and send updates to HomeKit
	// to keep the state in sync.
	for {
		msg, err := client.ReadMessage(api.ActionStatusTouchesChanged, "")
		if err != nil {
			log.Fatalln("failed to read message:", err)
		}

		var resp api.StatusTouchesChangedResponse

		err = json.Unmarshal(msg.Raw, &resp)
		if err != nil {
			log.Fatalln("failed to unmarshal message:", err)
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
					log.Printf("failed to remap lightning: %v\n", err)
				}

				accessory.Lightbulb.On.SetValue(newValue > 0)
				err = accessory.Lightbulb.Brightness.SetValue(newValue)
				if err != nil {
					log.Printf("failed to set brightness to %d: %v\n", newValue, err)
				}
			}
		}

		// handle thermostats
		{
			accessory := home.Thermostats[cellValue.IntID()]
			if accessory != nil {
				newValue, err := api.DecodeTemperatureValue(cellValue.Value)
				if err != nil {
					log.Printf("failed to remap temperature: %v\n", err)
				}

				accessory.Thermostat.TargetTemperature.SetValue(newValue)
			}
		}
	}
}

// Below is a hacky workaround for myself to open my gate from my phone.

func serviceListener(client *api.Client) {
	http.HandleFunc("/gate", func(w http.ResponseWriter, r *http.Request) {
		var result string
		err := client.SendEvent(260, api.ValueToggle)
		if err != nil {
			result = fmt.Sprintf("Failed to send event: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if result != "" {
			log.Print(result)
			fmt.Fprint(w, result)
		}
	})

	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		panic(err)
	}
}
