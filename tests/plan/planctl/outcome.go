package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	planapi "github.com/rancher/rancher/pkg/plan"
	"github.com/rancher/system-agent/pkg/k8splan"
	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// outcomeResult is the decode boundary between the Kubernetes Secret's byte-oriented data and
// the plain-JSON shape the Pester specs assert against.
type outcomeResult struct {
	// ResourceVersion is the Secret's own metadata.resourceVersion, not plan data. It lets a
	// spec assert that a monitoring-only reconcile (a terminal, canceled or failed plan)
	// performs zero writes at all, a stronger invariant than any individual field staying
	// unchanged: an unrelated field could theoretically be rewritten to the same value without
	// resourceVersion changing, but resourceVersion itself only advances on an actual write.
	ResourceVersion string `json:"resourceVersion"`

	PlanState       string `json:"planState"`
	PlanRevision    string `json:"planRevision"`
	AppliedChecksum string `json:"appliedChecksum"`
	FailedChecksum  string `json:"failedChecksum"`
	FailureCount    int    `json:"failureCount"`
	SuccessCount    int    `json:"successCount"`
	LastApplyTime   string `json:"lastApplyTime"`

	Checkpoint *checkpointOutcome `json:"checkpoint"`

	Output       map[string]string `json:"output"`
	FailedOutput map[string]string `json:"failedOutput"`

	// ProbeStatuses and PeriodicOutput are present regardless of plan-state: probe evaluation
	// and, once due, periodic instruction execution both continue on every reconcile for a
	// non-canceled, non-failed plan (including a paused one, whose reconcile still merges probe
	// statuses via the write-once interrupt-recording path). A canceled or failed plan is
	// terminal and reconciles in monitoring-only mode, where probe statuses still update but
	// periodic instructions never execute again.
	ProbeStatuses  map[string]probeStatusOutcome    `json:"probeStatuses"`
	PeriodicOutput map[string]periodicOutputOutcome `json:"periodicOutput"`

	CanceledAnnotation string `json:"canceledAnnotation"`
	PausedAnnotation   string `json:"pausedAnnotation"`

	// Present records, for keys the Secret only ever writes with omitempty semantics, whether the
	// key is present at all. Absence is itself an assertion the specs must be able to make; a zero
	// value in the corresponding field above is ambiguous between "absent" and "present but zero".
	Present outcomePresence `json:"present"`
}

type outcomePresence struct {
	AppliedChecksum    bool `json:"appliedChecksum"`
	FailedChecksum     bool `json:"failedChecksum"`
	LastApplyTime      bool `json:"lastApplyTime"`
	Checkpoint         bool `json:"checkpoint"`
	CanceledAnnotation bool `json:"canceledAnnotation"`
	PausedAnnotation   bool `json:"pausedAnnotation"`
}

// checkpointOutcome decodes secret.Data[planapi.PlanCheckpointKey]. Every field on
// planapi.PlanCheckpoint is tagged omitempty, so a plain unmarshal into that struct cannot
// distinguish "the key is absent from the checkpoint JSON" from "the key is present with its
// zero value" -- exactly the distinction a canceled plan's checkpoint depends on: cancellation
// never sets Paused, so a canceled plan's checkpoint JSON has no "paused" key at all, while a
// resumed-from-pause checkpoint has "paused":false explicitly. Present is decoded separately,
// from a raw key-presence map, to preserve that distinction. None of these fields is tagged
// omitempty here (unlike planapi.PlanCheckpoint): 0 is a legitimate, commonly asserted value
// for Completed (e.g. a plan paused before its first instruction), and omitempty would make the
// field disappear from this command's JSON output entirely rather than print 0.
type checkpointOutcome struct {
	Checksum              string `json:"checksum"`
	Completed             int    `json:"completedInstructions"`
	Total                 int    `json:"totalInstructions"`
	ResumeState           string `json:"resumeState"`
	Paused                bool   `json:"paused"`
	TerminationIncomplete bool   `json:"terminationIncomplete"`

	Present checkpointPresence `json:"present"`
}

