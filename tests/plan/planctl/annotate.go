package main

import (
	"context"
	"fmt"

	planapi "github.com/rancher/rancher/pkg/plan"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func annotateCommand() *cli.Command {
	return &cli.Command{
		Name:  "annotate",
		Usage: "set or clear the plan.cattle.io/canceled or plan.cattle.io/paused annotation on the plan Secret",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "canceled", Usage: `set the plan.cattle.io/canceled annotation to "true", "false", or "" to remove it`},
			&cli.StringFlag{Name: "paused", Usage: `set the plan.cattle.io/paused annotation to "true", "false", or "" to remove it`},
		},
		Action: func(cCtx *cli.Context) error {
			clientset, err := newClientset(cCtx)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// The agent concurrently writes to this same Secret (plan-state transitions,
			// feedback, probe statuses), so a plain Get-then-Update races on resourceVersion
			// and returns a 409 Conflict. Retry the whole read-modify-write, the same pattern
			// system-agent's own writeInterruptOutcome uses.
			return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("getting plan secret %s/%s: %w", namespace, secretName, err)
				}

				if secret.Annotations == nil {
					secret.Annotations = map[string]string{}
				}

				if cCtx.IsSet("canceled") {
					setOrRemoveAnnotation(secret.Annotations, planapi.PlanCanceledAnnotation, cCtx.String("canceled"))
				}
				if cCtx.IsSet("paused") {
					setOrRemoveAnnotation(secret.Annotations, planapi.PlanPausedAnnotation, cCtx.String("paused"))
				}

				if _, err := clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
					return fmt.Errorf("updating plan secret %s/%s: %w", namespace, secretName, err)
				}
				return nil
			})
		},
	}
}

func setOrRemoveAnnotation(annotations map[string]string, key, value string) {
	if value == "" {
		delete(annotations, key)
		return
	}
	annotations[key] = value
}
