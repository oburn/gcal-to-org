package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const defaultStoreDirFlag = "$HOME/.local/share/gcal-to-org"

func main() {
	app := &cli.App{
		Name:  "gcal-to-org",
		Usage: "Google Calendar tools",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "store",
				Value: defaultStoreDirFlag,
				Usage: "Directory to store OAuth tokens",
			},
		},
		Commands: []*cli.Command{
			generateCommand(),
			roomCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
