package homekit

import (
	"context"
	"log"
	"strings"

	"github.com/bartekpacia/fhome/cmd/fhomed/config"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
)

type OnLightbulbUpdated func(ID int, v bool)

type OnLEDUpdate func(ID int, brightness float64)

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

func (c *Client) SetUp(
	cfg *config.Config,
	lightbulbs chan<- map[int]*accessory.Lightbulb,
	garageDoors chan<- map[int]*accessory.GarageDoorOpener,
	thermostats chan<- map[int]*accessory.Thermostat,
) {
	var accessories []*accessory.A

	// maps cellID to lightbulbs
	lightbulbMap := make(map[int]*accessory.Lightbulb)
	thermostatsMap := make(map[int]*accessory.Thermostat)
	garageDoorMap := make(map[int]*accessory.GarageDoorOpener)
	for _, panel := range cfg.Panels {
		for _, cell := range panel.Cells {
			cell := cell

			accessoryInfo := accessory.Info{Name: strings.TrimSpace(cell.Name)}
			if cell.Icon == config.IconLighting {

				a := accessory.NewLightbulb(accessoryInfo)
				lightbulbMap[cell.ID] = a

				a.Lightbulb.On.OnValueRemoteUpdate(func(v bool) {
					c.OnLightbulbUpdate(cell.ID, v)
				})

				accessories = append(accessories, a.A)
			}

			if cell.Icon == config.IconTemperature {
				a := accessory.NewThermostat(accessoryInfo)
				thermostatsMap[cell.ID] = a

				a.Thermostat.TargetTemperature.MinVal = 12
				a.Thermostat.TargetTemperature.MaxVal = 28
				a.Thermostat.TemperatureDisplayUnits.Unit = characteristic.UnitCelsius

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

	lightbulbs <- lightbulbMap
	server.ListenAndServe(context.Background())
}
