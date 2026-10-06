# failure_handling_test.ps1 exercises failure/success bookkeeping on the plan Secret:
# failure-count and success-count tracking, failure-state reset on a subsequent successful
# apply, and the monitoring-only invariant for an already-failed terminal plan.
#
# Unlike system-agent's Linux e2e suite, every plan here is created with plan-state set (the
# "plan-state flow"), matching how planctl always operates: decidePlanStateAction
# (pkg/k8splan/plan_decision.go) never consults max-failures or a retry cooldown at all, that
# machinery belongs exclusively to the legacy "checksum flow" for plan Secrets with no
# plan-state key. In the plan-state flow, a failed plan only gets retried when the orchestrator
# explicitly resets plan-state back to "pending"; there is no automatic backoff-retry or
# max-failures threshold to test here.

Describe "Failure and success bookkeeping" {
    BeforeAll {
        Import-Module -Name "$PSScriptRoot\planutils.psm1" -WarningAction Ignore -Force
        Set-PlanKubeconfig -Path $env:WINS_PLAN_E2E_KUBECONFIG
        Assert-WinsServiceRunning
        $script:T0 = Get-Date
    }

    BeforeEach {
        Reset-Plan
        Reset-PlanScratchDirectory
    }

    AfterEach {
        # Pester does not expose a reliable, version-independent way to check the current test's
        # pass/fail state from inside AfterEach, so this dumps the outcome and Event Log
        # unconditionally rather than depending on an undocumented internal.
        Show-PlanFailureDiagnostics -Since $script:T0
        Clear-PlanScratchProcesses
        Clear-PlanScratchDirectory
    }

    AfterAll {
        Reset-Plan
    }

    It "FH1: failure count increments each time the orchestrator re-applies a permanently-failing plan" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "will-fail" -Command "cmd.exe" -Args @("/c", "exit 1") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "fh1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed"
        } | Judge -Timeout 60 -Throw

        $outcome = Get-PlanOutcome
        $outcome.failureCount | Should -Be 1
        $outcome.present.failedChecksum | Should -Be $true
        $outcome.failedOutput.PSObject.Properties.Name | Should -Contain "will-fail"

        # A failed plan is terminal and never retries on its own (decidePlanStateAction has no
        # default case that keeps applying); only re-applying with state: pending forces
        # another attempt, exactly as an orchestrator resetting plan-state would.
        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed" -and (Get-PlanOutcome).failureCount -eq 2
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).failureCount | Should -Be 2
    }

    It "FH2: failure state resets when a successful plan replaces a failing one" {
        $scratch = Get-PlanScratchDirectory

        $failingPlan = New-PlanSpec
        Add-PlanInstruction -Plan $failingPlan -Name "will-fail" -Command "cmd.exe" -Args @("/c", "exit 1")
        $failingPlanFile = Join-Path $scratch "fh2-failing-plan.json"
        Write-PlanFile -Plan $failingPlan -Path $failingPlanFile

        Invoke-PlanApply -PlanFile $failingPlanFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).failureCount | Should -BeGreaterThan 0
        (Get-PlanOutcome).present.failedChecksum | Should -Be $true

        $passingPlan = New-PlanSpec
        Add-PlanInstruction -Plan $passingPlan -Name "will-pass" -Command "cmd.exe" -Args @("/c", "exit 0")
        $passingPlanFile = Join-Path $scratch "fh2-passing-plan.json"
        Write-PlanFile -Plan $passingPlan -Path $passingPlanFile

        Invoke-PlanApply -PlanFile $passingPlanFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $outcome = Get-PlanOutcome
        $outcome.failureCount | Should -Be 0
        $outcome.present.failedChecksum | Should -Be $false
    }

    It "FH3: success count is tracked on a successful apply" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "will-pass" -Command "cmd.exe" -Args @("/c", "exit 0")
        $planFile = Join-Path (Get-PlanScratchDirectory) "fh3-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).successCount | Should -BeGreaterThan 0
    }

    It "FH4: an already-failed plan stays permanently monitoring-only, and cancelling it has no effect at all" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "will-fail" -Command "cmd.exe" -Args @("/c", "exit 1")
        Add-PlanPeriodicInstruction -Plan $plan -Name "p" -Command "cmd.exe" -Args @("/c", "exit 0") -PeriodSeconds 5
        $planFile = Join-Path (Get-PlanScratchDirectory) "fh4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed"
        } | Judge -Timeout 60 -Throw

        # A failing one-time instruction does not stop the periodic instruction from running in
        # that same apply: only an interruption (cancel/pause), not a plain failure, skips it.
        {
            (Get-PlanOutcome).periodicOutput.p.lastSuccessfulRunTime -ne $null
        } | Judge -Timeout 30 -Throw

        $runTimeBeforeCancel = (Get-PlanOutcome).periodicOutput.p.lastSuccessfulRunTime

        # handleCancellation's guard treats any already-terminal, non-succeeded plan-state
        # (failed, same as canceled) as already inert: cancelling a failed plan does not
        # transition it to "canceled" at all, unlike cancelling a succeeded plan (see C6).
        Invoke-PlanAnnotate -Canceled "true"

        # The annotate call itself is a Secret write (by planctl, not the agent), so the
        # resourceVersion baseline for "did the agent write anything else" must be captured
        # after that call returns, not before it.
        $resourceVersionAfterCancelSet = (Get-PlanOutcome).resourceVersion
        Start-Sleep -Seconds 15

        $outcomeWhileCanceledAnnotationSet = Get-PlanOutcome
        $outcomeWhileCanceledAnnotationSet.planState | Should -Be "failed"
        $outcomeWhileCanceledAnnotationSet.periodicOutput.p.lastSuccessfulRunTime | Should -Be $runTimeBeforeCancel
        # A no-op reconcile writes nothing at all: resourceVersion only advances on an actual
        # write, so its being unchanged is a stronger proof than any individual field matching.
        $outcomeWhileCanceledAnnotationSet.resourceVersion | Should -Be $resourceVersionAfterCancelSet

        Invoke-PlanAnnotate -Canceled ""

        $resourceVersionAfterClear = (Get-PlanOutcome).resourceVersion
        Start-Sleep -Seconds 15

        $outcomeAfterClearingAnnotation = Get-PlanOutcome
        $outcomeAfterClearingAnnotation.planState | Should -Be "failed"
        $outcomeAfterClearingAnnotation.periodicOutput.p.lastSuccessfulRunTime | Should -Be $runTimeBeforeCancel
        $outcomeAfterClearingAnnotation.resourceVersion | Should -Be $resourceVersionAfterClear
    }
}
