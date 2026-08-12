#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"
Set-Location "$PSScriptRoot/.."

Push-Location web
npm install
npm run build
Pop-Location

go build -o tbg-rse.exe ./cmd/tbg-rse
Write-Host "Built tbg-rse.exe"
