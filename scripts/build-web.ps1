# Build React and copy into Go embed directory
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\frontend
npm install
npm run build
Set-Location $PSScriptRoot
if (Test-Path web) { Remove-Item -Recurse -Force web }
New-Item -ItemType Directory -Path web | Out-Null
Copy-Item -Recurse frontend\dist\* web\
Write-Host "Frontend copied to web/ — run: go build -o poker-chip-tracker.exe ."
