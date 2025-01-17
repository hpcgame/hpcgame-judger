package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

func pollCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "poll",
		Usage:  "Poll a solution",
		Action: pollHandler,
	})
}

func pollHandler(c *cli.Context) error {
	sp, err := client.Poll(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", sp)
	return nil
}
