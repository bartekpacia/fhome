package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bartekpacia/fhome/env"
	"github.com/bartekpacia/fhome/fhome"
	"github.com/urfave/cli/v2"
)

func init() {
	log.SetFlags(0)
}

var listCommand = cli.Command{
	Name:  "list",
	Usage: "list all available objects",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Value:   false,
			Usage:   "print extensive logs",
		},
	},
	Action: func(c *cli.Context) error {
		err := client.OpenCloudSession(e.Email, e.CloudPassword)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenResourceSession(e.ResourcePassword)
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		file, err := client.GetUserConfig()
		if err != nil {
			return fmt.Errorf("failed to get user config: %v", err)
		}

		log.Println("successfully got user config")

		panels := map[string]fhome.Panel{}
		for _, panel := range file.Panels {
			panels[panel.ID] = panel
		}

		fmt.Printf("there are %d cells\n", len(file.Cells))
		for _, cell := range file.Cells {
			fmt.Printf("id: %3d, name: %s\n", cell.ObjectID, cell.Name)
			for _, pos := range cell.PositionInPanel {
				fmt.Printf("\tin panel %s\n", panels[pos.PanelID].Name)
			}
		}

		fmt.Printf("there are %d panels\n", len(file.Panels))
		for _, panel := range file.Panels {
			fmt.Printf("id: %s, name: %s\n", panel.ID, panel.Name)
		}
		return nil
	},
}

var watchCommand = cli.Command{
	Name:  "watch",
	Usage: "watch incoming messages on websockets",
	Action: func(c *cli.Context) error {
		err := client.OpenCloudSession(e.Email, e.CloudPassword)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenResourceSession(e.ResourcePassword)
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		for {
			msg, err := client.ReadAnyMessage()
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			if msg.ActionName == fhome.ActionStatusTouchesChanged {
				var touches fhome.StatusTouchesChangedResponse
				err = json.Unmarshal(msg.Orig, &touches)
				if err != nil {
					return fmt.Errorf("failed to unmarshal touches: %v", err)
				}

				fmt.Printf("%s\n", fhome.Pprint(touches))
			}

		}
	},
}

var toggleCommand = cli.Command{
	Name:  "toggle",
	Usage: "toggle an object on/off",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "object-id",
			Aliases:  []string{"id"},
			Value:    "",
			Usage:    "id of object to toggle",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Value:   false,
			Usage:   "print extensive logs",
		},
	},
	Action: func(c *cli.Context) error {
		objectID := c.Int("object-id")

		err := client.OpenCloudSession(e.Email, e.CloudPassword)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenResourceSession(e.ResourcePassword)
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		err = client.XEvent(objectID, fhome.ValueToggle)
		if err != nil {
			return fmt.Errorf("failed to send xevent to object with id %d: %v", objectID, err)
		}

		log.Println("successfully sent xevent to object with id", objectID)

		return nil
	},
}

var setCommand = cli.Command{
	Name:  "set",
	Usage: "set value of an object (0-100)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "object-id",
			Aliases:  []string{"id"},
			Value:    "",
			Usage:    "id of object to toggle",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "value",
			Aliases:  []string{"val"},
			Usage:    "value (0-100)",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Value:   false,
			Usage:   "print extensive logs",
		},
	},
	Action: func(c *cli.Context) error {
		objectID := c.Int("object-id")
		value := fhome.MapToValue(c.Int("value"))

		err := client.OpenCloudSession(e.Email, e.CloudPassword)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenResourceSession(e.ResourcePassword)
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		err = client.XEvent(objectID, value)
		if err != nil {
			return fmt.Errorf("failed to send xevent to object with id %d: %v", objectID, err)
		}

		log.Println("successfully sent xevent to object with id", objectID)

		return nil
	},
}

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
	app := &cli.App{
		Name:  "fh",
		Usage: "interact with smart devices connected to F&Home system",
		Commands: []*cli.Command{
			&listCommand,
			&watchCommand,
			&toggleCommand,
			&setCommand,
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
