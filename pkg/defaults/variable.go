package defaults

import (
	"path/filepath"
)

var (
	AppVersion          = "dev"
	AppCommit           = "0000000"
	ConfigPath          = filepath.Join("c:/", "etc", "rancher", "wins", "config")
	UpgradeWatchingPath = filepath.Join("c:/", "etc", "rancher", "wins", "wins.exe")
)
