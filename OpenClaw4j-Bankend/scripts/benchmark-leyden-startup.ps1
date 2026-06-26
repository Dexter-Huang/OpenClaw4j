param(
    [string]$JarPath = "target/OpenClaw4j-Bankend.jar",
    [string]$CachePath = "target/openclaw4j.aot",
    [string]$SpringProfile = "dev",
    [int]$ServerPort = 0,
    [int]$Runs = 2,
    [int]$TimeoutSeconds = 150,
    [string]$ProbePaths = "/console/v1/system/health,/console/v1/system/global-config,/api/prompts?pageNo=1&pageSize=1,/api/observability/overview",
    [int]$ProbeTimeoutSeconds = 10,
    [string]$LeydenJvmOptions = "-XX:+UnlockExperimentalVMOptions -XX:+UseCompactObjectHeaders"
)

$ErrorActionPreference = "Stop"
$JavaExe = "D:\jdk-26\bin\java.exe"
Add-Type -AssemblyName System.Net.Http

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

function Get-ProbePaths {
    param(
        [string]$Paths
    )

    if ([string]::IsNullOrWhiteSpace($Paths)) {
        return @()
    }

    return $Paths -split "," |
        ForEach-Object { $_.Trim() } |
        Where-Object { $_.Length -gt 0 }
}

function Get-LeydenJvmOptions {
    param(
        [string]$Options
    )

    if ([string]::IsNullOrWhiteSpace($Options)) {
        return @()
    }

    return $Options -split "\s+" |
        ForEach-Object { $_.Trim() } |
        Where-Object { $_.Length -gt 0 }
}

function Invoke-FirstRequestProbes {
    param(
        [int]$Port,
        [string[]]$Paths
    )

    if ($Port -le 0 -or $Paths.Count -eq 0) {
        return @()
    }

    $client = [System.Net.Http.HttpClient]::new()
    $client.Timeout = [TimeSpan]::FromSeconds($ProbeTimeoutSeconds)
    $results = @()
    try {
        foreach ($path in $Paths) {
            $normalizedPath = if ($path.StartsWith("/")) { $path } else { "/" + $path }
            $uri = "http://127.0.0.1:$Port$normalizedPath"
            $timer = [Diagnostics.Stopwatch]::StartNew()
            $status = "failed"
            $statusCode = 0
            try {
                $response = $client.GetAsync($uri).GetAwaiter().GetResult()
                $statusCode = [int]$response.StatusCode
                $status = "completed"
                $response.Dispose()
            }
            catch {
                $status = "failed"
            }
            finally {
                $timer.Stop()
            }

            $results += [pscustomobject]@{
                path = $normalizedPath
                status = $status
                statusCode = $statusCode
                firstRequestMilliseconds = [math]::Round($timer.Elapsed.TotalMilliseconds, 1)
            }
        }
    }
    finally {
        $client.Dispose()
    }

    return $results
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
    $javaArgs = @("--enable-final-field-mutation=ALL-UNNAMED")
    if ($UseLeyden) {
        $javaArgs += Get-LeydenJvmOptions -Options $LeydenJvmOptions
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
    $process = Start-Process -FilePath $JavaExe -ArgumentList $javaArgs -WorkingDirectory $WorkingDirectory -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru
    $status = "timeout"
    $startedLine = ""
    $rootLine = ""
    $actualPort = $ServerPort
    $probeResults = @()

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
                $portCandidate = $lines | Where-Object { $_ -like "*Tomcat started on port *" } | Select-Object -Last 1
                if ($portCandidate -and $portCandidate -match "Tomcat started on port ([0-9]+)") {
                    $actualPort = [int]$matches[1]
                }
                $status = "started"
                $probeResults = @(Invoke-FirstRequestProbes -Port $actualPort -Paths $script:ResolvedProbePaths)
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
        actualPort = $actualPort
        rootLine = $rootLine
        startedLine = $startedLine
        firstRequestMilliseconds = if (@($probeResults).Count -gt 0) { [math]::Round(($probeResults | Measure-Object firstRequestMilliseconds -Sum).Sum, 1) } else { $null }
        probeResults = $probeResults
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
$resolvedProjectDir = (Resolve-Path -LiteralPath $projectDir).Path
$targetDir = Join-Path $resolvedProjectDir "target"
$resolvedExtractParent = (Resolve-Path -LiteralPath (Split-Path -Parent $extractDir)).Path
if (-not $resolvedExtractParent.StartsWith($targetDir, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to clean extract directory outside target: $extractDir"
}
if (Test-Path -LiteralPath $extractDir) {
    Remove-Item -LiteralPath $extractDir -Recurse -Force
}
& $JavaExe -Djarmode=tools -jar $JarPath extract --destination $extractDir

$workingDirectory = (Resolve-Path -LiteralPath $extractDir).Path
$resolvedCachePath = (Resolve-Path -LiteralPath $CachePath).Path
$classpath = Get-LeydenClasspath -AppJar "OpenClaw4j-Bankend.jar"
$script:ResolvedProbePaths = @(Get-ProbePaths -Paths $ProbePaths)
$results = @()

for ($i = 1; $i -le $Runs; $i++) {
    $results += Invoke-StartupProbe -Name "baseline-$i" -UseLeyden:$false -WorkingDirectory $workingDirectory -Classpath $classpath -ResolvedCachePath $resolvedCachePath
    $results += Invoke-StartupProbe -Name "leyden-$i" -UseLeyden:$true -WorkingDirectory $workingDirectory -Classpath $classpath -ResolvedCachePath $resolvedCachePath
}

$results | Format-Table run, leyden, status, springStartedSeconds, processRunningSeconds, firstRequestMilliseconds, wallSeconds -AutoSize
if ($script:ResolvedProbePaths.Count -gt 0) {
    Write-Host ""
    Write-Host "First request probes:"
    $results |
        ForEach-Object {
            $run = $_.run
            $leyden = $_.leyden
            $_.probeResults | ForEach-Object {
                [pscustomobject]@{
                    run = $run
                    leyden = $leyden
                    path = $_.path
                    status = $_.status
                    statusCode = $_.statusCode
                    firstRequestMilliseconds = $_.firstRequestMilliseconds
                }
            }
        } |
        Format-Table -AutoSize
}
Write-Host ""
Write-Host "Average by mode:"
$results |
    Group-Object leyden |
    ForEach-Object {
        $firstRequestAverage = ($_.Group | Measure-Object firstRequestMilliseconds -Average).Average
        [pscustomobject]@{
            leyden = $_.Name
            runs = $_.Count
            avgSpringStartedSeconds = [math]::Round(($_.Group | Measure-Object springStartedSeconds -Average).Average, 3)
            avgProcessRunningSeconds = [math]::Round(($_.Group | Measure-Object processRunningSeconds -Average).Average, 3)
            avgFirstRequestMilliseconds = if ($null -ne $firstRequestAverage) { [math]::Round($firstRequestAverage, 1) } else { $null }
        }
    } |
    Format-Table -AutoSize
