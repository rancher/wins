package upgrade

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/cmd/upgrade/internal/powershell"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var _upgradeFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "wins-args",
		Usage: "[optional] Arguments to pass onto wins srv app run --register",
	},
	cli.BoolFlag{
		Name:  "debug",
		Usage: "[optional] whether to print debugging logs from performing the upgrade",
	},
}

const upgradePS1Fmt = `& {
	$ErrorActionPreference = "Stop";

	$newBinPath = "%[1]s"
	$winsArgs = "%[2]s"

	$winsCmd = (Get-CimInstance Win32_Service -Filter 'Name = "rancher-wins"')

	$winsSvc = Get-Service -Name rancher-wins
	if ($winsSvc) {
		$winsSvc | Stop-Service;
	}
	$winsPrc = Get-Process -Name wins -ErrorAction SilentlyContinue
	if ($winsPrc) {
		$winsPrc | Stop-Process -Force
	}

	if ($winsCmd -ne $null) {
		$winsCmdPath = $winsCmd.PathName
		$currBinPath = $winsCmdPath.Split(' ')[0]
		if ($currBinPath -ne $newBinPath) {
			Copy-Item -Recurse -Force -Path $newBinPath -Destination $currBinPath | Out-Null
		}
	} else {
		$currBinPath = $newBinPath
	}

	Invoke-Expression "& $currBinPath srv app run --register $winsArgs"
	Start-Service -Name rancher-wins;
	Write-Host "Upgraded rancher-wins"
}`

const restartServicePS1 = `& {
	$ErrorActionPreference = "Stop";

	$winsCmd = (Get-CimInstance Win32_Service -Filter 'Name = "rancher-wins"')

	if ($winsCmd -ne $null) {
		$winsCmdPath = $winsCmd.PathName
		$winsSvc = Get-Service -Name rancher-wins
		if ($winsSvc) {
			$winsSvc | Stop-Service;
		}
		$winsPrc = Get-Process -Name wins -ErrorAction SilentlyContinue
		if ($winsPrc) {
			$winsPrc | Stop-Process -Force
		}
		Invoke-Expression "& $winsCmdPath --register"
		Start-Service -Name rancher-wins;
		Write-Host "Restarted rancher-wins"
	}
}`

func _upgradeRequestParser(cliCtx *cli.Context) (err error) {
	// validate
	winsArgs := cliCtx.String("wins-args")
	if strings.Contains(winsArgs, "--unregister") {
		return fmt.Errorf(`cannot provide "--unregister" to --wins-args`)
	}
	if !strings.Contains(winsArgs, "--register") {
		return fmt.Errorf(`cannot provide "--register" to --wins-args`)
	}
	return nil
}

func _upgradeAction(cliCtx *cli.Context) (err error) {
	if cliCtx.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	binPath := os.Args[0]
	winsArgs := cliCtx.String("wins-args")
	out, err := powershell.RunCommandf(upgradePS1Fmt, binPath, winsArgs)
	if len(out) > 0 {
		logrus.Debugf("logs from upgrade.ps1 \n%s\nEOF", out)
	}
	if err != nil {
		logrus.Errorf("upgrade failed, attempting to ensure rancher-wins service is not stopped: %v", err)
		out, restartServiceErr := powershell.RunCommand(restartServicePS1)
		if restartServiceErr != nil {
			if len(out) > 0 {
				logrus.Debugf("logs from restartService.ps1 \n%s\nEOF", out)
			}
			logrus.Errorf("unable to restart rancher-wins service: %v", restartServiceErr)
		}
		return err
	}
	logrus.Info("upgrade succeeded")
	return nil
}
