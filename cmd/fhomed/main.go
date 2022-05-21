package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bartekpacia/fhome/cmd/fhomed/config"
	"github.com/bartekpacia/fhome/env"
	"github.com/bartekpacia/fhome/fhome"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
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
	flag.StringVar(&PIN, "pin", "00102003", "accessory PIN")
	flag.StringVar(&Name, "name", "fhome", "accessory name")

	var err error

	client, err = fhome.NewClient()
	if err != nil {
		log.Fatalf("failed to create fhome client: %v\n", err)
	}

	e = env.Env{}
	e.Load()
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

	file, err := client.GetUserConfig()
	if err != nil {
		log.Fatalf("failed to get user config: %v", err)
	}

	log.Println("got user config")

	touchesResp, err := client.Touches()
	if err != nil {
		log.Fatalf("failed to touches: %v", err)
	}

	config, err := merge(file, touchesResp)
	if err != nil {
		log.Fatalf("failed to merge config: %v", err)
	}

	err = dumpConfig(config)
	if err != nil {
		log.Fatalf("failed to dump config: %v", err)
	}

	results := make(chan map[int]*accessory.Lightbulb)
	go setUpHAP(config, results)

	result := <-results

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

		acc := result[cellValue.IntID()]

		if cellValue.ValueStr == "100%" {
			log.Printf("lamp %d enabled through fhome\n", cellValue.IntID())
			if acc != nil {
				acc.Lightbulb.On.SetValue(true)
			} else {
				log.Printf("switch for objectID %d not found\n", cellValue.IntID())
			}
		} else if cellValue.ValueStr == "0%" {
			log.Printf("lamp %d disabled through fhome\n", cellValue.IntID())
			if acc != nil {
				acc.Lightbulb.On.SetValue(false)
			} else {
				log.Printf("switch for objectID %d not found\n", cellValue.IntID())
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

func richPrint(cellValue *fhome.CellValue, cfg *config.Config) {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		log.Fatalf("get cell %d by ID %v", cellValue.IntID(), err)
	}

	log.Printf("%s (%s)\n", cell.Name, cellValue)
}

// merge create config from "get_user_config" action and "touches" action.
func merge(file *fhome.File, touchesResp *fhome.TouchesResponse) (*config.Config, error) {
	panels := make([]config.Panel, 0)

	for _, fPanel := range file.Panels {
		fCells := file.GetCellsByPanelID(fPanel.ID)
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
	}

	return &cfg, nil
}

func setUpHAP(cfg *config.Config, results chan map[int]*accessory.Lightbulb) {
	var accessories []*accessory.A

	// maps cellID to lightbulbs
	lightbulbMap := make(map[int]*accessory.Lightbulb)
	for _, panel := range cfg.Panels {
		for _, cell := range panel.Cells {
			cell := cell
			a := accessory.NewLightbulb(accessory.Info{Name: strings.TrimSpace(cell.Name)})
			lightbulbMap[cell.ID] = a

			a.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
				err := client.SendXEvent(cell.ID, fhome.ValueToggle)
				if err != nil {
					log.Fatalf("failed to send event to %d: %v\n", cell.ID, err)
				}
				log.Println("succeess")
			})

			accessories = append(accessories, a.A)
		}
	}

	bridge := accessory.NewBridge(accessory.Info{Name: Name})

	fs := hap.NewFsStore("./db")
	server, err := hap.NewServer(fs, bridge.A, accessories...)
	if err != nil {
		log.Panic(err)
	}
	server.Pin = PIN

	results <- lightbulbMap
	server.ListenAndServe(context.Background())
}
