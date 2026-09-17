# Windows plan e2e suite

This suite exercises `rancher-system-agent`'s plan-execution behavior end to end on a regular
Windows machine, running as the `rancher-wins` Windows Service and driven by a Kubernetes plan
Secret on an external Kubernetes cluster.

It covers six behavior areas: Job-Object-based cancellation and process-tree kill, pause and
resume from a checkpoint, failure/success bookkeeping on the plan Secret, instruction execution
(stdout/stderr capture, environment injection, exit codes), periodic instructions and probes, and
plan file operations (write, delete, directory creation).

It does not run in CI: it requires an external Kubernetes cluster, and installs `wins` on the machine.

## Prerequisites

- A regular Windows machine (no RKE2, no Rancher) with an elevated (Administrator) PowerShell
  session and a Go toolchain on `PATH`.
- Pester 5.0.0 or newer. Windows ships an older Pester (3.4.0) by default, which
  `plan_suite_test.ps1` cannot use; install a current one with:
  `Install-Module -Name Pester -MinimumVersion 5.0.0 -Force -SkipPublisherCheck -Scope CurrentUser`.
- This repo checked out on that machine: `bootstrap.ps1` builds `wins.exe` and `planctl.exe`
  locally with `go build` from `tests/plan/bootstrap.ps1`'s own checkout, so there is nothing to
  stage or copy separately.
- A kubeconfig for an external Kubernetes cluster, reachable from the machine. This cluster only
  hosts the plan Secret; it does not need to be the cluster the Windows machine is joined to,
  because it isn't joined to one.

## Running the suite

The single entry point is `run_suite.ps1`, from an elevated PowerShell session in `tests/plan/`:

```powershell
.\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
```

Every run rebuilds `wins.exe` and `planctl.exe` from the current checkout (via `bootstrap.ps1`,
described below) and then runs the 39 Pester specs across six files (`cancellation_test.ps1`,
`pause_test.ps1`, `failure_handling_test.ps1`, `instruction_execution_test.ps1`,
`periodic_probe_test.ps1`, `file_operations_test.ps1`). Expected runtime is 8 to 12 minutes,
dominated by the cancellation and pause specs.

- **If every test passes**, `run_suite.ps1` stops and unregisters `rancher-wins`, leaving the
  machine clean.
- **If any test fails**, it leaves `rancher-wins` installed and running so the failure can be
  diagnosed against a live service; re-running `run_suite.ps1` (or `bootstrap.ps1` on its own)
  rebuilds and redeploys over that installation.

To re-run only specific tests (for example, ones that just failed), pass `-TestName` with one or
more wildcard patterns matched against each test's full name (`<Describe name>.<It name>`); this
also skips the uninstall step regardless of outcome, since a partial run says nothing about the
suite as a whole:

```powershell
.\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig -TestName "*C1*", "*C4*"
```

### Running the steps individually

`run_suite.ps1` is `bootstrap.ps1` followed by `plan_suite_test.ps1`, with the uninstall step
added on top; each can also be run on its own, for example to iterate on a single spec without
rebuilding or uninstalling every time:

1. `bootstrap.ps1` builds `wins.exe` and `planctl.exe` locally with `go build`; creates the plan
   Secret, ServiceAccount, and RBAC on the external cluster via `planctl.exe`; writes the wins
   config directly (with `debug: true`, since `[applyinator]` log lines are `logrus.Debugf`) and
   registers/starts `rancher-wins`, without going through `install.ps1` since there is no Rancher
   endpoint to simulate; and runs a self-check plan to confirm the installation actually applies
   plans before any spec runs.

   It is safe to re-run: if `rancher-wins` is already installed, `bootstrap.ps1` stops it,
   deploys the freshly built binary over the existing installation, and restarts it.

   ```powershell
   .\bootstrap.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
   ```

2. `plan_suite_test.ps1` runs the Pester suite against whatever `rancher-wins` installation is
   already running, without rebuilding or redeploying anything:

   ```powershell
   .\plan_suite_test.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig -TestName "*C1*", "*C4*"
   ```

## Notes

- Unless `run_suite.ps1` uninstalled it (every test passed, no `-TestName` filter), the machine is
  permanently mutated: `wins` stays installed at `c:/usr/local/bin`, and `debug: true` stays set
  in `C:/etc/rancher/wins/config`. Use a dedicated test machine.
- Every plan's side effects are confined to `C:\wins-plan-e2e`.
- Local plan mode and OCI-image-based instructions are out of scope.
