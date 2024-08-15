package systemagent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/system-agent/pkg/config"
	"github.com/rancher/system-agent/pkg/image"
	"github.com/rancher/system-agent/pkg/k8splan"
	"github.com/rancher/system-agent/pkg/localplan"
	"github.com/rancher/system-agent/pkg/version"
	"github.com/sirupsen/logrus"
)

type Agent struct {
	cfg           *config.AgentConfig
	StrictTLSMode bool
}

func (a *Agent) Run(ctx context.Context) error {

	if a.cfg == nil {
		logrus.Info("Rancher System Agent configuration not found, not starting system agent.")
		return nil
	}

	logrus.Infof("Rancher System Agent version %s is starting", version.FriendlyVersion())

	if !a.cfg.LocalEnabled && !a.cfg.RemoteEnabled {
		return errors.New("local and remote were both not enabled. exiting, as one must be enabled")
	}

	logrus.Infof("Setting %s as the working directory", a.cfg.WorkDir)

	imageUtil := image.NewUtility(a.cfg.ImagesDir, a.cfg.ImageCredentialProviderConfig, a.cfg.ImageCredentialProviderBinDir, a.cfg.AgentRegistriesFile)
	// Currently we do not support the 'interlockDir' on Windows, as the system-agent install script does not yet utilize those files
	applier := applyinator.NewApplyinator(a.cfg.WorkDir, a.cfg.PreserveWorkDir, a.cfg.AppliedPlanDir, "", imageUtil)
	if a.cfg.RemoteEnabled {
		logrus.Infof("Starting remote watch of plans")

		var connInfo config.ConnectionInfo

		if err := config.Parse(a.cfg.ConnectionInfoFile, &connInfo); err != nil {
			return fmt.Errorf("unable to parse connection info file: %v", err)
		}

		k8splan.Watch(ctx, *applier, connInfo, a.StrictTLSMode)
	}

	if a.cfg.LocalEnabled {
		logrus.Infof("Starting local watch of plans in %s", a.cfg.LocalPlanDir)
		localplan.WatchFiles(ctx, *applier, a.cfg.LocalPlanDir)
	}

	return nil
}

func New(cfg *config.AgentConfig) *Agent {
	return &Agent{
		cfg: cfg,
	}
}
