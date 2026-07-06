[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [ValidateSet('Report', 'ClosePages', 'RestartBrowser', 'Auto')]
    [string] $Action = 'Report',

    [string] $ContainerName = 'openclaw4j-aio-sandbox',

    [int] $WorkingSetThresholdMiB = 1536,

    [int] $MaxPageCount = 5
)

$ErrorActionPreference = 'Stop'

function Invoke-DockerExec {
    param(
        [Parameter(Mandatory = $true)]
        [string[]] $ArgumentList
    )

    $output = & docker @('exec', $ContainerName) @ArgumentList
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "docker exec $ContainerName failed with exit code $exitCode"
    }
    return $output
}

function Get-CgroupMemory {
    $currentBytes = [int64] ((Invoke-DockerExec -ArgumentList @('cat', '/sys/fs/cgroup/memory.current')) -join '').Trim()
    $statLines = Invoke-DockerExec -ArgumentList @('cat', '/sys/fs/cgroup/memory.stat')
    $stats = @{}

    foreach ($line in $statLines) {
        $parts = $line -split ' '
        if ($parts.Count -eq 2) {
            $stats[$parts[0]] = [int64] $parts[1]
        }
    }

    $inactiveFileBytes = 0
    if ($stats.ContainsKey('inactive_file')) {
        $inactiveFileBytes = $stats['inactive_file']
    }

    [pscustomobject]@{
        CurrentMiB          = [math]::Round($currentBytes / 1MB, 1)
        InactiveFileMiB     = [math]::Round($inactiveFileBytes / 1MB, 1)
        WorkingSetMiB       = [math]::Round(($currentBytes - $inactiveFileBytes) / 1MB, 1)
        WorkingSetThreshold = $WorkingSetThresholdMiB
    }
}

function Get-ChromeTargets {
    $raw = (Invoke-DockerExec -ArgumentList @('curl', '-s', 'http://127.0.0.1:9222/json/list')) -join "`n"
    if ([string]::IsNullOrWhiteSpace($raw)) {
        return @()
    }

    return @(ConvertFrom-Json -InputObject $raw)
}

function Close-ChromePageTargets {
    param(
        [Parameter(Mandatory = $true)]
        [array] $Targets
    )

    $pageTargets = @($Targets | Where-Object { $_.type -eq 'page' })
    foreach ($target in $pageTargets) {
        $targetLabel = "$($target.title) <$($target.url)>"
        if ($PSCmdlet.ShouldProcess($targetLabel, 'close Chrome page target')) {
            Invoke-DockerExec -ArgumentList @(
                'curl',
                '-s',
                '-X',
                'PUT',
                "http://127.0.0.1:9222/json/close/$($target.id)"
            ) | Out-Null
        }
    }

    return $pageTargets.Count
}

function Restart-SandboxBrowser {
    if ($PSCmdlet.ShouldProcess($ContainerName, 'restart browser and mcp-server-browser supervisor programs')) {
        Invoke-DockerExec -ArgumentList @('supervisorctl', 'restart', 'browser', 'mcp-server-browser')
    }
}

$memory = Get-CgroupMemory
$targets = Get-ChromeTargets
$pages = @($targets | Where-Object { $_.type -eq 'page' })
$workers = @($targets | Where-Object { $_.type -eq 'worker' })

[pscustomobject]@{
    ContainerName        = $ContainerName
    Action               = $Action
    WorkingSetMiB        = $memory.WorkingSetMiB
    CurrentMiB           = $memory.CurrentMiB
    InactiveFileMiB      = $memory.InactiveFileMiB
    PageCount            = $pages.Count
    WorkerCount          = $workers.Count
    ThresholdMiB         = $WorkingSetThresholdMiB
    MaxPageCount         = $MaxPageCount
} | Format-List

if ($pages.Count -gt 0) {
    $pages | Select-Object type, title, url, id | Format-Table -AutoSize -Wrap
}

switch ($Action) {
    'Report' {
        return
    }
    'ClosePages' {
        $closedCount = Close-ChromePageTargets -Targets $targets
        Write-Host "Closed $closedCount Chrome page target(s)."
        return
    }
    'RestartBrowser' {
        Restart-SandboxBrowser
        return
    }
    'Auto' {
        if ($pages.Count -gt $MaxPageCount) {
            $closedCount = Close-ChromePageTargets -Targets $targets
            Write-Host "Closed $closedCount Chrome page target(s) because page count exceeded $MaxPageCount."
        }

        if ($memory.WorkingSetMiB -gt $WorkingSetThresholdMiB) {
            Restart-SandboxBrowser
        }
        return
    }
}
