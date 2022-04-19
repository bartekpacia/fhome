package main

import (
	"log"
	"os"

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
		// verbose := c.Bool("verbose")

		// err := fhome.List(verbose)
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
		// verbose := c.Bool("verbose")

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
