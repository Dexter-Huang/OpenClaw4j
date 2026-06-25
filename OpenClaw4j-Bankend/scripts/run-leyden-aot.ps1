param(
    [string]$JarPath = "target/OpenClaw4j-Bankend.jar",
    [string]$CachePath = "target/openclaw4j.aot",
    [string]$SpringProfile = "dev",
    [int]$ServerPort = 9004
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
    throw "Jar not found: $JarPath. Run: mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package"
}

if (-not (Test-Path -LiteralPath $CachePath)) {
    throw "AOT cache not found: $CachePath. Run: ./scripts/train-leyden-aot.ps1"
}

$extractDir = Join-Path (Split-Path -Parent $JarPath) "leyden-extracted"
if (-not (Test-Path -LiteralPath $extractDir)) {
    java -Djarmode=tools -jar $JarPath extract --destination $extractDir
}

$appJar = "OpenClaw4j-Bankend.jar"
$classpath = Get-LeydenClasspath -ExtractDir $extractDir -AppJar $appJar
$resolvedCachePath = Join-Path $projectDir $CachePath

Push-Location $extractDir
try {
    java "-XX:AOTCache=$resolvedCachePath" `
        "-Dspring.profiles.active=$SpringProfile" `
        -cp $classpath `
        com.seaskyland.llm.LLMApplication `
        "--server.port=$ServerPort"
}
finally {
    Pop-Location
}
