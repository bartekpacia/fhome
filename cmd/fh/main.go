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
	log.SetOutput(os.Stdout)
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
		&cli.BoolFlag{
			Name: "touches",
		},
		&cli.BoolFlag{
			Name: "get_user_config",
		},
	},
	Action: func(c *cli.Context) error {
		err := client.OpenCloudSession(e.Email, e.CloudPassword)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}
		log.Println("opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}
		log.Println("got my resources")

		err = client.OpenResourceSession(e.ResourcePassword)
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}
		log.Println("opened client to resource session")

		if c.Bool("touches") {
			touches, err := client.Touches()
			if err != nil {
				return fmt.Errorf("failed to get touches: %v", err)
			}
			log.Println("got touches")

			log.Println("touches go first")
			cells := touches.Response.MobileDisplayProperties.Cells
			for _, cell := range cells {
				log.Printf("id: %s %s, dt: %s, preset: %s, style: %s, perm: %s, step/value: %s\n", cell.ID, cell.Desc, cell.DisplayType, cell.Preset, cell.Style, cell.Permission, cell.Step)
			}
		}

		if c.Bool("get_user_config") {
			userConfig, err := client.GetUserConfig()
			if err != nil {
				return fmt.Errorf("failed to get user config: %v", err)
			}
			log.Println("successfully got user config")

			panels := map[string]fhome.Panel{}
			for _, panel := range userConfig.Panels {
				panels[panel.ID] = panel
			}

			log.Printf("there are %d cells\n", len(userConfig.Cells))
			for _, cell := range userConfig.Cells {
				log.Printf("id: %3d, name: %s, icon: %s panels:", cell.ObjectID, cell.Name, cell.Icon)
				for _, pos := range cell.PositionInPanel {
					log.Printf(" %s", panels[pos.PanelID].Name)
				}
				log.Println()
			}

			log.Printf("there are %d panels\n", len(userConfig.Panels))
			for _, panel := range userConfig.Panels {
				log.Printf("id: %s, name: %s\n", panel.ID, panel.Name)
			}
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

				log.Printf("%s\n", fhome.Pprint(touches))
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

		err = client.SendXEvent(objectID, fhome.ValueToggle)
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
		value := c.Int("value")

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

		err = client.SendXEvent(objectID, fhome.MapLightning(value))
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
