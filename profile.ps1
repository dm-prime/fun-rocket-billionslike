# Profile script for billionslike3
# Collects CPU profile and serves it with pprof

param(
    [int]$Seconds = 5,
    [int]$Port = 8080
)

Write-Host "Collecting CPU profile for $Seconds seconds from http://localhost:6060..." -ForegroundColor Cyan

# Download the profile
$profileFile = "cpu.prof"
curl -s "http://localhost:6060/debug/pprof/profile?seconds=$Seconds" -o $profileFile

if (-not (Test-Path $profileFile)) {
    Write-Host "ERROR: Failed to download profile. Is the game running?" -ForegroundColor Red
    exit 1
}

Write-Host "Profile collected: $profileFile" -ForegroundColor Green
Write-Host "Starting pprof web server on http://localhost:$Port..." -ForegroundColor Cyan
Write-Host "Flame graph will be available at: http://localhost:$Port/ui/flamegraph" -ForegroundColor Yellow

# Cleanup function
function Cleanup-Profile {
    param([string]$file)
    if (Test-Path $file) {
        Remove-Item $file -Force -ErrorAction SilentlyContinue
        Write-Host "`nCleaned up $file" -ForegroundColor Yellow
    }
}

# Set up cleanup handlers
$cleanupAction = { Cleanup-Profile -file $profileFile }
Register-EngineEvent PowerShell.Exiting -Action $cleanupAction | Out-Null
trap { Cleanup-Profile -file $profileFile; break }

# Start pprof server
$pprofProcess = Start-Process -FilePath "go" -ArgumentList "tool", "pprof", "-http=:$Port", $profileFile -PassThru -NoNewWindow

# Wait a moment for server to start
Start-Sleep -Seconds 2

# Open browser to flame graph
Start-Process "http://localhost:$Port/ui/flamegraph"

Write-Host "`nProfile server is running. Press Ctrl+C to stop." -ForegroundColor Green

try {
    # Wait for pprof process to exit (or user presses Ctrl+C)
    Wait-Process -Id $pprofProcess.Id -ErrorAction SilentlyContinue
} catch {
    # User interrupted or process ended
} finally {
    # Ensure process is stopped
    if (-not $pprofProcess.HasExited) {
        Stop-Process -Id $pprofProcess.Id -Force -ErrorAction SilentlyContinue
    }
    
    # Cleanup profile file
    Cleanup-Profile -file $profileFile
}

