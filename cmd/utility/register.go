package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func registerCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "register",
		Usage:  "Register a runner",
		Action: registerHandler,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Name of the runner",
				Required: true,
				EnvVars:  []string{"RUNNER_NAME"},
			},
			&cli.StringSliceFlag{
				Name:     "label",
				Aliases:  []string{"l"},
				Usage:    "Labels of the runner",
				Required: true,
				EnvVars:  []string{"RUNNER_LABELS"},
			},
			&cli.StringFlag{
				Name:        "version",
				Aliases:     []string{"V"},
				Usage:       "Version of the runner",
				Value:       "v0.1.0-alpha",
				DefaultText: "v0.1.0-alpha",
				EnvVars:     []string{"RUNNER_VERSION"},
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "Registration token",
				Required: true,
				EnvVars:  []string{"RUNNER_TOKEN"},
			},
			&cli.StringFlag{
				Name:        "write-file",
				Aliases:     []string{"w"},
				Usage:       "Write to file",
				Value:       "runner.env",
				DefaultText: "runner.env",
			},
		},
	})
}

func registerHandler(c *cli.Context) error {
	id, key, err := client.Register(context.Background(), c.String("name"),
		c.StringSlice("label"), c.String("version"), c.String("token"))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Runner ID: ", id)
	log.Println("Runner Key:", key)
	log.Println("Writing to runner environment file...")
	err = writeRunnerEnv(c.String("write-file"), id, key)

	return err
}

func writeRunnerEnv(file, id, key string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("RUNNER_ID=" + id + "\n" + "RUNNER_KEY=" + key + "\n")
	return err
}
