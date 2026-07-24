[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$go = (Get-Command go -ErrorAction Stop).Source
$arguments = @('test', './...')
& $go @arguments
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    exit $exitCode
}
