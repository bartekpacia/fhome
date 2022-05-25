package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/bartekpacia/fhome/cmd/fhomed/config"
	"github.com/bartekpacia/fhome/cmd/fhomed/homekit"
	"github.com/bartekpacia/fhome/env"
	"github.com/bartekpacia/fhome/fhome"
)

var (
	client *fhome.Client
	e      env.Env
)

var (
	PIN  string
	Name string
)

func init() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&PIN, "pin", "00102003", "accessory PIN")
	flag.StringVar(&Name, "name", "fhome", "accessory name")

	var err error

	client, err = fhome.NewClient()
	if err != nil {
		log.Fatalf("failed to create fhome client: %v\n", err)
	}

	e = env.Env{}
	err = e.Load()
	if err != nil {
		log.Fatalf("failed to load env variables: %v\n", err)
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

	touchesResp, err := client.Touches()
	if err != nil {
		log.Fatalf("failed to touches: %v", err)
	}

	config, err := merge(userConfig, touchesResp)
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
			err := client.SendXEvent(ID, fhome.ValueToggle)
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnLEDUpdate: func(ID int, brightness int) {
			err := client.SendXEvent(ID, fhome.MapLighting(brightness))
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnGarageDoorUpdate: func(ID int) {
			err := client.SendXEvent(ID, fhome.ValueToggle)
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
		OnThermostatUpdate: func(ID int, temperature float64) {
			err = client.SendXEvent(ID, fhome.MapTemperature(temperature))
			if err != nil {
				log.Fatalf("failed to send event to %d: %v\n", ID, err)
			}
		},
	}

	home := homekitClient.SetUp(config)

	for {
		msg, err := client.ReadMessage(fhome.ActionStatusTouchesChanged, "")
		if err != nil {
			log.Fatalln("failed to read message:", err)
		}

		var resp fhome.StatusTouchesChangedResponse

		err = json.Unmarshal(msg.Orig, &resp)
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
				newValue, err := fhome.RemapLighting(cellValue.Value)
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
				newValue, err := fhome.RemapTemperature(cellValue.Value)
				if err != nil {
					log.Printf("failed to remap temperature: %v\n", err)
				}

				accessory.Thermostat.TargetTemperature.SetValue(newValue)
			}
		}
	}
}

func dumpConfig(cfg *config.Config) error {
	file, err := os.Create("config.json")
	if err != nil {
		return fmt.Errorf("create config.json: %v", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %v", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("write config: %v", err)
	}

	return nil
}

func richPrint(cellValue *fhome.CellValue, cfg *config.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	log.Printf(",%d, %s, %s, %s, %s, %s\n", cell.ID, cell.Name, cell.Desc, cellValue.DisplayType, cellValue.Value, cellValue.ValueStr)
	return nil
}

// merge create config from "get_user_config" action and "touches" action.
func merge(userConfig *fhome.UserConfig, touchesResp *fhome.TouchesResponse) (*config.Config, error) {
	panels := make([]config.Panel, 0)

	for _, fPanel := range userConfig.Panels {
		fCells := userConfig.GetCellsByPanelID(fPanel.ID)
		cells := make([]config.Cell, 0)
		for _, fCell := range fCells {
			cell := config.Cell{
				ID:   fCell.ObjectID,
				Icon: config.CreateIcon(fCell.Icon),
				Name: fCell.Name,
			}
			cells = append(cells, cell)
		}

		panel := config.Panel{
			ID:    fPanel.ID,
			Name:  fPanel.Name,
			Cells: cells,
		}

		panels = append(panels, panel)
	}

	cfg := config.Config{Panels: panels}

	for _, cell := range touchesResp.Response.MobileDisplayProperties.Cells {
		cellID, err := strconv.Atoi(cell.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert cell ID %s to int: %v", cell.ID, err)
		}

		cfgCell, err := cfg.GetCellByID(cellID)
		if err != nil {
			log.Printf("could not find cell with id %d in config: %v", cellID, err)
			continue
		}

		cfgCell.Desc = cell.Desc
		cfgCell.Value = cell.Step
		cfgCell.TypeNumber = cell.TypeNumber
		cfgCell.Preset = cell.Preset
		cfgCell.Style = cell.Style
		cfgCell.MinValue = cell.MinValue
		cfgCell.MaxValue = cell.MaxValue
	}

	return &cfg, nil
}