type checkpointPresence struct {
	Paused      bool `json:"paused"`
	ResumeState bool `json:"resumeState"`
}

// probeStatusOutcome mirrors planapi.ProbeStatus, without its omitempty tags: Healthy=false and
// SuccessCount/FailureCount=0 are ordinary, commonly asserted values here, not "absent" signals.
type probeStatusOutcome struct {
	Healthy      bool `json:"healthy"`
	SuccessCount int  `json:"successCount"`
	FailureCount int  `json:"failureCount"`
}

// periodicOutputOutcome mirrors planapi.PeriodicInstructionOutput, except Stdout/Stderr are
// plain strings rather than []byte: Go's JSON encoder renders a []byte field as a base64
// string, so re-marshaling planapi.PeriodicInstructionOutput directly (as this command does,
// unlike the applyinator/Secret encoding this data started as) would silently turn instruction
// output into base64 in the JSON this command prints, the same conversion Output/FailedOutput
// already apply for one-time instruction output.
type periodicOutputOutcome struct {
	Name                  string `json:"name"`
	Stdout                string `json:"stdout"`
	Stderr                string `json:"stderr"`
	ExitCode              int    `json:"exitCode"`
	LastSuccessfulRunTime string `json:"lastSuccessfulRunTime"`
	Failures              int    `json:"failures"`
	LastFailedRunTime     string `json:"lastFailedRunTime"`
}

func outcomeCommand() *cli.Command {
	return &cli.Command{
		Name:  "outcome",
		Usage: "print the decoded plan Secret state as plain JSON",
		Action: func(cCtx *cli.Context) error {
			clientset, err := newClientset(cCtx)
			if err != nil {
				return err
			}
			ctx := context.Background()

			secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("getting plan secret %s/%s: %w", namespace, secretName, err)
			}

			result, err := decodeOutcome(secret)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}
}

func decodeOutcome(secret *corev1.Secret) (outcomeResult, error) {
	data := secret.Data
	annotations := secret.Annotations

	result := outcomeResult{
		ResourceVersion: secret.ResourceVersion,
		PlanState:       string(data[planapi.PlanStateKey]),
		PlanRevision:    string(data[planapi.PlanRevisionKey]),
	}

	if v, ok := data[k8splan.AppliedChecksumKey]; ok && len(v) > 0 {
		result.AppliedChecksum = string(v)
		result.Present.AppliedChecksum = true
	}
	if v, ok := data[k8splan.FailedChecksumKey]; ok && len(v) > 0 {
		result.FailedChecksum = string(v)
		result.Present.FailedChecksum = true
	}
	if v, ok := data[k8splan.LastApplyTimeKey]; ok && len(v) > 0 {
		result.LastApplyTime = string(v)
		result.Present.LastApplyTime = true
	}

	if v := data[k8splan.FailureCountKey]; len(v) > 0 {
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return outcomeResult{}, fmt.Errorf("parsing %s: %w", k8splan.FailureCountKey, err)
		}
		result.FailureCount = n
	}
	if v := data[k8splan.SuccessCountKey]; len(v) > 0 {
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return outcomeResult{}, fmt.Errorf("parsing %s: %w", k8splan.SuccessCountKey, err)
		}
		result.SuccessCount = n
	}

	checkpoint, err := decodeCheckpoint(data[planapi.PlanCheckpointKey])
	if err != nil {
		return outcomeResult{}, fmt.Errorf("parsing %s: %w", planapi.PlanCheckpointKey, err)
	}
	if checkpoint != nil {
		result.Checkpoint = checkpoint
		result.Present.Checkpoint = true
	}

	// Reuses planapi.ReadAppliedOutput directly, per the design's "using planapi constants
	// directly ... prevents the test from drifting by hardcoding key names": this is the one
	// helper planapi exports for this purpose. There is no equivalent exported helper for
	// failed-output, so that side is decoded manually below using the same gzip+JSON envelope.
	appliedOutput, err := planapi.ReadAppliedOutput(secret)
	if err != nil {
		return outcomeResult{}, fmt.Errorf("decoding applied output: %w", err)
	}
	result.Output = bytesMapToStringMap(appliedOutput)

	failedOutput, err := decodeGzipOutputMap(data[k8splan.FailedOutputKey])
	if err != nil {
		return outcomeResult{}, fmt.Errorf("decoding %s: %w", k8splan.FailedOutputKey, err)
	}
	result.FailedOutput = failedOutput

	probeStatuses, err := decodeProbeStatuses(data[k8splan.ProbeStatusesKey])
	if err != nil {
		return outcomeResult{}, fmt.Errorf("decoding %s: %w", k8splan.ProbeStatusesKey, err)
	}
	result.ProbeStatuses = probeStatuses

	periodicOutput, err := planapi.ReadAppliedPeriodicOutput(secret)
	if err != nil {
		return outcomeResult{}, fmt.Errorf("decoding applied periodic output: %w", err)
	}
	result.PeriodicOutput = periodicOutputToOutcome(periodicOutput)

	if v, ok := annotations[planapi.PlanCanceledAnnotation]; ok {
		result.CanceledAnnotation = v
		result.Present.CanceledAnnotation = true
	}
	if v, ok := annotations[planapi.PlanPausedAnnotation]; ok {
		result.PausedAnnotation = v
		result.Present.PausedAnnotation = true
	}

	return result, nil
}

