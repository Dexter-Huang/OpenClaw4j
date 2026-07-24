[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($env:OPENCLAW_TEST_PGVECTOR_DSN)) {
    throw 'OPENCLAW_TEST_PGVECTOR_DSN is required for pgvector integration tests.'
}

$go = (Get-Command go -ErrorAction Stop).Source
$arguments = @('test', '-tags=integration', './internal/rag')
& $go @arguments
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    exit $exitCode
}
