package main

import (
	"log"
	"os"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/env"
	"github.com/urfave/cli/v2"
)

var (
	client *api.Client
	e      env.Env
)

func init() {
	log.SetFlags(0)
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
	app := &cli.App{
		Name:  "fh",
		Usage: "Interact with smart home devices connected to F&Home",
		Commands: []*cli.Command{
			&configCommand,
			&eventCommand,
			&objectCommand,
		},
		CommandNotFound: func(c *cli.Context, command string) {
			log.Printf("invalid command '%s'. See 'fh --help'\n", command)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
