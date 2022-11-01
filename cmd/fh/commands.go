package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/bartekpacia/fhome/api"
	dice "github.com/imjasonmiller/godice"
	"github.com/urfave/cli/v2"
)

func bestObjectMatch(object string, config *api.Config) (*api.Cell, float64) {
	var bestScore float64
	var bestObject *api.Cell = nil
	for _, cell := range config.Cells() {
		cell := cell

		if cell.DisplayType != string(api.Percentage) {
			continue
		}

		score := dice.CompareString(object, cell.Name)
		if score > bestScore {
			bestScore = score
			bestObject = &cell
		}
	}

	return bestObject, bestScore
}

var configCommand = cli.Command{
	Name:  "config",
	Usage: "Manage system configuration",
	Subcommands: []*cli.Command{
		{
			Name:  "list",
			Usage: "List all available objects",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "system",
					Usage: "Print config set in the configurator app",
				},
				&cli.BoolFlag{
					Name:  "user",
					Usage: "Print config set in the client apps",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("system") && c.Bool("user") {
					return fmt.Errorf("cannot use both --system and --user")
				}

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

				sysConfig, err := client.GetSystemConfig()
				if err != nil {
					return fmt.Errorf("failed to get sysConfig: %v", err)
				}
				log.Println("got system config")

				userConfig, err := client.GetUserConfig()
				if err != nil {
					return fmt.Errorf("failed to get user config: %v", err)
				}
				log.Println("got user config")

				if c.Bool("system") {
					w := tabwriter.NewWriter(os.Stdout, 8, 8, 0, ' ', 0)
					defer w.Flush()

					fmt.Fprintf(w, "id\tdt\tpreset\tstyle\tperm\tmin\tmax\tstep\tdesc\n")

					cells := sysConfig.Response.MobileDisplayProperties.Cells
					for _, cell := range cells {
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", cell.ID, cell.DisplayType, cell.Preset, cell.Style, cell.Permission, cell.MinValue, cell.MaxValue, cell.Step, cell.Desc)
					}
				} else if c.Bool("user") {
					panels := map[string]api.UserPanel{}
					for _, panel := range userConfig.Panels {
						panels[panel.ID] = panel
					}

					w := tabwriter.NewWriter(os.Stdout, 8, 8, 0, ' ', 0)
					defer w.Flush()

					log.Printf("there are %d cells in %d panels\n", len(userConfig.Cells), len(userConfig.Panels))

					fmt.Fprintf(w, "id\ticon\tname\tpanels\n")

					for _, cell := range userConfig.Cells {
						var p []string
						for _, pos := range cell.PositionInPanel {
							p = append(p, panels[pos.PanelID].Name)
						}

						fmt.Fprintf(w, "%3d\t%s\t%s\t%s\n", cell.ObjectID, cell.IconName(), cell.Name, p)
					}
				} else {
					config, err := api.MergeConfigs(userConfig, sysConfig)
					if err != nil {
						return fmt.Errorf("failed to merge configs: %v", err)
					}

					log.Printf("there are %d panels and %d cells\n", len(config.Panels), len(config.Cells()))
					for _, panel := range config.Panels {
						log.Printf("panel id: %s, name: %s", panel.ID, panel.Name)
						for _, cell := range panel.Cells {
							log.Printf("\tid: %d, name: %s", cell.ID, cell.Name)
						}
						log.Println()
					}
				}

				return nil
			},
		},
	},
}

var eventCommand = cli.Command{
	Name:  "event",
	Usage: "Manage events",
	Subcommands: []*cli.Command{
		{
			Name:  "watch",
			Usage: "Print all incoming messages",
			Action: func(c *cli.Context) error {
				err := client.OpenCloudSession(e.Email, e.CloudPassword)
				if err != nil {
					return fmt.Errorf("failed to open client session: %v", err)
				}

				_, err = client.GetMyResources()
				if err != nil {
					return fmt.Errorf("failed to get my resources: %v", err)
				}

				err = client.OpenResourceSession(e.ResourcePassword)
				if err != nil {
					return fmt.Errorf("failed to open client to resource session: %v", err)
				}

				for {
					msg, err := client.ReadAnyMessage()
					if err != nil {
						return fmt.Errorf("failed to listen: %v", err)
					}

					if msg.ActionName == api.ActionStatusTouchesChanged {
						var touches api.StatusTouchesChangedResponse
						err = json.Unmarshal(msg.Raw, &touches)
						if err != nil {
							return fmt.Errorf("failed to unmarshal touches: %v", err)
						}

						log.Printf("%s\n", api.Pprint(touches))
					}
				}
			},
		},
	},
}

