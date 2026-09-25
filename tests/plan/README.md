# Windows Plan E2E Suite

This suite tests `rancher-system-agent`'s plan execution end-to-end on Windows. 
It runs as the `rancher-wins` Windows service and applies plans driven by a Kubernetes Secret on an external cluster.

**Covered areas:**
- Job-Object-based cancellation and process-tree termination
- Pause and resume from checkpoints
- Plan Secret failure/success bookkeeping
- Instruction execution (stdout/stderr capture, environment variables, exit codes)
- Periodic instructions and probes
- Plan file operations (write, delete, create directories)

> **Note:** This suite installs `wins` on the host and requires an external Kubernetes cluster. It does not run in CI.

## Prerequisites

Run all commands from an elevated (Administrator) PowerShell session at the **repository root**.

- **Windows machine**: Dedicated Windows host (no RKE2, no Rancher) with a Go toolchain on `PATH`.
- **Pester 5.0 or newer+**: Windows ships Pester 3.4.0 by default. Install or upgrade via:
  ```powershell
  Install-Module -Name Pester -MinimumVersion 5.0.0 -Force -SkipPublisherCheck -Scope CurrentUser
  ```
- **Kubeconfig**: A kubeconfig for an external Kubernetes cluster (used only to host the plan Secret).
- **Repo checkout**: Binaries (`wins.exe` and `planctl.exe`) are built locally from the current checkout.

## Running the Suite

From the repository root, run `run_suite.ps1`:

```powershell
.\tests\plan\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
```

The script automatically:
1. Builds `wins.exe` and `planctl.exe` from the current checkout.
2. Deploys and configures the `rancher-wins` service.
3. Executes the 39 Pester specs across all 6 test files.
4. Cleans up and uninstalls `rancher-wins` on success.

### Test Outcomes

- **All tests pass**: `rancher-wins` is stopped and uninstalled, leaving the machine clean.
- **Any test fails**: `rancher-wins` remains installed and running for diagnosis. Re-running `run_suite.ps1` rebuilds and redeploys over the existing installation.

### Running Specific Tests

Pass `-TestName` with wildcard patterns matching `<Describe>.<It>` to run a subset of tests. This also skips uninstallation:

```powershell
.\tests\plan\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig -TestName "*C1*", "*C4*"
```

## Running Steps Individually

For faster iteration during development, you can run the individual steps directly from the repository root:

### 1. Bootstrap the environment
Builds binaries, sets up cluster RBAC/Secret, writes config with `debug: true`, registers and starts `rancher-wins`, and runs a self-check plan:

```powershell
.\tests\plan\bootstrap.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
```

*Safe to re-run: redeploys freshly built binaries over the existing installation.*

### 2. Run tests against the active installation
Runs the Pester suite against the currently running `rancher-wins` service without rebuilding or redeploying:

```powershell
.\tests\plan\plan_suite_test.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig -TestName "*C1*", "*C4*"
```

## Notes

- **Machine mutation**: Unless cleanly uninstalled by a passing full suite run, `wins.exe` remains in `C:\usr\local\bin` and config in `C:\etc\rancher\wins\config`. Use a dedicated test machine.
- **Side effects**: All plan side effects are confined to `C:\wins-plan-e2e`.
- **Scope**: Local plan mode and OCI image-based instructions are out of scope.
