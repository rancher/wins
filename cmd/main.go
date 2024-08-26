package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/rancher/wins/cmd/stackdump"

	"github.com/mattn/go-colorable"
	"github.com/rancher/wins/cmd/client"
	"github.com/rancher/wins/cmd/server"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	defer panics.Log()

	app := cli.NewApp()
	app.Version = defaults.AppVersion
	app.Name = defaults.WindowsServiceName
	app.Usage = "A way to operate the Windows host inside the Windows container"
	app.Description = fmt.Sprintf(`%s Component (%s)`, defaults.WindowsServiceDisplayName, defaults.AppCommit)
	app.Writer = colorable.NewColorableStdout()
	app.ErrWriter = colorable.NewColorableStderr()
	app.CommandNotFound = func(cliCtx *cli.Context, s string) {
		_, err := fmt.Fprintf(cliCtx.App.Writer, "Invalid Command: %s \n\n", s)
		if err != nil {
			return
		}
		if pcliCtx := cliCtx.Lineage(); pcliCtx[1] == nil {
			cli.ShowAppHelpAndExit(cliCtx, 1)
		} else {
			cli.ShowCommandHelpAndExit(cliCtx, cliCtx.Command.Name, 1)
		}
	}
	app.OnUsageError = func(cliCtx *cli.Context, err error, isSubcommand bool) error {
		_, err = fmt.Fprintf(cliCtx.App.Writer, "Incorrect Usage: %s \n\n", err.Error())
		if err != nil {
			return err
		}
		if isSubcommand {
			err := cli.ShowSubcommandHelp(cliCtx)
			if err != nil {
				return err
			}
		} else {
			err := cli.ShowAppHelp(cliCtx)
			if err != nil {
				return err
			}
		}
		return nil
	}
	app.Before = func(cliCtx *cli.Context) error {
		logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
		logrus.SetOutput(cliCtx.App.Writer)
		return nil
	}

	app.Commands = []*cli.Command{
		server.NewCommand(),
		client.NewCommand(),
		stackdump.NewCommand(),
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Turn on verbose debug logging",
		},
		&cli.BoolFlag{
			Name:  "quiet",
			Usage: "Turn on off all logging",
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if c.Bool("quiet") {
			logrus.SetOutput(ioutil.Discard)
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil && err != io.EOF {
		logrus.Fatal(err)
	}

	logrus.Debug("Finished")
}
