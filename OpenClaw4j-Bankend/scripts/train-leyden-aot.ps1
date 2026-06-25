param(
    [string]$JarPath = "target/OpenClaw4j-Bankend.jar",
    [string]$CachePath = "target/openclaw4j.aot",
    [string]$SpringProfile = "dev",
    [int]$ServerPort = 0,
    [int]$TrainingTimeoutSeconds = 900,
    [ValidateSet("profiled", "classpath", "spring")]
    [string]$TrainingMode = "profiled",
    [string]$WarmupPaths = "/console/v1/system/health,/console/v1/system/global-config,/api/prompts?pageNo=1&pageSize=1,/api/observability/overview"
)

$ErrorActionPreference = "Stop"

function Get-LeydenClasspath {
    param(
        [string]$ExtractDir,
        [string]$AppJar
    )

    return @($AppJar, (Join-Path "lib" "*")) -join [System.IO.Path]::PathSeparator
}

$projectDir = Split-Path -Parent $PSScriptRoot
Set-Location $projectDir

if (-not (Test-Path -LiteralPath $JarPath)) {
    mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package
}

$extractDir = Join-Path (Split-Path -Parent $JarPath) "leyden-extracted"
$resolvedProjectDir = (Resolve-Path -LiteralPath $projectDir).Path
$targetDir = Join-Path $resolvedProjectDir "target"
$resolvedExtractParent = (Resolve-Path -LiteralPath (Split-Path -Parent $extractDir)).Path
if (-not $resolvedExtractParent.StartsWith($targetDir, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to clean extract directory outside target: $extractDir"
}

$existingTrainingProcesses = Get-CimInstance Win32_Process -Filter "name = 'java.exe'" |
    Where-Object { $_.CommandLine -like "*AOTCacheOutput=*$resolvedProjectDir*" }
foreach ($existingProcess in $existingTrainingProcesses) {
    Stop-Process -Id $existingProcess.ProcessId -Force
}

if (Test-Path -LiteralPath $extractDir) {
    Remove-Item -LiteralPath $extractDir -Recurse -Force
}

java -Djarmode=tools -jar $JarPath extract --destination $extractDir

if (Test-Path -LiteralPath $CachePath) {
    Remove-Item -LiteralPath $CachePath -Force
}

$resolvedCachePath = Join-Path $projectDir $CachePath
$appJar = "OpenClaw4j-Bankend.jar"
$classpath = Get-LeydenClasspath -ExtractDir $extractDir -AppJar $appJar

$javaArgs = @(
    "-XX:AOTCacheOutput=$resolvedCachePath",
    "-cp",
    $classpath
)

if ($TrainingMode -eq "profiled") {
    $javaArgs += @(
        "-Dspring.profiles.active=$SpringProfile",
        "-Dopenclaw4j.leyden.training.profiled=true",
        "-Dopenclaw4j.leyden.training.warmup-paths=$WarmupPaths",
        "com.seaskyland.llm.LLMApplication",
        "--server.port=$ServerPort"
    )
}
elseif ($TrainingMode -eq "spring") {
    $javaArgs += @(
        "-Dspring.context.exit=onRefresh",
        "-Dspring.profiles.active=$SpringProfile",
        "com.seaskyland.llm.LLMApplication",
        "--server.port=$ServerPort"
    )
}
else {
    $javaArgs += "com.seaskyland.llm.LeydenTrainingApplication"
}

$workingDirectory = (Resolve-Path -LiteralPath $extractDir).Path
$job = Start-Job -ScriptBlock {
    param($directory, $arguments)
    Set-Location $directory
    & java @arguments
    exit $LASTEXITCODE
} -ArgumentList $workingDirectory, $javaArgs

if (-not (Wait-Job -Job $job -Timeout $TrainingTimeoutSeconds)) {
    Stop-Job -Job $job
    $childProcesses = Get-CimInstance Win32_Process -Filter "name = 'java.exe'" |
        Where-Object { $_.CommandLine -like "*AOTCacheOutput=$resolvedCachePath*" }
    foreach ($childProcess in $childProcesses) {
        Stop-Process -Id $childProcess.ProcessId -Force
    }
    throw "Training process did not exit after $TrainingTimeoutSeconds seconds"
}

$jobOutput = Receive-Job -Job $job -ErrorAction Continue 2>&1
if ($jobOutput) {
    $jobOutput | ForEach-Object { Write-Host $_ }
}
if ($job.State -eq "Failed") {
    if (-not (Test-Path -LiteralPath $CachePath)) {
        throw "Training process failed"
    }
}
Remove-Job -Job $job -Force

if (-not (Test-Path -LiteralPath $CachePath)) {
    throw "Leyden AOT cache was not created: $CachePath"
}

Write-Host "Leyden AOT cache written to $CachePath"
