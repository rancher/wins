package defaults

import (
	"path/filepath"
)

// application
var (
	AppVersion = "dev"
	AppCommit  = "0000000"
)

// configuration
var (
	ConfigPath = filepath.Join("c:/", "etc", "rancher", "wins", "config")
)

// upgrading
var (
	UpgradeWatchingPath = filepath.Join("c:/", "etc", "rancher", "wins", "wins.exe")
)
