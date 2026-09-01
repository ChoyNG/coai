#Requires -Version 5.1
<#
.SYNOPSIS
    Stop the local CoAI stack started by start-local.ps1.

.EXAMPLE
    .\stop-local.ps1
    .\stop-local.ps1 -Volumes
#>

[CmdletBinding()]
param(
    # Also remove the attached volumes (deletes local MySQL / Redis data).
    [switch]$Volumes
)

$ErrorActionPreference = "Stop"

$StableCompose = "docker-compose.stable.yaml"
$LocalCompose  = "docker-compose.local.yaml"

function Test-Command {
    param([string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Host "[stop-local] ERROR: '$Name' was not found in PATH." -ForegroundColor Red
        exit 1
    }
}

Test-Command "docker"

Push-Location $PSScriptRoot
try {
    if ($Volumes) {
        Write-Host "[stop-local] Stopping services and removing volumes..." -ForegroundColor Cyan
        & docker compose -f $StableCompose -f $LocalCompose down -v
    }
    else {
        Write-Host "[stop-local] Stopping services..." -ForegroundColor Cyan
        & docker compose -f $StableCompose -f $LocalCompose down
    }

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[stop-local] ERROR: docker compose down failed with exit code $LASTEXITCODE." -ForegroundColor Red
        exit $LASTEXITCODE
    }

    Write-Host "[stop-local] Done." -ForegroundColor Green
}
finally {
    Pop-Location
}
