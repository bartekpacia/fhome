package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/bartekpacia/fhome/env"
)

var (
	client *api.Client
	e      env.Env
)

var (
	PIN  string
	Name string
)

func init() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&PIN, "pin", "00102003", "accessory PIN")
	flag.StringVar(&Name, "name", "api", "accessory name")

	var err error

	client, err = api.NewClient()
	if err != nil {
		log.Fatalf("failed to create api api client: %v\n", err)
	}

	e = env.Env{}
	err = e.Load()
	if err != nil {
		log.Fatalf("failed to load env: %v\n", err)
	}
}

func main() {
	flag.Parse()
	err := client.OpenCloudSession(e.Email, e.CloudPassword)
	if err != nil {
		log.Fatalf("failed to open client session: %v", err)
	}

	log.Println("opened client session")

	_, err = client.GetMyResources()
	if err != nil {
		log.Fatalf("failed to get my resources: %v", err)
	}

	log.Println("got my resources")

	err = client.OpenResourceSession(e.ResourcePassword)
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

	err = dumpConfig(config)
	if err != nil {
		log.Fatalf("failed to dump config: %v", err)
	}

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
