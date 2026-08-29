#Requires -Version 5.1
<#
.SYNOPSIS
    Start CoAI locally (built from source) together with MySQL and Redis.

.DESCRIPTION
    Merges docker-compose.stable.yaml with docker-compose.local.yaml so that the
    `chatnio` service is built from this working tree instead of pulling the
    prebuilt programzmh/chatnio:stable image.

    This script never contains and never asks for your Sub2API API key.
    The key is entered later in the CoAI admin panel and stored in the database.

.EXAMPLE
    .\start-local.ps1
    .\start-local.ps1 -Logs
#>

[CmdletBinding()]
param(
    [switch]$Logs
)

$ErrorActionPreference = "Stop"

$StableCompose = "docker-compose.stable.yaml"
$LocalCompose  = "docker-compose.local.yaml"
$AppUrl        = "http://localhost:8000"

function Write-Step {
    param([string]$Message)
    Write-Host "[start-local] $Message" -ForegroundColor Cyan
}

function Write-Ok {
    param([string]$Message)
    Write-Host "[start-local] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[start-local] $Message" -ForegroundColor Yellow
}

function Test-Command {
    param([string]$Name)

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        Write-Host "[start-local] ERROR: '$Name' was not found in PATH." -ForegroundColor Red
        Write-Host "[start-local] Install Docker Desktop and make sure the docker CLI is available." -ForegroundColor Red
        exit 1
    }
    return $cmd
}

Write-Step "Checking Docker availability..."
Test-Command "docker" | Out-Null

try {
    $dockerVersion = (docker version --format '{{.Server.Version}}' 2>$null)
    if (-not $dockerVersion) {
        throw "no server version"
    }
    Write-Ok "Docker engine is running (server $dockerVersion)."
}
catch {
    Write-Host "[start-local] ERROR: Docker engine is not running or is not reachable." -ForegroundColor Red
    Write-Host "[start-local] Start Docker Desktop, wait until it reports 'Running', then retry." -ForegroundColor Red
    exit 1
}

Write-Step "Checking Docker Compose plugin..."
$composeArgs = @("compose", "version")
try {
    & docker @composeArgs | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "compose failed" }
    Write-Ok "Docker Compose is available."
}
catch {
    Write-Host "[start-local] ERROR: 'docker compose' is not available." -ForegroundColor Red
    Write-Host "[start-local] Update Docker Desktop, or install the Compose v2 plugin." -ForegroundColor Red
    exit 1
}

Push-Location $PSScriptRoot
try {
    foreach ($file in @($StableCompose, $LocalCompose)) {
        if (-not (Test-Path $file)) {
            Write-Host "[start-local] ERROR: '$file' not found in $PSScriptRoot" -ForegroundColor Red
            exit 1
        }
    }

    if ([string]::IsNullOrWhiteSpace($env:SECRET) -or $env:SECRET.Length -lt 32) {
        Write-Host "[start-local] ERROR: Set `$env:SECRET to a random value of at least 32 characters before starting." -ForegroundColor Red
        Write-Host "[start-local] Example: `$env:SECRET = -join ((1..48) | ForEach-Object { [char](Get-Random -Minimum 33 -Maximum 127) })" -ForegroundColor Yellow
        exit 1
    }

    Write-Step "Building CoAI from source and starting services (this takes a few minutes on first run)..."
    & docker compose -f $StableCompose -f $LocalCompose up -d --build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[start-local] ERROR: docker compose up failed with exit code $LASTEXITCODE." -ForegroundColor Red
        exit $LASTEXITCODE
    }

    Write-Host ""
    Write-Ok "CoAI is starting."
    Write-Host ""
    Write-Host "  CoAI:  $AppUrl" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Services: chatnio (8000 -> 8094), mysql (3306, internal), redis (6379, internal)" -ForegroundColor Gray
    Write-Host ""

    Write-Step "Checking whether a local Sub2API is reachable on the Windows host..."
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        Write-Ok "Sub2API responded with HTTP $($response.StatusCode)."
    }
    catch {
        Write-Warn "No Sub2API health response on http://localhost:8080/health."
        Write-Warn "Start Sub2API first if CoAI should talk to it."
    }

    Write-Host ""
    Write-Host "  Next step - create a channel in the CoAI admin panel:" -ForegroundColor Cyan
    Write-Host "    Type:     OpenAI" -ForegroundColor White
    Write-Host "    Endpoint: http://host.docker.internal:8080" -ForegroundColor White
    Write-Host "              (no trailing slash, do NOT append /v1)" -ForegroundColor Gray
    Write-Host "    Secret:   your Sub2API API key (server side only)" -ForegroundColor White
    Write-Host "    Models:   gpt-5 and gpt-image-2 (or whatever Sub2API exposes)" -ForegroundColor White
    Write-Host ""
    Write-Host "  Run '.\stop-local.ps1' to stop everything." -ForegroundColor Gray
    Write-Host ""

    if ($Logs) {
        Write-Step "Following logs (Ctrl+C to stop)..."
        & docker compose -f $StableCompose -f $LocalCompose logs -f
    }
}
finally {
    Pop-Location
}
