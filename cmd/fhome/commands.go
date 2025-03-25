package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/urfave/cli/v3"
)

// bestObjectMatch returns the cell with the highest similarity score to the given object and the score itself.
//
// If no objects match at all (i.e., bestScore is 0), then this method returns nil.
func bestObjectMatch(object string, config *api.Config) (bestObject *api.Cell, bestScore float64) {
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
	Commands: []*cli.Command{
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
				&cli.BoolFlag{
					Name:  "merged",
					Usage: "Print merged config (system + user)",
				},
				&cli.BoolFlag{
					Name:  "glance",
					Usage: "Print nice, at glance status of system",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				if cmd.Bool("system") && cmd.Bool("user") {
					return fmt.Errorf("cannot use both --system and --user")
				}

				config := loadConfig()

				client, err := highlevel.Connect(ctx, config, nil)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
				}

				sysConfig, err := client.GetSystemConfig(ctx)
				if err != nil {
					return fmt.Errorf("failed to get sysConfig: %v", err)
				}
				log.Println("got system config")

				userConfig, err := client.GetUserConfig(ctx)
				if err != nil {
					return fmt.Errorf("failed to get user config: %v", err)
				}
				log.Println("got user config")

				apiConfig, err := api.MergeConfigs(userConfig, sysConfig)
				if err != nil {
					return fmt.Errorf("failed to merge configs: %v", err)
				}
				_ = apiConfig

				if cmd.Bool("system") {
					w := tabwriter.NewWriter(os.Stdout, 8, 8, 0, ' ', 0)
					defer w.Flush()

					fmt.Fprintf(w, "id\tdt\tpreset\tstyle\tperm\tmin\tmax\tstep\tdesc\n")

					cells := sysConfig.Response.MobileDisplayProperties.Cells
					for _, cell := range cells {
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", cell.ID, cell.DisplayType, cell.Preset, cell.Style, cell.Permission, cell.MinValue, cell.MaxValue, cell.Step, cell.Desc)
					}
				} else if cmd.Bool("user") {
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
				} else if cmd.Bool("merged") {
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
				} else if cmd.Bool("glance") {
					// We want to see the real values of the system resources.
					// To do that, we need to send the "statustouches" action and
					// wait for its response.

					msg, err := client.SendAction(ctx, api.ActionStatusTouches)
					if err != nil {
						return fmt.Errorf("failed to send action: %v", err)
					}

					var touchesResponse api.StatusTouchesChangedResponse

					err = json.Unmarshal(msg.Raw, &touchesResponse)
					if err != nil {
						slog.Error("failed to unmarshal message", slog.Any("error", err))
						return err
					}

					cells := make([]struct {
						Name  string
						Value int
					}, 0)

					mdCells := sysConfig.Response.MobileDisplayProperties.Cells
					for _, cell := range mdCells {
						if cell.DisplayType != api.Percentage {
							continue
						}
						if !strings.HasPrefix(cell.Step, "0x60") {
							continue
						}

						var cellValue *api.CellValue
						for _, cv := range touchesResponse.Response.CellValues {
							if cv.ID == cell.ID {
								cellValue = &cv
								break
							}
						}
						if cellValue == nil {
							slog.Error("failed to find corresponding cell value", slog.String("cell", cell.ID))
							continue
						}

						slog.Info("remapping lighting value", slog.String("cell", cell.Desc), slog.String("value", cellValue.Value), slog.String("step", cell.Step))
						val, err := api.RemapLighting(cellValue.Value)
						if err != nil {
							slog.Error(
								"error remapping lighting value",
								slog.Group("cell", slog.String("id", cell.ID), slog.String("desc", cell.Desc)),
								slog.Any("error", err),
							)
							continue
						}

						cells = append(cells, struct {
							Name  string
							Value int
						}{Name: cell.Desc, Value: val})
					}

					text := "Oto status oświetlenia:\n"
					for _, cell := range cells {
						text += fmt.Sprintf("• %s: %d%%\n", cell.Name, cell.Value)
					}
					fmt.Print(text)
				}

				return nil
			},
		},
	},
}

