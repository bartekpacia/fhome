package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

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

func init() {
	log.SetFlags(0)

	var err error

	client, err = fhome.NewClient()
	if err != nil {
		log.Fatalf("failed to create fhome client: %v\n", err)
	}

	e = env.Env{}
	e.Load()
}

func main() {
	err := client.OpenCloudSession(e.Email, e.CloudPassword)
	if err != nil {
		log.Fatalf("failed to open client session: %v", err)
	}

	log.Println("successfully opened client session")

	_, err = client.GetMyResources()
	if err != nil {
		log.Fatalf("failed to get my resources: %v", err)
	}

	log.Println("successfully got my resources")

	err = client.OpenResourceSession(e.ResourcePassword)
	if err != nil {
		log.Fatalf("failed to open client to resource session: %v", err)
	}

	log.Println("successfully opened client to resource session")

	file, err := client.GetUserConfig()
	if err != nil {
		log.Fatalf("failed to get user config: %v", err)
	}
	config, err := fileToConfig(file)
	if err != nil {
		log.Fatalf("failed to convert file to config: %v", err)
	}

	results := make(chan map[int]*accessory.Switch)
	errors := make(chan error)
	go setUpHap(config, results, errors)
	go func() {
		err := <-errors
		log.Fatalln("set up hap failed:", err)
	}()

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

		if len(resp.Response.Cv) == 0 {
			continue
		}

		cellValue := resp.Response.Cv[0]
		cellID, err := strconv.Atoi(cellValue.Voi)
		if err != nil {
			log.Fatalln("failed to convert cell id to int:", err)
		}

		if cellID == 291 || cellID == 370 || cellID == 380 || cellID == 381 || cellID == 382 {
			swtch := result[cellID]

			if cellValue.Dvs == "100%" {
				log.Printf("lamp %d enabled through fhome\n", cellID)
				if swtch != nil {
					swtch.Switch.On.SetValue(true)
				} else {
					log.Printf("switch for objectID %d not found\n", cellID)
				}
			} else {
				log.Printf("lamp %d disabled through fhome\n", cellID)
				if swtch != nil {
					swtch.Switch.On.SetValue(false)
				} else {
					log.Printf("switch for objectID %d not found\n", cellID)
				}
			}
		}
	}
}

func fileToConfig(f *fhome.File) (*config.Config, error) {
	panels := make([]config.Panel, 0)
	for _, fPanel := range f.Panels {
		fCells := f.GetCellsByPanelID(fPanel.ID)
		cells := make([]config.Cell, 0)
		for _, fCell := range fCells {
			cell := config.Cell{
				ID:   fCell.ObjectID,
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

	return &config.Config{Panels: panels}, nil
}

func setUpHap(cfg *config.Config, results chan map[int]*accessory.Switch, errors chan error) {
	var switches []*accessory.A
	bartekPanel, err := cfg.GetPanelByName("Bartek")
	if err != nil {
		errors <- fmt.Errorf("failed to get panel by name: %v", err)
		return
	}

	mapping := make(map[int]*accessory.Switch)
	for _, cell := range bartekPanel.Cells {
		cell := cell
		swtch := accessory.NewSwitch(accessory.Info{Name: cell.Name})
		mapping[cell.ID] = swtch

		swtch.Switch.On.OnValueRemoteUpdate(func(on bool) {
			var newValue string
			if on {
				log.Printf("c %d ON\n", cell.ID)
				newValue = fhome.Value100
			} else {
				log.Printf("Tapped %d OFF\n", cell.ID)
				newValue = fhome.Value0
			}

			err := client.SendXEvent(cell.ID, newValue)
			if err != nil {
				log.Fatalf("failed to send event with value %s: %v\n", newValue, err)
			}
			log.Println("succeess")
		})

		switches = append(switches, swtch.A)
	}

	bridge := accessory.NewBridge(accessory.Info{Name: "Bartek"})

	fs := hap.NewFsStore("./db")
	server, err := hap.NewServer(fs, bridge.A, switches...)
	if err != nil {
		log.Panic(err)
	}

	results <- mapping
	server.ListenAndServe(context.Background())
}
