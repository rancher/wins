package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	planapi "github.com/rancher/rancher/pkg/plan"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// planKeyData matches k8splan.PlanKey ("plan"), the plan Secret data key. It is not exported by
// planapi, so it is duplicated here rather than importing the internal k8splan package.
const planKeyData = "plan"

// utf8BOM is the UTF-8 byte order mark that PowerShell's Set-Content/Out-File -Encoding utf8
// writes at the start of a file on Windows PowerShell 5.1 (though not on PowerShell 7+, which
// treats "utf8" as BOM-less). encoding/json does not skip a leading BOM, so it must be stripped
// here rather than relying on every plan-authoring caller to avoid emitting one.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// stripUTF8BOM removes a leading UTF-8 byte order mark, if present.
func stripUTF8BOM(raw []byte) []byte {
	return bytes.TrimPrefix(raw, utf8BOM)
}

func applyPlanCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply-plan",
		Usage: "write the plan and plan-state keys to the plan Secret and print the computed checksum",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "path to the plan JSON file", Required: true},
			&cli.StringFlag{Name: "state", Usage: "plan-state to write (pending, in-progress, ...)", Value: string(planapi.PlanStatePending)},
			// Setting the cancellation or pause annotation in the same write as the plan
			// content is the only way to guarantee the agent never observes an intermediate
			// secret version with the plan pending but not yet interrupted: two separate
			// planctl invocations (apply-plan, then annotate) each pay real process-startup
			// cost, while the agent's watch loop can pick up and fully execute a trivial
			// instruction in a fraction of that time.
			&cli.StringFlag{Name: "canceled", Usage: `also set the plan.cattle.io/canceled annotation to "true" in the same write`},
			&cli.StringFlag{Name: "paused", Usage: `also set the plan.cattle.io/paused annotation to "true" in the same write`},
		},
		Action: func(cCtx *cli.Context) error {
			clientset, err := newClientset(cCtx)
			if err != nil {
				return err
			}
			ctx := context.Background()

			raw, err := os.ReadFile(cCtx.String("file"))
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}
			raw = stripUTF8BOM(raw)

			checksum := planapi.Checksum(raw)

			// The agent concurrently writes to this same Secret (plan-state transitions,
			// feedback, probe statuses), so a plain Get-then-Update races on resourceVersion
			// and returns a 409 Conflict. Retry the whole read-modify-write, the same pattern
			// system-agent's own writeInterruptOutcome uses.
			err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("getting plan secret %s/%s: %w", namespace, secretName, err)
				}

				if secret.Data == nil {
					secret.Data = map[string][]byte{}
				}
				secret.Data[planKeyData] = raw
				secret.Data[planapi.PlanStateKey] = []byte(cCtx.String("state"))

				if cCtx.IsSet("canceled") {
					if secret.Annotations == nil {
						secret.Annotations = map[string]string{}
					}
					setOrRemoveAnnotation(secret.Annotations, planapi.PlanCanceledAnnotation, cCtx.String("canceled"))
				}
				if cCtx.IsSet("paused") {
					if secret.Annotations == nil {
						secret.Annotations = map[string]string{}
					}
					setOrRemoveAnnotation(secret.Annotations, planapi.PlanPausedAnnotation, cCtx.String("paused"))
				}

				if _, err := clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
					return fmt.Errorf("updating plan secret %s/%s: %w", namespace, secretName, err)
				}
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(checksum)
			return nil
		},
	}
}
