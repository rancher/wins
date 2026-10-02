# cancellation_test.ps1 exercises Job-Object-based process-tree cancellation.

Describe "Cancellation and process-tree kill" {
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

    It "C1: terminates the entire process tree, not only the direct child" {
        $scratch = Get-PlanScratchDirectory
        $heartbeat = Join-Path $scratch "heartbeat.log"
        $marker = Join-Path $scratch "ready.marker"

        # The heartbeat writer is its own script file, invoked with -File and plain arguments
        # (no embedded -Command string), so no argument ever needs nested quoting: nesting a
        # quoted -Command string inside -ArgumentList, inside an already-quoted outer script,
        # is fragile and was observed to silently fail to start the grandchild.
        $writerScript = Join-Path $scratch "heartbeat-writer.ps1"
        $writerContent = @"
param(
    [string]`$HeartbeatPath
)
for (`$i = 0; `$i -lt 1500; `$i++) {
    Add-Content -Path `$HeartbeatPath -Value (Get-Date).Ticks
    Start-Sleep -Milliseconds 200
}
"@
        Set-Content -Path $writerScript -Value $writerContent

        # The grandchild is detached (Start-Process from within this script), appends to
        # heartbeat.log every 200ms, writes ready.marker once running, then sleeps 300s.
        $script = @"
`$heartbeat = '$heartbeat'
`$marker = '$marker'
Start-Process -WindowStyle Hidden -FilePath 'powershell.exe' -ArgumentList @('-NoProfile', '-File', '$writerScript', '-HeartbeatPath', `$heartbeat)
Start-Sleep -Seconds 1
New-Item -Path `$marker -ItemType File -Force | Out-Null
Start-Sleep -Seconds 300
"@
        $scriptPath = Join-Path $scratch "c1.ps1"
        Set-Content -Path $scriptPath -Value $script

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "c1" -Command "powershell.exe" -Args @("-NoProfile", "-File", $scriptPath)
        $planFile = Join-Path $scratch "c1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        Wait-Ready -Path $marker -Timeout 30 -Throw

        $firstCount = (Get-Content -Path $heartbeat -ErrorAction SilentlyContinue | Measure-Object).Count
        Start-Sleep -Seconds 2
        $secondCount = (Get-Content -Path $heartbeat -ErrorAction SilentlyContinue | Measure-Object).Count
        $secondCount | Should -BeGreaterThan $firstCount

        Invoke-PlanAnnotate -Canceled "true"

        Start-Sleep -Seconds 10
        $countAfterCancel = (Get-Content -Path $heartbeat -ErrorAction SilentlyContinue | Measure-Object).Count
        Start-Sleep -Seconds 8
        $countStillAfterCancel = (Get-Content -Path $heartbeat -ErrorAction SilentlyContinue | Measure-Object).Count
        $countStillAfterCancel | Should -Be $countAfterCancel
    }

    It "C2: cancels in-flight and never starts the next instruction" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"
        $markerB = Join-Path $scratch "marker-b"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA && ping -n 60 127.0.0.1 > nul"
        )
        Add-PlanInstruction -Plan $plan -Name "b" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerB"
        )
        $planFile = Join-Path $scratch "c2-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        Wait-Ready -Path $markerA -Timeout 30 -Throw

        Invoke-PlanAnnotate -Canceled "true"

        {
            (Get-PlanOutcome).planState -eq "canceled"
        } | Judge -Timeout 30 -Throw

        Test-Path -Path $markerB | Should -Be $false
        Start-Sleep -Seconds 15
        Test-Path -Path $markerB | Should -Be $false

        $outcome = Get-PlanOutcome
        $outcome.planState | Should -Be "canceled"
        $outcome.present.appliedChecksum | Should -Be $false
        $outcome.checkpoint.completedInstructions | Should -BeLessThan $outcome.checkpoint.totalInstructions
        # Cancellation never sets Paused (handleCancellation leaves it zero), so the checkpoint's
        # "paused" key is entirely absent from the JSON, not merely false -- the same distinction
        # a resumed-from-pause checkpoint depends on.
        $outcome.checkpoint.present.paused | Should -Be $false
    }

    It "C3: a canceled plan stays terminal after the annotation is removed" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA && ping -n 60 127.0.0.1 > nul"
        )
        $planFile = Join-Path $scratch "c3-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        Wait-Ready -Path $markerA -Timeout 30 -Throw

        Invoke-PlanAnnotate -Canceled "true"
        {
            (Get-PlanOutcome).planState -eq "canceled"
        } | Judge -Timeout 30 -Throw

        $revisionAfterCancel = (Get-PlanOutcome).planRevision

        Invoke-PlanAnnotate -Canceled ""
        Start-Sleep -Seconds 30

        $outcome = Get-PlanOutcome
        $outcome.planState | Should -Be "canceled"
        $outcome.planRevision | Should -Be $revisionAfterCancel
    }

    It "C4: cancelling a pending plan executes nothing" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
            "/c", "echo started > $markerA"
        )
        $planFile = Join-Path $scratch "c4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" -Canceled "true" | Out-Null

        Start-Sleep -Seconds 40

        Test-Path -Path $markerA | Should -Be $false
        $outcome = Get-PlanOutcome
        $outcome.present.appliedChecksum | Should -Be $false
        $outcome.planState | Should -Be "canceled"
    }

    It "C5: a canceled plan with periodic instructions and probes never runs periodic instructions again, but probes keep updating" {
        $scratch = Get-PlanScratchDirectory
        $markerA = Join-Path $scratch "marker-a"
        $probePort = 18234

        $probeJob = Start-ProbeTargetServer -Port $probePort -StatusCode 500
        try {
            $plan = New-PlanSpec
            Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @(
                "/c", "echo started > $markerA && ping -n 60 127.0.0.1 > nul"
            )
            Add-PlanPeriodicInstruction -Plan $plan -Name "p" -Command "cmd.exe" -Args @("/c", "exit 0") -PeriodSeconds 5
            # FailureThreshold is deliberately large: applyProbeResult (status.go) stops
            # incrementing FailureCount once it reaches the threshold, so a small threshold like
            # the default 1 would cap the count on the very first evaluation and make it
            # impossible to observe further growth across the test's later wait.
            Add-PlanProbe -Plan $plan -Name "probe1" -Url "http://localhost:$probePort/" -FailureThreshold 100
            $planFile = Join-Path $scratch "c5-plan.json"
            Write-PlanFile -Plan $plan -Path $planFile

            Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

            Wait-Ready -Path $markerA -Timeout 30 -Throw

            Invoke-PlanAnnotate -Canceled "true"

            {
                (Get-PlanOutcome).planState -eq "canceled"
            } | Judge -Timeout 30 -Throw

            # The one-time instruction was interrupted before completing, so Apply returned
            # before ever reaching the periodic instructions in that call; a canceled plan is
            # then permanently terminal and reconciles in monitoring-only mode from then on,
            # which never executes periodic instructions again.
            (Get-PlanOutcome).periodicOutput.PSObject.Properties.Name | Should -Not -Contain "p"

            $outcomeAfterCancel = Get-PlanOutcome
            $outcomeAfterCancel.probeStatuses.probe1.healthy | Should -Be $false
            $failureCountAfterCancel = $outcomeAfterCancel.probeStatuses.probe1.failureCount

            # Monitoring-only mode still merges probe statuses on every reconcile (the design's
            # stated reason: keeping health data current for Rancher's MachineHealthCheck even
            # once no further Apply ever runs), so the failure count keeps climbing well after
            # the plan is terminal.
            Start-Sleep -Seconds 15

            $outcomeAfterWait = Get-PlanOutcome
            $outcomeAfterWait.probeStatuses.probe1.failureCount | Should -BeGreaterThan $failureCountAfterCancel
            $outcomeAfterWait.periodicOutput.PSObject.Properties.Name | Should -Not -Contain "p"
            $outcomeAfterWait.planState | Should -Be "canceled"
        }
        finally {
            Stop-ProbeTargetServer -Port $probePort -Job $probeJob
        }
    }

    It "C6: cancelling a succeeded plan with a periodic instruction stops it permanently, even from a state that would otherwise keep reconciling forever" {
        $scratch = Get-PlanScratchDirectory

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "noop" -Command "cmd.exe" -Args @("/c", "exit 0")
        Add-PlanPeriodicInstruction -Plan $plan -Name "p" -Command "cmd.exe" -Args @("/c", "exit 0") -PeriodSeconds 5
        $planFile = Join-Path $scratch "c6-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        # Confirm the periodic instruction is genuinely running on its own cadence before
        # cancelling: a "succeeded" plan keeps calling Apply on every reconcile (unlike a
        # canceled or failed one), which is exactly the behavior cancellation must permanently
        # stop, not just skip once.
        {
            (Get-PlanOutcome).periodicOutput.p.lastSuccessfulRunTime -ne $null
        } | Judge -Timeout 30 -Throw

        # Cancelling a plan with nothing currently in flight is not a race the way cancelling a
        # running instruction is: there is no window during which the agent could execute
        # something the cancel would otherwise have prevented, only a wait for the next ~5s
        # reconcile to observe the annotation.
        Invoke-PlanAnnotate -Canceled "true"
        {
            (Get-PlanOutcome).planState -eq "canceled"
        } | Judge -Timeout 30 -Throw

        $runTimeAtCancel = (Get-PlanOutcome).periodicOutput.p.lastSuccessfulRunTime

        # A canceled plan is permanently terminal and reconciles in monitoring-only mode, which
        # never executes periodic instructions again, unlike "succeeded" which would have kept
        # re-running this one indefinitely.
        Start-Sleep -Seconds 15
        (Get-PlanOutcome).periodicOutput.p.lastSuccessfulRunTime | Should -Be $runTimeAtCancel

        # Cancellation is a one-way transition, even from a state (succeeded) that would
        # otherwise resume periodic work: removing the annotation does not un-cancel the plan.
        Invoke-PlanAnnotate -Canceled ""
        Start-Sleep -Seconds 15

        $outcome = Get-PlanOutcome
        $outcome.planState | Should -Be "canceled"
        $outcome.periodicOutput.p.lastSuccessfulRunTime | Should -Be $runTimeAtCancel
    }
}