var eventCommand = cli.Command{
	Name:  "event",
	Usage: "Manage events",
	Commands: []*cli.Command{
		{
			Name:  "watch",
			Usage: "Print all incoming messages",
			Action: func(ctx context.Context, c *cli.Command) error {
				config := loadConfig()

				client, err := highlevel.Connect(ctx, config, nil)
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
	Commands: []*cli.Command{
		{
			Name:      "toggle",
			Aliases:   []string{"t"},
			Usage:     "Toggle object's state (on/off)",
			ArgsUsage: "<object>",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				object := cmd.Args().First()
				if object == "" {
					return fmt.Errorf("object not specified")
				}

				config := loadConfig()

				client, err := highlevel.Connect(ctx, config, nil)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
				}

				objectID, err := strconv.Atoi(object)
				if err != nil {
					// string
					log.Printf("looking for object with name %q", object)

					userConfig, err := client.GetUserConfig(ctx)
					if err != nil {
						return fmt.Errorf("failed to get user config: %v", err)
					}

					sysConfig, err := client.GetSystemConfig(ctx)
					if err != nil {
						return fmt.Errorf("failed to get system config: %v", err)
					}

					config, err := api.MergeConfigs(userConfig, sysConfig)
					if err != nil {
						return fmt.Errorf("failed to merge configs: %v", err)
					}

					bestObject, bestScore := bestObjectMatch(object, config)
					if bestObject == nil {
						return fmt.Errorf("no matching object found, confidence is %d%%", int(bestScore*100))
					}

					log.Printf("selected object %q with id %d with %d%% confidence\n", bestObject.Name, bestObject.ID, int(bestScore*100))

					err = client.SendEvent(ctx, bestObject.ID, api.ValueToggle)
					if err != nil {
						return fmt.Errorf("failed to send event to object %q with id %d", bestObject.Name, bestObject.ID)
					}

					log.Printf("sent event %s to object %q with id %d\n", api.ValueToggle, bestObject.Name, bestObject.ID)
					return nil
				} else {
					// int

					err = client.SendEvent(ctx, objectID, api.ValueToggle)
					if err != nil {
						return fmt.Errorf("failed to send event to object with id %d: %v", objectID, err)
					}

					log.Println("sent event to object with id", objectID)
					return nil
				}
			},
			ShellComplete: func(ctx context.Context, cmd *cli.Command) {
				config := loadConfig()
				client, err := highlevel.Connect(ctx, config, nil)
				if err != nil {
					panic(err)
				}

				// TODO: Save to cache because it's slow
				userConfig, err := client.GetUserConfig(ctx)
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
			Action: func(ctx context.Context, cmd *cli.Command) error {
				object := cmd.Args().Get(0)
				if object == "" {
					return fmt.Errorf("object not specified")
				}

				value, err := strconv.Atoi(cmd.Args().Get(1))
				if err != nil {
					return fmt.Errorf("invalid value: %v", err)
				}

				config := loadConfig()

				client, err := highlevel.Connect(ctx, config, nil)
				if err != nil {
					return fmt.Errorf("failed to create api client: %v", err)
				}

				objectID, err := strconv.Atoi(object)
				if err != nil {
					// string
					slog.Info("looking for object", slog.String("name", object))

					userConfig, err := client.GetUserConfig(ctx)
					if err != nil {
						return fmt.Errorf("failed to get user config: %v", err)
					}

					sysConfig, err := client.GetSystemConfig(ctx)
					if err != nil {
						return fmt.Errorf("failed to get system config: %v", err)
					}

					config, err := api.MergeConfigs(userConfig, sysConfig)
					if err != nil {
						return fmt.Errorf("failed to merge configs: %v", err)
					}

					bestObject, bestScore := bestObjectMatch(object, config)
					if bestObject == nil {
						return fmt.Errorf("no matching object found, confidence is %d%%", int(bestScore*100))
					}

					slog.Info("found best match",
						slog.Int("confidence", int(bestScore*100)),
						slog.Group("object", slog.String("name", bestObject.Name), slog.Int("id", bestObject.ID)),
					)

					value := api.MapLighting(value)
					err = client.SendEvent(ctx, bestObject.ID, value)
					if err != nil {
						return fmt.Errorf("failed to send event to object %q with id %d", bestObject.Name, bestObject.ID)
					} else {
						slog.Info("sent event to object",
							slog.String("name", bestObject.Name),
							slog.Int("id", bestObject.ID),
							slog.String("value", value),
						)
						return nil
					}
				} else {
					err = client.SendEvent(ctx, objectID, api.MapLighting(value))
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
