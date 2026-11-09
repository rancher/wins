# pause_test.ps1 exercises plan pause/resume via the plan.cattle.io/paused annotation. Unlike
# cancellation, pause does not interrupt a running instruction: it holds the plan at the next
# instruction boundary and resumes from the recorded checkpoint once the annotation is cleared.

Describe "Pause and resume" {
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

    It "P1: pausing a pending plan executes nothing until the annotation is removed" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA"
        )
        $planFile = Join-Path $scratch "p1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" -Paused "true" | Out-Null

        {
            (Get-PlanOutcome).planState -eq "paused"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $false

        $outcome = Get-PlanOutcome
        $outcome.planState | Should -Be "paused"
        $outcome.present.appliedChecksum | Should -Be $false
        $outcome.checkpoint.completedInstructions | Should -Be 0
        $outcome.checkpoint.present.resumeState | Should -Be $true
        $outcome.checkpoint.resumeState | Should -Be "pending"

        Invoke-PlanAnnotate -Paused ""

        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $true
        (Get-PlanOutcome).present.appliedChecksum | Should -Be $true
    }

    It "P2: pausing mid-execution completes the current instruction, holds before the next, and resumes from the checkpoint when the annotation is removed" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"
        $markerB = Join-Path $scratch "marker-b"

        $plan = New-PlanSpec
        # Instruction a runs long enough for a separate planctl process invocation to annotate paused and
        # for the agent's 2s interrupt poll interval to notice it before a finishes.
        # Pause never interrupts a running instruction, it only holds before the next one starts,
        # so a itself always finishes and writes marker-a regardless of when the annotation lands.
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA && ping -n 20 127.0.0.1 > nul"
        )
        Add-PlanInstruction -Plan $plan -Name "b" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerB"
        )
        $planFile = Join-Path $scratch "p2-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        Wait-Ready -Path $markerA -Timeout 30 -Throw

        Invoke-PlanAnnotate -Paused "true"

        {
            (Get-PlanOutcome).planState -eq "paused"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $true
        Test-Path -Path $markerB | Should -Be $false

        $outcome = Get-PlanOutcome
        $outcome.planState | Should -Be "paused"
        $outcome.present.appliedChecksum | Should -Be $false
        $outcome.checkpoint.completedInstructions | Should -Be 1
        $outcome.checkpoint.totalInstructions | Should -Be 2
        $outcome.checkpoint.present.resumeState | Should -Be $true
        $outcome.checkpoint.resumeState | Should -Be "in-progress"

        Invoke-PlanAnnotate -Paused ""

        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 30 -Throw

        # Resuming continues from the checkpoint rather than restarting: a is not re-run, so its
        # marker's content is unchanged, and b now runs for the first time.
        Test-Path -Path $markerB | Should -Be $true
        (Get-PlanOutcome).present.appliedChecksum | Should -Be $true
    }

    It "P3: pausing mid-execution with periodic instructions and probes holds periodic execution until resumed, while probes keep updating" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"
        $probePort = 18235

        $probeJob = Start-ProbeTargetServer -Port $probePort -StatusCode 500
        try {
            $plan = New-PlanSpec
            # A single one-time instruction: pause never interrupts it, so once it finishes the
            # one-time loop ends naturally (no further one-time instruction to check
            # interruption before), and it is periodic's own loop-top interruption check (or the
            # final check at the end of Apply) that holds the periodic instruction back.
            Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
                "/c", "echo started > $markerA && ping -n 20 127.0.0.1 > nul"
            )
            Add-PlanPeriodicInstruction -Plan $plan -Name "p" -Command "cmd.exe" -Args @("/c", "exit 0") -PeriodSeconds 5
            # FailureThreshold is deliberately large: applyProbeResult (status.go) stops
            # incrementing FailureCount once it reaches the threshold, so a small threshold like
            # the default 1 would cap the count on the very first evaluation and make it
            # impossible to observe further growth across the test's later wait.
            Add-PlanProbe -Plan $plan -Name "probe1" -Url "http://localhost:$probePort/" -FailureThreshold 100
            $planFile = Join-Path $scratch "p3-plan.json"
            Write-PlanFile -Plan $plan -Path $planFile

            Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

            Wait-Ready -Path $markerA -Timeout 30 -Throw

            Invoke-PlanAnnotate -Paused "true"

            {
                (Get-PlanOutcome).planState -eq "paused"
            } | Judge -Timeout 30 -Throw

            Test-Path -Path $markerA | Should -Be $true

            $outcomeAfterPause = Get-PlanOutcome
            $outcomeAfterPause.planState | Should -Be "paused"
            $outcomeAfterPause.periodicOutput.PSObject.Properties.Name | Should -Not -Contain "p"
            $outcomeAfterPause.probeStatuses.probe1.healthy | Should -Be $false
            $failureCountAfterPause = $outcomeAfterPause.probeStatuses.probe1.failureCount

            # A paused plan's reconcile keeps merging probe statuses on the write-once interrupt
            # path even after the pause is already recorded (the design's stated reason: keeping
            # health data current for Rancher's MachineHealthCheck), so the failure count keeps
            # climbing while the periodic instruction stays held back.
            Start-Sleep -Seconds 15

            $outcomeAfterWait = Get-PlanOutcome
            $outcomeAfterWait.probeStatuses.probe1.failureCount | Should -BeGreaterThan $failureCountAfterPause
            $outcomeAfterWait.periodicOutput.PSObject.Properties.Name | Should -Not -Contain "p"
            $outcomeAfterWait.planState | Should -Be "paused"

            Invoke-PlanAnnotate -Paused ""

            {
                (Get-PlanOutcome).planState -eq "succeeded"
            } | Judge -Timeout 30 -Throw

            # Resuming forces the periodic instruction to run at least once regardless of its
            # own period, the same "ranOneTime" forcing that lets a freshly (re)applied plan's
            # periodic instructions run immediately rather than waiting out their full period.
            (Get-PlanOutcome).periodicOutput.PSObject.Properties.Name | Should -Contain "p"
        }
        finally {
            Stop-ProbeTargetServer -Port $probePort -Job $probeJob
        }
    }

    It "P4: a pause checkpoint survives an agent restart, without re-running what already completed" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"
        $counterA = Join-Path $scratch "counter-a.txt"
        $markerB = Join-Path $scratch "marker-b"

        $plan = New-PlanSpec
        # a appends to a counter file, rather than just echoing a fixed marker, so a re-run
        # after the restart (a bug this spec exists to catch) would be visible as a second line.
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA && echo run >> $counterA && ping -n 20 127.0.0.1 > nul"
        )
        Add-PlanInstruction -Plan $plan -Name "b" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerB"
        )
        $planFile = Join-Path $scratch "p4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        Wait-Ready -Path $markerA -Timeout 30 -Throw

        Invoke-PlanAnnotate -Paused "true"

        {
            (Get-PlanOutcome).planState -eq "paused"
        } | Judge -Timeout 30 -Throw

        (Get-PlanOutcome).checkpoint.completedInstructions | Should -Be 1

        Restart-WinsService
        Assert-WinsServiceRunning

        # The checkpoint is keyed only by plan checksum, in the Secret, not anything tied to a
        # single agent process instance, so the restarted agent must recognize the existing
        # suspension from the Secret alone and not rewrite or restart it.
        Start-Sleep -Seconds 10

        $outcomeAfterRestart = Get-PlanOutcome
        $outcomeAfterRestart.planState | Should -Be "paused"
        $outcomeAfterRestart.checkpoint.completedInstructions | Should -Be 1
        Test-Path -Path $markerB | Should -Be $false
        (Get-Content -Path $counterA | Measure-Object).Count | Should -Be 1

        Invoke-PlanAnnotate -Paused ""

        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 30 -Throw

        # Resuming after the restart still continues from the checkpoint rather than
        # restarting: a is not re-run (the counter file still has exactly one line), and b now
        # runs for the first time.
        Test-Path -Path $markerB | Should -Be $true
        (Get-Content -Path $counterA | Measure-Object).Count | Should -Be 1
    }

    It "P5: an invalid pause annotation value is rejected and writes nothing at all" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA"
        )
        $planFile = Join-Path $scratch "p5-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        # "True" (capital T) is not a valid annotation value; only "true" and "false" are. Set
        # atomically with apply-plan so the agent's very first reconcile of this plan already
        # observes the invalid value, for the same reason C4/P1 set their annotations
        # atomically: a separate, later Invoke-PlanAnnotate call could race with the agent
        # already having executed the instruction.
        Invoke-PlanApply -PlanFile $planFile -State "pending" -Paused "True" | Out-Null

        $resourceVersionAtApply = (Get-PlanOutcome).resourceVersion

        # An invalid annotation value makes readInterrupt return an error, which
        # checkAndRecordInterrupt propagates without writing anything at all: not even a
        # plan-state change. Confirmed here by resourceVersion staying completely unchanged,
        # not just by plan-state staying "pending" (a weaker check that a spurious no-op write
        # would still pass).
        Start-Sleep -Seconds 15

        $outcome = Get-PlanOutcome
        $outcome.resourceVersion | Should -Be $resourceVersionAtApply
        $outcome.planState | Should -Be "pending"
        Test-Path -Path $markerA | Should -Be $false

        Invoke-PlanAnnotate -Paused "true"

        {
            (Get-PlanOutcome).planState -eq "paused"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $false
    }

    It "P6: setting the pause annotation to false resumes a paused plan exactly as removing it does" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA"
        )
        $planFile = Join-Path $scratch "p6-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" -Paused "true" | Out-Null

        {
            (Get-PlanOutcome).planState -eq "paused"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $false

        Invoke-PlanAnnotate -Paused "false"

        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerA | Should -Be $true
    }
}
