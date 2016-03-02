package main

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "ttcli"
	app.Usage = "for uploading data to Traintracks from the command line"
	app.Action = func(c *cli.Context) {
		println("hello world")
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "auth-file",
			Usage: "file for authentication",
		},
		cli.StringFlag{
			Name:  "config-file",
			Usage: "config file with type information of file",
		},
		cli.StringFlag{
			Name:  "event-type",
			Usage: "optional flag for explicitly using name for event type",
		},
	}

	app.Run(os.Args)
}
