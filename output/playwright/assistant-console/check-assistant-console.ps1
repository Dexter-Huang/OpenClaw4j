$ErrorActionPreference = 'Stop'

$session = 'assistant-console-check'
$targetUrl = 'http://localhost:8000/app/assistant/2072641584112357378'
$cliArgs = @('--yes', '--package', '@playwright/cli', 'playwright-cli', "-s=$session")

function Invoke-PwCli {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]] $Arguments
    )

    $output = & npx @cliArgs @Arguments 2>&1
    $exitCode = $LASTEXITCODE
    $text = $output -join [Environment]::NewLine
    if ($exitCode -ne 0) {
        throw "playwright-cli $($Arguments -join ' ') failed with exit code $exitCode`n$text"
    }
    return $text
}

try {
    $loginResponse = Invoke-RestMethod `
        -Method Post `
        -Uri 'http://127.0.0.1:9004/console/v1/auth/login' `
        -ContentType 'application/json' `
        -Body (@{ username = 'saa'; password = '123456' } | ConvertTo-Json)
    $sessionData = [ordered]@{
        access_token = $loginResponse.data.access_token
        refresh_token = $loginResponse.data.refresh_token
        expires_time = $loginResponse.data.expires_in * 1000
    } | ConvertTo-Json -Compress

    Invoke-PwCli delete-data | Out-Null
    Invoke-PwCli open 'http://localhost:8000/' | Out-Null
    Invoke-PwCli localstorage-set 'data-prefers-session' $sessionData | Out-Null
    Invoke-PwCli goto $targetUrl | Out-Null
    $deadline = (Get-Date).AddSeconds(60)
    $snapshot = ''
    do {
        Start-Sleep -Seconds 2
        $snapshot = Invoke-PwCli snapshot
        if ($snapshot -match 'ChatBot' -and $snapshot -match 'API Configuration') {
            break
        }
    } while ((Get-Date) -lt $deadline)

    if ($snapshot -notmatch 'ChatBot' -or $snapshot -notmatch 'API Configuration') {
        throw "Assistant page did not finish rendering expected content.`n$snapshot"
    }

    $console = Invoke-PwCli console
    $requests = Invoke-PwCli requests
    $console | Set-Content -Path (Join-Path $PSScriptRoot 'last-console.log') -Encoding utf8
    $requests | Set-Content -Path (Join-Path $PSScriptRoot 'last-requests.log') -Encoding utf8

    $patterns = @(
        'validateDOMNesting',
        'Panel ".+" has an invalid configuration',
        'Invalid layout total size',
        'findDOMNode is deprecated',
        '\[antd: Tooltip\] `overlay(?:InnerStyle|ClassName|Style)` is deprecated'
    )

    $matches = foreach ($pattern in $patterns) {
        if ($console -match $pattern) {
            $pattern
        }
    }

    $failedRequests = @()
    foreach ($line in ($requests -split "`r?`n")) {
        if ($line -match '=> \[(4\d\d|5\d\d)\]') {
            $failedRequests += $line
        }
    }

    $report = [ordered]@{
        url = $targetUrl
        matchedWarnings = @($matches)
        failedRequests = $failedRequests
        consoleSummary = ($console -split "`r?`n" | Select-Object -First 8)
    }
    $report | ConvertTo-Json -Depth 5

    if ($matches.Count -gt 0 -or $failedRequests.Count -gt 0) {
        exit 1
    }
}
finally {
    try {
        Invoke-PwCli close | Out-Null
    }
    catch {
        Write-Warning $_
    }
}
