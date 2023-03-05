package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cfg"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/spf13/viper"
)

var config cfg.Config

var (
	PIN  string
	Name string
)

func init() {
	log.SetFlags(0)

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
		log.Fatalf("failed to create api client: %v\n", err)
	}

	err = client.OpenCloudSession(config.Email, config.CloudPassword)
	if err != nil {
		log.Fatalf("failed to open client session: %v", err)
	}

	log.Println("opened client session")

	_, err = client.GetMyResources()
	if err != nil {
		log.Fatalf("failed to get my resources: %v", err)
	}

	log.Println("got my resources")

	err = client.OpenResourceSession(config.ResourcePassword)
	if err != nil {
		log.Fatalf("failed to open client to resource session: %v", err)
	}

	log.Println("opened client to resource session")

	userConfig, err := client.GetUserConfig()
	if err != nil {
		log.Fatalf("failed to get user config: %v", err)
	}

	log.Println("got user config")

	touchesResp, err := client.GetSystemConfig()
	if err != nil {
		log.Fatalf("failed to touches: %v", err)
	}

	config, err := api.MergeConfigs(userConfig, touchesResp)
	if err != nil {
		log.Fatalf("failed to merge config: %v", err)
	}

	go serviceListener(client)

	homekitClient := &homekit.Client{
		PIN:  PIN,
		Name: Name,
		OnLightbulbUpdate: func(ID int, on bool) {
			err := client.SendEvent(ID, api.ValueToggle)
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnLEDUpdate: func(ID int, brightness int) {
			err := client.SendEvent(ID, api.MapLighting(brightness))
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnGarageDoorUpdate: func(ID int) {
			err := client.SendEvent(ID, api.ValueToggle)
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnThermostatUpdate: func(ID int, temperature float64) {
			err = client.SendEvent(ID, api.EncodeTemperature(temperature))
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
	}

	home, err := homekitClient.SetUp(config)
	if err != nil {
		log.Fatalf("failed to set up homekit: %v", err)
	}

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
		richPrint(&cellValue, config)

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
