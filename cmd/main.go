package main

import (
	"fmt"
	"log"
	"os"

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
		client, err := fhome.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create fhome client: %v", err)
		}
		env := Env{}
		env.Load()

		// TODO: don't pass password hash
		err = client.OpenClientSession(env.email, env.password, env.passwordHash)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenClientToResourceSession()
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		file, err := client.GetUserConfig()
		if err != nil {
			return fmt.Errorf("failed to get user config: %v", err)
		}

		log.Println("successfully got user config")

		fmt.Printf("there are %d cells\n", len(file.Cells))
		for _, cell := range file.Cells {
			fmt.Printf("id: %3d, name: %s\n", cell.ObjectID, cell.Name)
		}
		return nil
	},
}

var toggleCommand = cli.Command{
	Name:  "toggle",
	Usage: "toggle an object on/off",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "object-id",
			Aliases: []string{"id"},
			Value:   "",
			Usage:   "id of object to toggle",
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

		client, err := fhome.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create fhome client: %v", err)
		}
		env := Env{}
		env.Load()

		// TODO: don't pass password hash
		err = client.OpenClientSession(env.email, env.password, env.passwordHash)
		if err != nil {
			return fmt.Errorf("failed to open client session: %v", err)
		}

		log.Println("successfully opened client session")

		_, err = client.GetMyResources()
		if err != nil {
			return fmt.Errorf("failed to get my resources: %v", err)
		}

		log.Println("successfully got my resources")

		err = client.OpenClientToResourceSession()
		if err != nil {
			return fmt.Errorf("failed to open client to resource session: %v", err)
		}

		log.Println("successfully opened client to resource session")

		err = client.XEvent(objectID, "0x4001", "HEX")
		if err != nil {
			return fmt.Errorf("failed to send xevent to object with id %d: %v", objectID, err)
		}

		log.Println("successfully sent xevent to object with id", objectID)

		return nil
	},
}

func main() {
	app := &cli.App{
		Name:  "fhome",
		Usage: "interact with F&Home API",
		Commands: []*cli.Command{
			&listCommand,
			&toggleCommand,
		},
		CommandNotFound: func(c *cli.Context, command string) {
			log.Printf("invalid command '%s'. See 'fhome --help'\n", command)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
