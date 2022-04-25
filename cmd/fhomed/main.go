package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bartekpacia/fhome/env"
	"github.com/bartekpacia/fhome/fhome"
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
)

var (
	client *fhome.Client
	e      env.Env
	a      *accessory.Switch
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
	go setUpHap()

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

func setUpHap() {
	// Create the switch accessory.
	a = accessory.NewSwitch(accessory.Info{
		Name: "Bartek's Lamp",
	})

	a.Switch.On.OnValueRemoteUpdate(func(on bool) {
		var newValue string
		if on {
			log.Println("switch is enabled through apple")
			newValue = fhome.Value100
		} else {
			log.Println("switch is disabled through apple")
			newValue = fhome.Value0
		}

		err := client.XEvent(291, newValue)
		if err != nil {
			log.Fatalf("failed to send event with value %s: %v\n", newValue, err)
		}
		log.Println("succeess")
	})

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, a.A)
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
