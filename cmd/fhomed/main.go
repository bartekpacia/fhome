package main

import (
	"context"
	"fmt"
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
	setUpHap()

	err := client.OpenClientSession(e.Email, e.CloudPassword)
	if err != nil {
		log.Fatalf("failed to open client session: %v", err)
	}

	log.Println("successfully opened client session")

	_, err = client.GetMyResources()
	if err != nil {
		log.Fatalf("failed to get my resources: %v", err)
	}

	log.Println("successfully got my resources")

	err = client.OpenClientToResourceSession(e.ResourcePassword)
	if err != nil {
		log.Fatalf("failed to open client to resource session: %v", err)
	}

	log.Println("successfully opened client to resource session")

	responses := make(chan fhome.StatusTouchesChangedResponse)
	errors := make(chan error)

	go client.Listen(responses, errors)

	for {
		select {
		case msg := <-responses:
			if msg.Response.Cv[0].Voi == "291" {
			}

			fmt.Printf("%s\n", fhome.Pprint(msg))
		case err := <-errors:
			log.Fatalf("failed to listen: %v", err)
		}
	}
}

func setUpHap() {
	// Create the switch accessory.
	a := accessory.NewSwitch(accessory.Info{
		Name: "Bartek's Lamp",
	})

	a.Switch.On.OnValueRemoteUpdate(func(on bool) {
		if on {
			log.Println("Switch is on")
		} else {
			log.Println("Switch is off")
		}
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
