package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// newClientset builds a Kubernetes clientset from the --kubeconfig flag.
func newClientset(cCtx *cli.Context) (*kubernetes.Clientset, error) {
	kubeconfigPath := cCtx.String("kubeconfig")
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("--kubeconfig is required")
	}

	restCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building rest config from %s: %w", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("building clientset: %w", err)
	}

	return clientset, nil
}