var objectCommand = cli.Command{
	Name:  "object",
	Usage: "Manage objects",
	Subcommands: []*cli.Command{
		{
			Name:      "toggle",
			Usage:     "Toggle object's state (on/off)",
			ArgsUsage: "<object>",
			Action: func(c *cli.Context) error {
				object := c.Args().First()
				if object == "" {
					return fmt.Errorf("object not specified")
				}

				err := client.OpenCloudSession(e.Email, e.CloudPassword)
				if err != nil {
					return fmt.Errorf("failed to open client session: %v", err)
				}

				_, err = client.GetMyResources()
				if err != nil {
					return fmt.Errorf("failed to get my resources: %v", err)
				}

				err = client.OpenResourceSession(e.ResourcePassword)
				if err != nil {
					return fmt.Errorf("failed to open client to resource session: %v", err)
				}

				objectID, err := strconv.Atoi(object)
				if err != nil {
					// string
					log.Println("looking for object with name", object)

					userConfig, err := client.GetUserConfig()
					if err != nil {
						return fmt.Errorf("failed to get user config: %v", err)
					}

					sysConfig, err := client.GetSystemConfig()
					if err != nil {
						return fmt.Errorf("failed to get system config: %v", err)
					}

					config, err := api.MergeConfigs(userConfig, sysConfig)
					if err != nil {
						return fmt.Errorf("failed to merge configs: %v", err)
					}

					bestObject, bestScore := bestObjectMatch(object, config)

					log.Printf("selected object %#v with id %d with %d%% confidence\n", bestObject.Name, bestObject.ID, int(bestScore*100))

					err = client.SendEvent(bestObject.ID, api.ValueToggle)
					if err != nil {
						return fmt.Errorf("failed to send event to object %#v with id %d", bestObject.Name, bestObject.ID)
					}

					log.Printf("sent event to object %#v with id %d\n", bestObject.Name, bestObject.ID)
					return nil
				} else {
					// int

					err = client.SendEvent(objectID, api.ValueToggle)
					if err != nil {
						return fmt.Errorf("failed to send event to object with id %d: %v", objectID, err)
					}

					log.Println("sent event to object with id", objectID)
					return nil
				}
			},
		},
		{
			Name:      "set",
			Usage:     "Set object's state (0-100)",
			ArgsUsage: "<object> <0-100>",
			Action: func(c *cli.Context) error {
				object := c.Args().Get(0)
				if object == "" {
					return fmt.Errorf("object not specified")
				}

				value, err := strconv.Atoi(c.Args().Get(1))
				if err != nil {
					return fmt.Errorf("invalid value: %v", err)
				}

				err = client.OpenCloudSession(e.Email, e.CloudPassword)
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

				objectID, err := strconv.Atoi(object)
				if err != nil {
					// string
					log.Println("looking for object with name", object)

					userConfig, err := client.GetUserConfig()
					if err != nil {
						return fmt.Errorf("failed to get user config: %v", err)
					}

					sysConfig, err := client.GetSystemConfig()
					if err != nil {
						return fmt.Errorf("failed to get system config: %v", err)
					}

					config, err := api.MergeConfigs(userConfig, sysConfig)
					if err != nil {
						return fmt.Errorf("failed to merge configs: %v", err)
					}

					bestObject, bestScore := bestObjectMatch(object, config)

					log.Printf("selected object %#v with id %d with %d%% confidence\n", bestObject.Name, bestObject.ID, int(bestScore*100))

					err = client.SendEvent(bestObject.ID, api.MapLighting(value))
					if err != nil {
						return fmt.Errorf("failed to send event to object %#v with id %d", bestObject.Name, bestObject.ID)
					} else {
						log.Printf("sent event to object %#v with id %d\n", bestObject.Name, bestObject.ID)
						return nil
					}
				} else {
					err = client.SendEvent(objectID, api.MapLighting(value))
					if err != nil {
						return fmt.Errorf("sent event to object with id %d: %v", objectID, err)
					}

					log.Println("sent event to object with id", objectID)
					return nil
				}
			},
		},
	},
}
