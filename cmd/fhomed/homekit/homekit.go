package homekit

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bartekpacia/fhome/cmd/fhomed/config"
	"github.com/bartekpacia/fhome/fhome"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
)

type OnLightbulbUpdated func(ID int, v bool)

type OnLEDUpdate func(ID int, brightness int)

type OnGarageDoorUpdated func(ID int)

type OnThermostatUpdated func(ID int, v float64)

type Client struct {
	PIN                string
	Name               string
	OnLightbulbUpdate  OnLightbulbUpdated
	OnLEDUpdate        OnLEDUpdate
	OnGarageDoorUpdate OnGarageDoorUpdated
	OnThermostatUpdate OnThermostatUpdated
}

type Home struct {
	Lightbulbs        map[int]*accessory.Lightbulb
	ColoredLightbulbs map[int]*accessory.ColoredLightbulb
	GarageDoors       map[int]*accessory.GarageDoorOpener
	Thermostats       map[int]*accessory.Thermostat
}

func (c *Client) SetUp(cfg *config.Config) (*Home, error) {
	var accessories []*accessory.A

	// maps cellID to lightbulbs
	lightbulbMap := make(map[int]*accessory.Lightbulb)
	coloredLightbulbs := make(map[int]*accessory.ColoredLightbulb)
	thermostatsMap := make(map[int]*accessory.Thermostat)
	garageDoorMap := make(map[int]*accessory.GarageDoorOpener)
	for _, panel := range cfg.Panels {
		for _, cell := range panel.Cells {
			cell := cell

			accessoryInfo := accessory.Info{Name: strings.TrimSpace(cell.Name)}
			if cell.Icon == config.IconLighting {
				if strings.Contains(cell.Name, "LED") {
					a := accessory.NewColoredLightbulb(accessoryInfo)
					coloredLightbulbs[cell.ID] = a

					a.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
						var val int
						if on {
							val = 100
						}

						c.OnLEDUpdate(cell.ID, val)
					})

					a.Lightbulb.Brightness.OnValueRemoteUpdate(func(v int) {
						c.OnLEDUpdate(cell.ID, v)
					})

					accessories = append(accessories, a.A)
				} else {
					a := accessory.NewLightbulb(accessoryInfo)
					lightbulbMap[cell.ID] = a

					a.Lightbulb.On.OnValueRemoteUpdate(func(v bool) {
						c.OnLightbulbUpdate(cell.ID, v)
					})

					accessories = append(accessories, a.A)
				}
			}
			if cell.Icon == config.IconTemperature {
				a := accessory.NewThermostat(accessoryInfo)
				thermostatsMap[cell.ID] = a

				a.Thermostat.TargetTemperature.Val = 12
				a.Thermostat.TargetTemperature.MinVal = 12
				a.Thermostat.TargetTemperature.MaxVal = 28

				currentTemp, err := fhome.DecodeTemperatureValue(cell.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to remap temperature: %v", err)
				}
				fmt.Println("cell name: ", cell.Name, "currentTemp: ", currentTemp)

				a.Thermostat.CurrentTemperature.Val = currentTemp

				a.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(v float64) {
					c.OnThermostatUpdate(cell.ID, v)
				})

				accessories = append(accessories, a.A)
			}

			if cell.Icon == config.IconGate {
				a := accessory.NewGarageDoorOpener(accessoryInfo)
				garageDoorMap[cell.ID] = a

				a.GarageDoorOpener.TargetDoorState.OnValueRemoteUpdate(func(v int) {
					c.OnGarageDoorUpdate(cell.ID)
				})

				accessories = append(accessories, a.A)
			}
		}
	}

	bridge := accessory.NewBridge(accessory.Info{Name: c.Name})

	fs := hap.NewFsStore("./db")
	server, err := hap.NewServer(fs, bridge.A, accessories...)
	if err != nil {
		log.Panic(err)
	}
	server.Pin = c.PIN

	go server.ListenAndServe(context.Background())

	return &Home{
		Lightbulbs:        lightbulbMap,
		ColoredLightbulbs: coloredLightbulbs,
		GarageDoors:       garageDoorMap,
		Thermostats:       thermostatsMap,
	}, nil
}
