[CmdletBinding()]
param(
    [switch]$Rebuild,
    [switch]$Online
)

$ErrorActionPreference = 'Stop'

$projectRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$binaryDir = Join-Path $projectRoot 'bin'
$binaryPath = Join-Path $binaryDir 'openclaw4j-backend-go.exe'

# 仅比较会参与 go build 的源码和模块清单；.env 由运行中的程序直接读取，无需重建二进制。
$buildInputs = @(
    Get-ChildItem -LiteralPath $projectRoot -Recurse -Filter '*.go' -File |
        Where-Object { $_.FullName -notlike "$binaryDir\\*" -and $_.Name -notlike '*_test.go' }
    Get-Item -LiteralPath (Join-Path $projectRoot 'go.mod'), (Join-Path $projectRoot 'go.sum')
)
$latestInput = $buildInputs | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
$needsBuild = $Rebuild -or -not (Test-Path -LiteralPath $binaryPath)

if (-not $needsBuild) {
    $needsBuild = (Get-Item -LiteralPath $binaryPath).LastWriteTimeUtc -lt $latestInput.LastWriteTimeUtc
}

if ($needsBuild) {
    New-Item -ItemType Directory -Path $binaryDir -Force | Out-Null

    $previousProxy = $env:GOPROXY
    $previousSumDB = $env:GOSUMDB
    try {
        if (-not $Online) {
            # 依赖缓存齐全时禁止版本元数据探测，避免不可用代理拖慢每次开发启动。
            $env:GOPROXY = 'off'
            $env:GOSUMDB = 'off'
        }

        Write-Host 'Building Go backend...'
        $goArgs = @('build', '-o', $binaryPath, './cmd/openclaw4j-backend-go')
        & go @goArgs
        $exitCode = $LASTEXITCODE
        if ($exitCode -ne 0) {
            if (-not $Online) {
                throw "go build failed with exit code $exitCode. Use -Online if the local module cache is incomplete."
            }
            throw "go build failed with exit code $exitCode."
        }
    }
    finally {
        $env:GOPROXY = $previousProxy
        $env:GOSUMDB = $previousSumDB
    }
}
else {
    Write-Host 'Using current Go backend binary.'
}

Write-Host "Starting Go backend: $binaryPath"
& $binaryPath
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    throw "Go backend exited with code $exitCode."
}
