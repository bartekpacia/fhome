package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/internal"
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

		score := strutil.Similarity(object, cell.Name, metrics.NewSorensenDice())
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

				client, err := internal.Connect(config)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
				}

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
				client, err := internal.Connect(config)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
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
	Name:    "object",
	Aliases: []string{"o"},
	Usage:   "Manage objects",
	Subcommands: []*cli.Command{
		{
			Name:      "toggle",
			Aliases:   []string{"t"},
			Usage:     "Toggle object's state (on/off)",
			ArgsUsage: "<object>",
			Action: func(c *cli.Context) error {
				object := c.Args().First()
				if object == "" {
					return fmt.Errorf("object not specified")
				}

				client, err := internal.Connect(config)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
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
			BashComplete: func(c *cli.Context) {
				client, err := internal.Connect(config)
				if err != nil {
					panic(err)
				}

				// TODO: Save to cache because it's slow
				userConfig, err := client.GetUserConfig()
				if err != nil {
					panic(err)
				}

				for _, cell := range userConfig.Cells {
					fmt.Println(cell.Name)
				}
			},
		},
		{
			Name:      "set",
			Aliases:   []string{"s"},
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

				client, err := internal.Connect(config)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
				}

				objectID, err := strconv.Atoi(object)
				if err != nil {
					// string
					slog.Info("looking for object", slog.String("name", object))

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

					slog.Info("selected object",
						slog.String("name", bestObject.Name),
						slog.Int("id", bestObject.ID),
						slog.Int("confidence", int(bestScore*100)),
					)

					value := api.MapLighting(value)
					err = client.SendEvent(bestObject.ID, value)
					if err != nil {
						return fmt.Errorf("failed to send event to object %#v with id %d", bestObject.Name, bestObject.ID)
					} else {
						slog.Info("sent event to object",
							slog.String("name", bestObject.Name),
							slog.Int("id", bestObject.ID),
							slog.String("value", value),
						)
						return nil
					}
				} else {
					err = client.SendEvent(objectID, api.MapLighting(value))
					if err != nil {
						return fmt.Errorf("sent event to object with id %d: %v", objectID, err)
					}

					slog.Info("sent event to object", slog.Int("id", objectID), slog.String("value", api.MapLighting(value)))
					return nil
				}
			},
		},
	},
}

var experimentCommand = cli.Command{
	Name:     "experiment",
	Usage:    "Shell completion stuff",
	HideHelp: true,
	// Flags: []cli.Flag{
	// 	&cli.BoolFlag{
	// 		Name:  "option",
	// 		Usage: "Select one of many options.",
	// 		// BashComplete: func(c *cli.Context) {},
	// 	},
	// 	&cli.BoolFlag{
	// 		Name:  "user",
	// 		Usage: "Print config set in the client apps",
	// 	},
	// },
	Action: func(c *cli.Context) error {
		log.Print("action executed")

		return nil
	},
}
