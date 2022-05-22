package homekit

import (
	"context"
	"log"
	"strings"

	"github.com/bartekpacia/fhome/cmd/fhomed/config"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
)

type OnLightbulbUpdated func(ID int, v bool)

type Client struct {
	PIN               string
	Name              string
	OnLightbulbUpdate OnLightbulbUpdated
	// OnThermostatUpdated
}

func (c *Client) SetUp(cfg *config.Config, results chan map[int]*accessory.Lightbulb) {
	var accessories []*accessory.A

	// maps cellID to lightbulbs
	lightbulbMap := make(map[int]*accessory.Lightbulb)
	for _, panel := range cfg.Panels {
		for _, cell := range panel.Cells {
			cell := cell
			a := accessory.NewLightbulb(accessory.Info{Name: strings.TrimSpace(cell.Name)})
			lightbulbMap[cell.ID] = a

			a.Lightbulb.On.OnValueRemoteUpdate(func(v bool) {
				c.OnLightbulbUpdate(cell.ID, v)
			})

			accessories = append(accessories, a.A)
		}
	}

	bridge := accessory.NewBridge(accessory.Info{Name: c.Name})

	fs := hap.NewFsStore("./db")
	server, err := hap.NewServer(fs, bridge.A, accessories...)
	if err != nil {
		log.Panic(err)
	}
	server.Pin = c.PIN

	results <- lightbulbMap
	server.ListenAndServe(context.Background())
}
