param(
    [string]$BaseRef = "HEAD",
    [switch]$FullSpotBugs,
    [switch]$SkipSpotBugs
)

$ErrorActionPreference = "Stop"

$BackendRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RepoRoot = (Resolve-Path (Join-Path $BackendRoot "..")).Path
$MavenRepo = "D:\apache-maven-3.9.1\m2\repository"

$env:JAVA_HOME = "D:\jdk-26"
$env:Path = "D:\jdk-26\bin;" + $env:Path

function Invoke-MavenStep {
    param(
        [string]$Name,
        [string[]]$Arguments
    )

    Write-Host ""
    Write-Host "==> $Name"
    Write-Host ("mvn " + ($Arguments -join " "))
    Push-Location $BackendRoot
    try {
        & mvn @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$Name failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }
}

function Get-ChangedBackendJavaFiles {
    $paths = @(
        "OpenClaw4j-Bankend/src/main/java",
        "OpenClaw4j-Bankend/src/test/java"
    )

    $tracked = & git -C $RepoRoot diff --name-only --diff-filter=ACMR $BaseRef -- $paths
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read changed files from git diff against $BaseRef"
    }

    $untracked = & git -C $RepoRoot ls-files --others --exclude-standard -- $paths
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to read untracked backend Java files"
    }

    @($tracked + $untracked) |
        Where-Object { $_ -like "*.java" } |
        Sort-Object -Unique
}

$changedJavaFiles = @(Get-ChangedBackendJavaFiles)
$changedMainJavaFiles = @($changedJavaFiles | Where-Object { $_ -like "OpenClaw4j-Bankend/src/main/java/*" })

Write-Host "Backend fast quality check"
Write-Host "Base ref: $BaseRef"
Write-Host "Changed backend Java files: $($changedJavaFiles.Count)"
foreach ($file in $changedJavaFiles) {
    Write-Host "  $file"
}

Invoke-MavenStep `
    -Name "Spotless check" `
    -Arguments @("-Dmaven.repo.local=$MavenRepo", "spotless:check")

if ($changedJavaFiles.Count -gt 0) {
    $includePatterns = $changedJavaFiles |
        ForEach-Object { "**/$([System.IO.Path]::GetFileName($_))" } |
        Sort-Object -Unique
    $checkstyleIncludes = $includePatterns -join ","

    Invoke-MavenStep `
        -Name "Checkstyle changed Java files" `
        -Arguments @(
            "-Dmaven.repo.local=$MavenRepo",
            "-Dcheckstyle.includes=$checkstyleIncludes",
            "checkstyle:check"
        )
}
else {
    Write-Host ""
    Write-Host "==> Checkstyle changed Java files"
    Write-Host "No backend Java changes; skipped."
}

if ($SkipSpotBugs) {
    Write-Host ""
    Write-Host "==> SpotBugs"
    Write-Host "Skipped by -SkipSpotBugs."
}
elseif ($FullSpotBugs) {
    Invoke-MavenStep `
        -Name "SpotBugs full check" `
        -Arguments @("-Dmaven.repo.local=$MavenRepo", "spotbugs:check")
}
elseif ($changedMainJavaFiles.Count -gt 0) {
    Invoke-MavenStep `
        -Name "SpotBugs fast check" `
        -Arguments @(
            "-Dmaven.repo.local=$MavenRepo",
            "-Dspotbugs.effort=Default",
            "spotbugs:check"
        )
}
else {
    Write-Host ""
    Write-Host "==> SpotBugs fast check"
    Write-Host "No main Java changes; skipped because SpotBugs does not analyze tests by default."
}

Write-Host ""
Write-Host "Backend fast quality check completed."
Write-Host "Before reporting final backend code completion, still run the full quality gate when required."
