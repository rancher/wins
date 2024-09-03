package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/suc/pkg"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {

	defer panics.Log()
	app := cli.NewApp()
	app.Version = defaults.AppVersion
	app.Name = defaults.WindowsSUCName
	app.Usage = "A way to modify rancher-wins via the Rancher System Upgrade Controller"
	app.Action = pkg.Run
	app.Description = fmt.Sprintf(`%s (%s)`, defaults.WindowsSUCName, defaults.AppCommit)
	app.Writer = colorable.NewColorableStdout()
	app.ErrWriter = colorable.NewColorableStderr()
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if c.Bool("quiet") {
			logrus.SetOutput(io.Discard)
		} else {
			logrus.SetOutput(c.App.Writer)
		}

		logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
		return nil
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Turn on verbose debug logging",
		},
		&cli.BoolFlag{
			Name:  "quiet",
			Usage: "Turn off all logging",
		},
	}

	if err := app.Run(os.Args); err != nil && err != io.EOF {
		logrus.Fatal(err)
	}
}
