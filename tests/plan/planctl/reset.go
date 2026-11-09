package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func resetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "delete and recreate the plan Secret, producing a new UID, forcing the agent to re-apply",
		Action: func(cCtx *cli.Context) error {
			clientset, err := newClientset(cCtx)
			if err != nil {
				return err
			}
			return resetPlanSecret(context.Background(), clientset)
		},
	}
}

// resetPlanSecret deletes the plan Secret if it exists and recreates it empty, producing a new
// UID. It is shared by the reset command and bootstrap: bootstrap must not leave a stale
// plan/plan-state/annotations from a prior, possibly crashed, run in place, exactly like a
// standalone reset guarantees between specs.
func resetPlanSecret(ctx context.Context, clientset *kubernetes.Clientset) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting plan secret %s/%s: %w", namespace, secretName, err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("recreating plan secret %s/%s: %w", namespace, secretName, err)
	}
	return nil
}
