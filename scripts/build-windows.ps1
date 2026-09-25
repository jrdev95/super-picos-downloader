$ErrorActionPreference = "Stop"

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$Out = Join-Path $Root "dist\windows-amd64"
$Archive = Join-Path $Root "dist\super-picos-windows-amd64.zip"

if (Test-Path $Out) {
    Remove-Item $Out -Recurse -Force
}

New-Item -ItemType Directory -Force -Path (Join-Path $Out "secrets") | Out-Null

Push-Location $Root
try {
    Write-Host "==> Executando testes"
    go test ./...
    go vet ./...

    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    Write-Host "==> Compilando Windows amd64"
    go build -trimpath -ldflags="-s -w" `
        -o (Join-Path $Out "super-picos.exe") .\cmd\bot

    go build -trimpath -ldflags="-s -w -H=windowsgui" `
        -o (Join-Path $Out "super-picos-hidden.exe") .\cmd\bot

    Copy-Item .env.example, README.md -Destination $Out
    Copy-Item packaging\windows\register-autostart.ps1 -Destination $Out

    @"
Coloque aqui o arquivo instagram-cookies.txt, se necessário.
Nunca distribua cookies reais dentro de um release.
"@ | Set-Content -Encoding UTF8 (Join-Path $Out "secrets\README.txt")

    if (Test-Path $Archive) {
        Remove-Item $Archive -Force
    }

    Compress-Archive -Path (Join-Path $Out "*") -DestinationPath $Archive
    Write-Host "==> Release criado: $Archive"
}
finally {
    Pop-Location
}
