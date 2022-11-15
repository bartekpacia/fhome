package main

import (
	"log"
	"os"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/cfg"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

var (
	client *api.Client
	config cfg.Config
)

func init() {
	log.SetFlags(0)
	var err error

	client, err = api.NewClient()
	if err != nil {
		log.Fatalf("failed to create api api client: %v\n", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/fh/")
	viper.AddConfigPath("/etc/fh/")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("failed to read in config: %v\n", err)
		}
	}

	config = cfg.Config{
		Email:            viper.GetString("FHOME_EMAIL"),
		CloudPassword:    viper.GetString("FHOME_CLOUD_PASSWORD"),
		ResourcePassword: viper.GetString("FHOME_RESOURCE_PASSWORD"),
	}

	err = config.Verify()
	if err != nil {
		log.Fatalf("failed to load config: %v\n", err)
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
