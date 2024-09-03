package app

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/server/config"
	"github.com/rancher/wins/pkg/apis"
	"github.com/rancher/wins/pkg/csiproxy"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/profilings"
	"github.com/rancher/wins/pkg/systemagent"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var _runFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "register",
		Usage: "[optional] Register to the Windows Service",
	},
	&cli.BoolFlag{
		Name:  "unregister",
		Usage: "[optional] Unregister from the Windows Service",
	},
	&cli.StringFlag{
		Name:  "config",
		Usage: "[optional] Specifies the path of the configuration",
		Value: defaults.ConfigPath,
	},
	&cli.StringFlag{
		Name:  "profile",
		Usage: "[optional] Specifies the name of profile to capture (none|cpu|heap|goroutine|threadcreate|block|mutex)",
		Value: "none",
	},
	&cli.StringFlag{
		Name:  "profile-output",
		Usage: "[optional] Specifies the name of the file to write the profile to",
		Value: "profile.pprof",
	},
}

func _profilingInit(cliCtx *cli.Context) error {
	return profilings.Init(cliCtx.String("profile"), cliCtx.String("profile-output"))
}

func _profilingFlush(cliCtx *cli.Context) error {
	return profilings.Flush(cliCtx.String("profile"), cliCtx.String("profile-output"))
}

func _runAction(cliCtx *cli.Context) error {
	defer panics.Log()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// register / unregister service
	register := cliCtx.Bool("register")
	unregister := cliCtx.Bool("unregister")
	if register {
		if unregister {
			return errors.New("failed to execute: --register and --unregister could not use together")
		}

		err := registerService()
		if err != nil {
			return errors.Wrap(err, "failed to register service")
		}
		return nil
	}
	if unregister {
		err := unregisterService()
		if err != nil {
			return errors.Wrap(err, "failed to unregister service")
		}
		return nil
	}

	// parse config
	cfg := config.DefaultConfig()
	cfgPath := cliCtx.String("config")
	err := config.LoadConfig(cfgPath, cfg)
	if err != nil {
		return errors.Wrapf(err, "failed to load config from %s", cfgPath)
	}

	serverOptions := []grpc.ServerOption{
		grpc.ConnectionTimeout(5 * time.Second),
	}

	serverOptions, err = setupGRPCServerOptions(serverOptions, cfg)
	if err != nil {
		return errors.Wrap(err, "failed to setup grpc middlewares")
	}

	logrus.Debugf("Proxy port whitelist: %v", cfg.WhiteList.ProxyPorts)
	server, err := apis.NewServer(cfg.Listen, serverOptions, cfg.Proxy, cfg.WhiteList.ProxyPorts)
	if err != nil {
		return errors.Wrap(err, "failed to create server")
	}

	// adding system agent
	agent := systemagent.New(cfg.SystemAgent)

	// Determine if the agent should use strict verification
	agent.StrictTLSMode = cfg.AgentStrictTLSMode

	//checking if CSI Proxy has config, if so enables it.
	if cfg.CSIProxy != nil {
		logrus.Infof("CSI Proxy will be enabled as a Windows service.")
		csi, err := csiproxy.New(cfg.CSIProxy, cfg.TLSConfig)
		if err != nil {
			return err
		}
		if err := csi.Enable(); err != nil {
			return err
		}
	}

	err = runService(ctx, server, agent)
	if err != nil {
		return errors.Wrap(err, "failed to run server")
	}

	return nil
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:   "run",
		Usage:  "Run application",
		Flags:  _runFlags,
		Before: _profilingInit,
		Action: _runAction,
		After:  _profilingFlush,
	}
}