// decodeCheckpoint parses raw checkpoint JSON twice: once into the typed struct for values, and
// once into a raw key-presence map so the caller can tell an absent key from a present zero value.
func decodeCheckpoint(raw []byte) (*checkpointOutcome, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var checkpoint checkpointOutcome
	if err := json.Unmarshal(raw, &checkpoint); err != nil {
		return nil, err
	}

	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawFields); err != nil {
		return nil, err
	}
	if _, ok := rawFields["paused"]; ok {
		checkpoint.Present.Paused = true
	}
	if _, ok := rawFields["resumeState"]; ok {
		checkpoint.Present.ResumeState = true
	}

	return &checkpoint, nil
}

// decodeProbeStatuses parses secret.Data[k8splan.ProbeStatusesKey], which the agent writes as
// plain JSON (json.Marshal(probeStatuses) directly into the Secret), unlike the gzip-wrapped
// one-time and periodic instruction output.
func decodeProbeStatuses(raw []byte) (map[string]probeStatusOutcome, error) {
	if len(raw) == 0 {
		return map[string]probeStatusOutcome{}, nil
	}
	var out map[string]probeStatusOutcome
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func bytesMapToStringMap(in map[string][]byte) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = string(v)
	}
	return out
}

// periodicOutputToOutcome converts planapi.ReadAppliedPeriodicOutput's result into
// periodicOutputOutcome, converting Stdout/Stderr from []byte to string in the process. See
// periodicOutputOutcome's doc comment for why that conversion is required.
func periodicOutputToOutcome(in map[string]planapi.PeriodicInstructionOutput) map[string]periodicOutputOutcome {
	out := make(map[string]periodicOutputOutcome, len(in))
	for k, v := range in {
		out[k] = periodicOutputOutcome{
			Name:                  v.Name,
			Stdout:                string(v.Stdout),
			Stderr:                string(v.Stderr),
			ExitCode:              v.ExitCode,
			LastSuccessfulRunTime: v.LastSuccessfulRunTime,
			Failures:              v.Failures,
			LastFailedRunTime:     v.LastFailedRunTime,
		}
	}
	return out
}

// decodeGzipOutputMap unwraps the gzip envelope and the per-value base64 that encoding/json
// applies to Go []byte, matching how the applyinator package encodes one-time instruction output.
// Used for failed-output, for which planapi exports no ReadFailedOutput-style helper.
func decodeGzipOutputMap(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var out map[string][]byte
	if err := json.Unmarshal(decoded, &out); err != nil {
		return nil, err
	}

	return bytesMapToStringMap(out), nil
}
