param(
    [Parameter(Mandatory = $true)]
    [string]$BatchPath
)

$ErrorActionPreference = "Stop"

$resolved = Resolve-Path -Path $BatchPath
$batch = $resolved.ProviderPath

$script:cancelled = $false
$script:proc = $null

$handler = [ConsoleCancelEventHandler]{
    param($sender, $eventArgs)
    $eventArgs.Cancel = $true
    $script:cancelled = $true
    if ($script:proc -and -not $script:proc.HasExited) {
        try {
            & taskkill /T /F /PID $script:proc.Id | Out-Null
        } catch {
        }
    }
}

[Console]::add_CancelKeyPress($handler)

try {
    $script:proc = Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "`"$batch`"" -NoNewWindow -PassThru
    $script:proc.WaitForExit()
    if ($script:cancelled) {
        exit 130
    }
    exit $script:proc.ExitCode
} finally {
    [Console]::remove_CancelKeyPress($handler)
}
