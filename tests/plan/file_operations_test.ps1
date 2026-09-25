# file_operations_test.ps1 exercises plan file write, delete, and directory operations.

Describe "File operations" {
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

    It "F1: writes a file with the expected content" {
        $target = Join-Path (Get-PlanScratchDirectory) "f1.txt"
        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $target -Content "f1 content"
        $planFile = Join-Path (Get-PlanScratchDirectory) "f1-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-Content -Path $target -Raw) | Should -Be "f1 content"
    }

    It "F2: creates a directory when directory is true" {
        $target = Join-Path (Get-PlanScratchDirectory) "f2-dir"
        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $target -Directory
        $planFile = Join-Path (Get-PlanScratchDirectory) "f2-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-Item -Path $target).PSIsContainer | Should -Be $true
    }

    It "F3: creates missing parent directories for a deep path" {
        $target = Join-Path (Get-PlanScratchDirectory) "a\b\c\f3.txt"
        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $target -Content "deep content"
        $planFile = Join-Path (Get-PlanScratchDirectory) "f3-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-Content -Path $target -Raw) | Should -Be "deep content"
    }

    It "F4: deletes a file with action: delete" {
        $target = Join-Path (Get-PlanScratchDirectory) "f4.txt"
        Set-Content -Path $target -Value "to be deleted"

        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $target -Delete
        $planFile = Join-Path (Get-PlanScratchDirectory) "f4-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        Test-Path -Path $target | Should -Be $false
    }

    It "F5: recursively deletes a directory with directory: true and action: delete" {
        $targetDir = Join-Path (Get-PlanScratchDirectory) "f5-dir"
        New-Item -Path $targetDir -ItemType Directory -Force | Out-Null
        Set-Content -Path (Join-Path $targetDir "child.txt") -Value "child content"

        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $targetDir -Directory -Delete
        $planFile = Join-Path (Get-PlanScratchDirectory) "f5-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        Test-Path -Path $targetDir | Should -Be $false
    }

    It "F6: does not rewrite a file whose content is unchanged" {
        $target = Join-Path (Get-PlanScratchDirectory) "f6.txt"
        $plan = New-PlanSpec
        Add-PlanFile -Plan $plan -Path $target -Content "stable content"
        $planFile = Join-Path (Get-PlanScratchDirectory) "f6-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        $firstWriteTime = (Get-Item -Path $target).LastWriteTimeUtc
        Start-Sleep -Seconds 3

        # Re-applying the identical plan content forces a second pending->succeeded pass;
        # writeContentToFile (file.go) skips the write when the content on disk already matches.
        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null
        {
            (Get-PlanOutcome).planRevision -gt 1
        } | Judge -Timeout 60 -Throw
        {
            (Get-PlanOutcome).planState -eq "succeeded"
        } | Judge -Timeout 60 -Throw

        (Get-Item -Path $target).LastWriteTimeUtc | Should -Be $firstWriteTime
    }

    It "F7: aborts the apply when deleting a non-empty directory without directory: true" {
        $targetDir = Join-Path (Get-PlanScratchDirectory) "f7-dir"
        New-Item -Path $targetDir -ItemType Directory -Force | Out-Null
        Set-Content -Path (Join-Path $targetDir "child.txt") -Value "child content"
        $marker = Join-Path (Get-PlanScratchDirectory) "f7-marker"

        $plan = New-PlanSpec
        # directory omitted: this is a non-directory delete of a path that is actually a
        # non-empty directory, so os.Remove fails with ERROR_DIR_NOT_EMPTY.
        $plan.files += , @{ path = $targetDir; action = "delete" }
        Add-PlanInstruction -Plan $plan -Name "after" -Command "cmd.exe" -Args @("/c", "echo started > $marker")
        $planFile = Join-Path (Get-PlanScratchDirectory) "f7-plan.json"
        Write-PlanFile -Plan $plan -Path $planFile

        Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

        # reconcileFiles fails before any one-time instruction runs. Apply returns the error to
        # reconcileSecret, which returns it up the stack without writing a terminal plan-state:
        # the pending->in-progress transition already committed, so plan-state is stuck at
        # in-progress rather than moving to failed. The whole Apply, including the later
        # instruction, is aborted.
        {
            (Get-PlanOutcome).planState -eq "in-progress"
        } | Judge -Timeout 30 -Throw

        Start-Sleep -Seconds 10
        (Get-PlanOutcome).planState | Should -Be "in-progress"

        Test-Path -Path $targetDir | Should -Be $true
        Test-Path -Path $marker | Should -Be $false
    }
}
