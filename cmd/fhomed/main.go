package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	errors := make(chan error)
	go setUpHap(config, errors)
	go func() {
		err := <-errors
		log.Fatalln("set up hap failed:", err)
	}()

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
		if cellValue.Voi == "291" {
			if cellValue.Dvs == "100%" {
				log.Println("lamp 291 enabled through fhome")
				a.Switch.On.SetValue(true)
			} else {
				log.Println("lamp 291 disabled through fhome")
				a.Switch.On.SetValue(false)
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

func setUpHap(cfg *config.Config, errors chan error) {
	var switches []*accessory.A
	bartekPanel, err := cfg.GetPanelByName("Bartek")
	if err != nil {
		errors <- fmt.Errorf("failed to get panel by name: %v", err)
		return
	}

	for _, cell := range bartekPanel.Cells {
		swtch := accessory.NewSwitch(accessory.Info{Name: cell.Name})
		swtch.Switch.On.OnValueRemoteUpdate(func(on bool) {
			var newValue string
			if on {
				log.Printf("Tapped %d\n ON", cell.ID)
				newValue = fhome.Value100
			} else {
				log.Printf("Tapped %d\n OFF", cell.ID)
				newValue = fhome.Value0
			}

			err := client.XEvent(cell.ID, newValue)
			if err != nil {
				log.Fatalf("failed to send event with value %s: %v\n", newValue, err)
			}
			log.Println("succeess")
		})

		switches = append(switches, swtch.A)
	}

	bridge := accessory.NewBridge(accessory.Info{Name: "Bartek"})

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, bridge.A, switches...)
	if err != nil {
		// stop if an error happens
		log.Panic(err)
	}

	// Setup a listener for interrupts and SIGTERM signals
	// to stop the server.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		// Stop delivering signals.
		signal.Stop(c)
		// Cancel the context to stop the server.
		cancel()
	}()

	// Run the server.
	server.ListenAndServe(ctx)
}
