package main

import (
	"log"
	"os"

	"github.com/bartekpacia/fhome/cfg"
	"github.com/urfave/cli/v2"
)

var config cfg.Config

// 1. Flag
// 2. Env var
// 3. $HOME/.config/fhome/config.toml
// 4. /etc/fhome/config.toml

func init() {
	log.SetFlags(0)

	// viper.SetConfigName("config")
	// viper.SetConfigType("toml")
	// viper.AddConfigPath(".")
	// viper.AddConfigPath("$HOME/.config/fhome/")
	// viper.AddConfigPath("/etc/fhome/")
	// if err := viper.ReadInConfig(); err != nil {
	// 	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
	// 		log.Fatalf("failed to read in config: %v\n", err)
	// 	}
	// }

	// config = cfg.Config{
	// 	Email:            viper.GetString("FHOME_EMAIL"),
	// 	CloudPassword:    viper.GetString("FHOME_CLOUD_PASSWORD"),
	// 	ResourcePassword: viper.GetString("FHOME_RESOURCE_PASSWORD"),
	// }

	// err := config.Verify()
	// if err != nil {
	// 	log.Fatalf("failed to load config: %v\n", err)
	// }
}

func main() {
	// topLevelFlags := []cli.Flag{
	// 	altsrc.NewStringFlag(&cli.StringFlag{Name: "email"}),
	// 	altsrc.NewStringFlag(&cli.StringFlag{Name: "cloud-password"}),
	// 	altsrc.NewStringFlag(&cli.StringFlag{Name: "resource-password"}),
	// }

	app := &cli.App{
		Name: "fhome",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "email",
				EnvVars:  []string{"FHOME_EMAIL"},
				FilePath: "",
			},
		},
		// Before: altsrc.InitInputSourceWithContext(topLevelFlags, altsrc.NewYamlSourceFromFlagFunc("load")),
		Usage: "Interact with smart home devices connected to F&Home",
		Commands: []*cli.Command{
			&configCommand,
			&eventCommand,
			&objectCommand,
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
