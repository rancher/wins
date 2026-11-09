# periodic_probe_test.ps1 exercises periodic instructions and probes on their own, standalone
# from cancellation/pause (see cancellation_test.ps1's C5 and pause_test.ps1's P3 for their
# interaction with cancellation/pause).

Describe "Periodic instructions and probes" {
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

    It "PP1: runs a periodic instruction immediately on first apply, regardless of its period" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
        # periodSeconds is deliberately long: the first run must happen immediately because the
        # apply is forced (ranOneTime=true), not because the period elapsed.
        Add-PlanPeriodicInstruction -Plan $plan -Name "p1" -Command "cmd.exe" -Args @("/c", "echo periodic-hello") -PeriodSeconds 600
        $planFile = Join-Path (Get-PlanScratchDirectory) "pp1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $periodic = (Get-PlanOutcome).periodicOutput.p1
        $periodic.stdout | Should -Match "periodic-hello"
        $periodic.exitCode | Should -Be 0
        $periodic.lastSuccessfulRunTime | Should -Not -BeNullOrEmpty
    }

    It "PP2: re-runs a periodic instruction after its period elapses" {
        $counterFile = Join-Path (Get-PlanScratchDirectory) "pp2-counter.txt"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
        Add-PlanPeriodicInstruction -Plan $plan -Name "p1" -Command "cmd.exe" -Args @("/c", "echo run >> $counterFile") -PeriodSeconds 5
        $planFile = Join-Path (Get-PlanScratchDirectory) "pp2-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $firstRunTime = (Get-PlanOutcome).periodicOutput.p1.lastSuccessfulRunTime

        # Longer than the 5s period plus the agent's own ~5s reconcile cadence, so at least one
        # more due-check and run has had a chance to happen.
        Start-Sleep -Seconds 15

        $secondRunTime = (Get-PlanOutcome).periodicOutput.p1.lastSuccessfulRunTime
        $secondRunTime | Should -Not -Be $firstRunTime
        (Get-Content -Path $counterFile | Measure-Object).Count | Should -BeGreaterThan 1
    }

    It "PP3: omits stderr output when saveStderrOutput is false, but still records exitCode and Failures for a failing instruction" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
        Add-PlanPeriodicInstruction -Plan $plan -Name "p1" -Command "cmd.exe" -Args @("/c", "echo err-output 1>&2 && exit 1") -PeriodSeconds 600
        $planFile = Join-Path (Get-PlanScratchDirectory) "pp3-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        # A failing periodic instruction does not fail the plan: plan-state is driven only by
        # the one-time instructions.
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $periodic = (Get-PlanOutcome).periodicOutput.p1
        $periodic.exitCode | Should -Be 1
        $periodic.stderr | Should -BeNullOrEmpty
        $periodic.failures | Should -BeGreaterThan 0
    }

    It "PP4: injects plan-supplied environment variables into a periodic instruction" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
        Add-PlanPeriodicInstruction -Plan $plan -Name "p1" -Command "cmd.exe" -Args @("/c", "echo %WINS_E2E_PERIODIC_FOO%") `
            -Env @("WINS_E2E_PERIODIC_FOO=periodic-bar") -PeriodSeconds 600
        $planFile = Join-Path (Get-PlanScratchDirectory) "pp4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).periodicOutput.p1.stdout | Should -Match "periodic-bar"
    }

    It "PP5: a probe becomes healthy after successThreshold consecutive successes" {
        $probePort = 18236
        $probeJob = Start-ProbeTargetServer -Port $probePort -StatusCode 200
        try {
            $plan = New-PlanSpec
            Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
            Add-PlanProbe -Plan $plan -Name "probe1" -Url "http://localhost:$probePort/" -SuccessThreshold 1 -FailureThreshold 1
            $planFile = Join-Path (Get-PlanScratchDirectory) "pp5-plan.json"
            Write-PlanFile -Plan $plan -Path $planFile

            Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
            {
                (Get-PlanOutcome).planState -eq "succeeded"
            } | Judge -Timeout 60 -Throw

            {
                (Get-PlanOutcome).probeStatuses.probe1.healthy -eq $true
            } | Judge -Timeout 30 -Throw

            $probeStatus = (Get-PlanOutcome).probeStatuses.probe1
            $probeStatus.healthy | Should -Be $true
            $probeStatus.successCount | Should -BeGreaterThan 0
            $probeStatus.failureCount | Should -Be 0
        }
        finally {
            Stop-ProbeTargetServer -Port $probePort -Job $probeJob
        }
    }

    It "PP6: a probe becomes unhealthy after failureThreshold consecutive failures" {
        $probePort = 18237
        $probeJob = Start-ProbeTargetServer -Port $probePort -StatusCode 500
        try {
            $plan = New-PlanSpec
            Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
            Add-PlanProbe -Plan $plan -Name "probe1" -Url "http://localhost:$probePort/" -SuccessThreshold 1 -FailureThreshold 1
            $planFile = Join-Path (Get-PlanScratchDirectory) "pp6-plan.json"
            Write-PlanFile -Plan $plan -Path $planFile

            Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
            {
                (Get-PlanOutcome).planState -eq "succeeded"
            } | Judge -Timeout 60 -Throw

            {
                (Get-PlanOutcome).probeStatuses.probe1.failureCount -ge 1
            } | Judge -Timeout 30 -Throw

            $probeStatus = (Get-PlanOutcome).probeStatuses.probe1
            $probeStatus.healthy | Should -Be $false
            $probeStatus.failureCount | Should -BeGreaterThan 0
            $probeStatus.successCount | Should -Be 0
        }
        finally {
            Stop-ProbeTargetServer -Port $probePort -Job $probeJob
        }
    }

    It "PP7: multiple probes are tracked independently" {
        $healthyPort = 18238
        $failingPort = 18239
        $healthyJob = Start-ProbeTargetServer -Port $healthyPort -StatusCode 200
        $failingJob = Start-ProbeTargetServer -Port $failingPort -StatusCode 500
        try {
            $plan = New-PlanSpec
            Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
            Add-PlanProbe -Plan $plan -Name "healthy-probe" -Url "http://localhost:$healthyPort/" -SuccessThreshold 1 -FailureThreshold 1
            Add-PlanProbe -Plan $plan -Name "failing-probe" -Url "http://localhost:$failingPort/" -SuccessThreshold 1 -FailureThreshold 1
            $planFile = Join-Path (Get-PlanScratchDirectory) "pp7-plan.json"
            Write-PlanFile -Plan $plan -Path $planFile

            Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
            {
                (Get-PlanOutcome).planState -eq "succeeded"
            } | Judge -Timeout 60 -Throw

            {
                $outcome = Get-PlanOutcome
                $outcome.probeStatuses."healthy-probe".healthy -eq $true -and $outcome.probeStatuses."failing-probe".failureCount -ge 1
            } | Judge -Timeout 30 -Throw

            $outcome = Get-PlanOutcome
            $outcome.probeStatuses."healthy-probe".healthy | Should -Be $true
            $outcome.probeStatuses."failing-probe".healthy | Should -Be $false
        }
        finally {
            Stop-ProbeTargetServer -Port $healthyPort -Job $healthyJob
            Stop-ProbeTargetServer -Port $failingPort -Job $failingJob
        }
    }
}
