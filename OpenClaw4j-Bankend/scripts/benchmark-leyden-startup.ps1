param(
    [string]$JarPath = "target/OpenClaw4j-Bankend.jar",
    [string]$CachePath = "target/openclaw4j.aot",
    [string]$SpringProfile = "dev",
    [int]$ServerPort = 0,
    [int]$Runs = 2,
    [int]$TimeoutSeconds = 150
)

$ErrorActionPreference = "Stop"

function Get-LeydenClasspath {
    param(
        [string]$AppJar
    )

    return @($AppJar, (Join-Path "lib" "*")) -join [System.IO.Path]::PathSeparator
}

function Stop-BenchmarkProcesses {
    Get-CimInstance Win32_Process -Filter "name = 'java.exe'" |
        Where-Object { $_.CommandLine -like "*com.seaskyland.llm.LLMApplication*" -and $_.CommandLine -like "*OpenClaw4j-Bankend.jar*" } |
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
}

function Invoke-StartupProbe {
    param(
        [string]$Name,
        [bool]$UseLeyden,
        [string]$WorkingDirectory,
        [string]$Classpath,
        [string]$ResolvedCachePath
    )

    Stop-BenchmarkProcesses
    Start-Sleep -Milliseconds 500

    $stdout = Join-Path $env:TEMP ("openclaw-startup-" + $Name + "-" + (Get-Random) + ".out.log")
    $stderr = Join-Path $env:TEMP ("openclaw-startup-" + $Name + "-" + (Get-Random) + ".err.log")
    $javaArgs = @()
    if ($UseLeyden) {
        $javaArgs += "-XX:AOTCache=$ResolvedCachePath"
    }
    $javaArgs += @(
        "-Dspring.profiles.active=$SpringProfile",
        "-cp",
        $Classpath,
        "com.seaskyland.llm.LLMApplication",
        "--server.port=$ServerPort"
    )

    $timer = [Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath "java" -ArgumentList $javaArgs -WorkingDirectory $WorkingDirectory -RedirectStandardOutput $stdout -RedirectStandardError $stderr -NoNewWindow -PassThru
    $status = "timeout"
    $startedLine = ""
    $rootLine = ""

    try {
        while ($timer.Elapsed.TotalSeconds -lt $TimeoutSeconds) {
            Start-Sleep -Milliseconds 500
            $process.Refresh()
            $output = if (Test-Path -LiteralPath $stdout) { Get-Content -LiteralPath $stdout -Raw -ErrorAction SilentlyContinue } else { "" }
            $lines = ($output -split "`r?`n") | Where-Object { $_.Trim().Length -gt 0 }
            $candidate = $lines | Where-Object { $_ -like "*Started LLMApplication in * seconds*" } | Select-Object -Last 1
            if ($candidate) {
                $startedLine = $candidate
                $rootCandidate = $lines | Where-Object { $_ -like "*Root WebApplicationContext: initialization completed in*" } | Select-Object -Last 1
                if ($rootCandidate) {
                    $rootLine = $rootCandidate
                }
                $status = "started"
                break
            }
            if ($process.HasExited) {
                $status = "exited"
                $startedLine = $lines | Select-Object -Last 1
                break
            }
        }
    }
    finally {
        $wallSeconds = [math]::Round($timer.Elapsed.TotalSeconds, 3)
        Stop-BenchmarkProcesses
    }

    $springSeconds = $null
    $processSeconds = $null
    if ($startedLine -match "Started LLMApplication in ([0-9.]+) seconds \(process running for ([0-9.]+)\)") {
        $springSeconds = [double]$matches[1]
        $processSeconds = [double]$matches[2]
    }

    [pscustomobject]@{
        run = $Name
        leyden = $UseLeyden
        status = $status
        wallSeconds = $wallSeconds
        springStartedSeconds = $springSeconds
        processRunningSeconds = $processSeconds
        rootLine = $rootLine
        startedLine = $startedLine
        stdout = $stdout
        stderr = $stderr
    }
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

$workingDirectory = (Resolve-Path -LiteralPath $extractDir).Path
$resolvedCachePath = (Resolve-Path -LiteralPath $CachePath).Path
$classpath = Get-LeydenClasspath -AppJar "OpenClaw4j-Bankend.jar"
$results = @()

for ($i = 1; $i -le $Runs; $i++) {
    $results += Invoke-StartupProbe -Name "baseline-$i" -UseLeyden:$false -WorkingDirectory $workingDirectory -Classpath $classpath -ResolvedCachePath $resolvedCachePath
    $results += Invoke-StartupProbe -Name "leyden-$i" -UseLeyden:$true -WorkingDirectory $workingDirectory -Classpath $classpath -ResolvedCachePath $resolvedCachePath
}

$results | Format-Table run, leyden, status, springStartedSeconds, processRunningSeconds, wallSeconds -AutoSize
Write-Host ""
Write-Host "Average by mode:"
$results |
    Group-Object leyden |
    ForEach-Object {
        [pscustomobject]@{
            leyden = $_.Name
            runs = $_.Count
            avgSpringStartedSeconds = [math]::Round(($_.Group | Measure-Object springStartedSeconds -Average).Average, 3)
            avgProcessRunningSeconds = [math]::Round(($_.Group | Measure-Object processRunningSeconds -Average).Average, 3)
        }
    } |
    Format-Table -AutoSize
