# instruction_execution_test.ps1 exercises command instructions, output capture, environment
# injection, and exit/failure handling.

Describe "Instruction execution" {
    BeforeAll {
        Import-Module -Name "$PSScriptRoot\planutils.psm1" -WarningAction Ignore -Force
        Set-PlanKubeconfig -Path $env:WINS_PLAN_E2E_KUBECONFIG
        Assert-WinsServiceRunning
        $script:T0 = Get-Date
        $script:AppliedPlanDir = "C:/var/lib/rancher/agent/applied"
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

    It "I1: runs a cmd.exe instruction and captures stdout" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i1" -Command "cmd.exe" -Args @("/c", "echo hello-from-i1") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        $checksum = Invoke-PlanApply -PlanFile $planFile -State "pending"

        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $outcome = Get-PlanOutcome
        $outcome.output.i1 | Should -Match "hello-from-i1"

        {
            $applied = Get-ChildItem -Path $script:AppliedPlanDir -Filter "*-applied.plan" -ErrorAction SilentlyContinue |
                Sort-Object LastWriteTime -Descending | Select-Object -First 1
            $applied -and ((Get-Content -Path $applied.FullName -Raw) -match [regex]::Escape($checksum))
        } | Judge -Timeout 30 -Throw
    }

    It "I2: captures both stdout and stderr in one-time output" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i2" -Command "cmd.exe" -Args @(
            "/c", "echo stdout-line & echo stderr-line 1>&2"
        ) -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i2-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $output = (Get-PlanOutcome).output.i2
        $output | Should -Match "stdout-line"
        $output | Should -Match "stderr-line"
    }

    It "I3: injects plan-supplied environment variables" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i3" -Command "cmd.exe" -Args @("/c", "echo %WINS_E2E_FOO%") `
            -Env @("WINS_E2E_FOO=bar") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i3-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).output.i3 | Should -Match "bar"
    }

    It "I4: sets CATTLE_AGENT_EXECUTION_PWD" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i4" -Command "cmd.exe" -Args @("/c", "echo %CATTLE_AGENT_EXECUTION_PWD% & cd") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        $checksum = Invoke-PlanApply -PlanFile $planFile -State "pending"
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $output = (Get-PlanOutcome).output.i4
        $output | Should -Match ([regex]::Escape("C:/var/lib/rancher/agent/work") -replace "/", "[\\/]")
        $output | Should -Match ([regex]::Escape("${checksum}_0"))
    }

    It "I5: sets CATTLE_AGENT_ATTEMPT_NUMBER to 1 on the first attempt" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i5" -Command "cmd.exe" -Args @("/c", "echo %CATTLE_AGENT_ATTEMPT_NUMBER%") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i5-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).output.i5.Trim() | Should -Be "1"
    }

    It "I6: stops after a failing instruction" {
        $markerB = Join-Path (Get-PlanScratchDirectory) "marker-b"

        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "a" -Command "cmd.exe" -Args @("/c", "exit 1") -SaveOutput
        Add-PlanInstruction -Plan $plan -Name "b" -Command "cmd.exe" -Args @("/c", "echo started > $markerB") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i6-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed"
        } | Judge -Timeout 60 -Throw

        $outcome = Get-PlanOutcome
        $outcome.failedOutput.PSObject.Properties.Name | Should -Contain "a"
        $outcome.failedOutput.PSObject.Properties.Name | Should -Not -Contain "b"
        Test-Path -Path $markerB | Should -Be $false
        $outcome.failureCount | Should -Be 1
    }

    It "I7: omits an instruction when saveOutput is false" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i7" -Command "cmd.exe" -Args @("/c", "echo not-saved")
        $planFile = Join-Path (Get-PlanScratchDirectory) "i7-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).output.PSObject.Properties.Name | Should -Not -Contain "i7"
    }

    It "I8: fails predictably when command is omitted" {
        # Documents that every Windows plan instruction must set command: applyinator's
        # defaultCommand ("/run.sh") is applied by string concatenation and has no Windows form.
        $plan = New-PlanSpec
        $plan.instructions += , @{ name = "i8"; args = @(); env = @(); saveOutput = $true }
        $planFile = Join-Path (Get-PlanScratchDirectory) "i8-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "failed"
        } | Judge -Timeout 60 -Throw

        (Get-PlanOutcome).planState | Should -Be "failed"
    }

    It "I9: the execution directory is not on PATH" {
        $plan = New-PlanSpec
        Add-PlanInstruction -Plan $plan -Name "i9" -Command "cmd.exe" -Args @("/c", "echo %PATH%") -SaveOutput
        $planFile = Join-Path (Get-PlanScratchDirectory) "i9-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        $checksum = Invoke-PlanApply -PlanFile $planFile -State "pending"
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $output = (Get-PlanOutcome).output.i9.Trim()
        # applyinator.go joins the execution directory onto PATH with a colon, not a semicolon
        # (see the "PATH="+os.Getenv("PATH")+":"+executionDir concatenation), pinning this as
        # observed Windows behavior rather than the platform-correct separator.
        $output | Should -Match ":.*${checksum}_0$"
    }
}
