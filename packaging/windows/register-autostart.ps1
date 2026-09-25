$ErrorActionPreference = "Stop"

$AppDir = $PSScriptRoot
$Exe = Join-Path $AppDir "super-picos-hidden.exe"
$EnvFile = Join-Path $AppDir ".env"
$TaskName = "Super Picos Downloader"

if (-not (Test-Path $Exe)) {
    throw "Executável não encontrado: $Exe"
}

if (-not (Test-Path $EnvFile)) {
    throw "Crie o arquivo .env em $AppDir antes de registrar o autostart."
}

$Action = New-ScheduledTaskAction `
    -Execute $Exe `
    -WorkingDirectory $AppDir

$Trigger = New-ScheduledTaskTrigger -AtLogOn

$Settings = New-ScheduledTaskSettingsSet `
    -RestartCount 5 `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -StartWhenAvailable

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $Action `
    -Trigger $Trigger `
    -Settings $Settings `
    -Description "Inicia automaticamente o Super Picos Downloader" `
    -Force | Out-Null

Start-ScheduledTask -TaskName $TaskName
Start-Sleep -Seconds 2

Write-Host "Tarefa registrada: $TaskName"
Write-Host "Diagnóstico: Get-ScheduledTaskInfo -TaskName '$TaskName'"
Write-Host "Parar:       Stop-ScheduledTask -TaskName '$TaskName'"
