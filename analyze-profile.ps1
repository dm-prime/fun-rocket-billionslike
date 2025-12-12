# Analyze a performance profile captured by the game
# Usage: .\analyze-profile.ps1 [profile-file]

param(
    [string]$ProfileFile = "",
    [int]$Port = 8080
)

# If no profile file specified, find the most recent one
if ([string]::IsNullOrEmpty($ProfileFile)) {
    $profilesDir = "profiles"
    if (-not (Test-Path $profilesDir)) {
        Write-Host "ERROR: Profiles directory not found. No profiles have been captured yet." -ForegroundColor Red
        exit 1
    }
    
    # Find most recent CPU profile
    $latestProfile = Get-ChildItem -Path $profilesDir -Filter "*.cpu.prof" | 
        Sort-Object LastWriteTime -Descending | 
        Select-Object -First 1
    
    if ($null -eq $latestProfile) {
        Write-Host "ERROR: No CPU profiles found in $profilesDir" -ForegroundColor Red
        exit 1
    }
    
    $ProfileFile = $latestProfile.FullName
    Write-Host "Using most recent profile: $($latestProfile.Name)" -ForegroundColor Cyan
}

if (-not (Test-Path $ProfileFile)) {
    Write-Host "ERROR: Profile file not found: $ProfileFile" -ForegroundColor Red
    exit 1
}

Write-Host "Analyzing profile: $ProfileFile" -ForegroundColor Cyan
Write-Host "Starting pprof web server on http://localhost:$Port..." -ForegroundColor Cyan
Write-Host "Flame graph will be available at: http://localhost:$Port/ui/flamegraph" -ForegroundColor Yellow

# Cleanup function
function Cleanup {
    if ($null -ne $pprofProcess -and -not $pprofProcess.HasExited) {
        Stop-Process -Id $pprofProcess.Id -Force -ErrorAction SilentlyContinue
    }
}

# Set up cleanup handlers
Register-EngineEvent PowerShell.Exiting -Action { Cleanup } | Out-Null
trap { Cleanup; break }

# Start pprof server
$pprofProcess = Start-Process -FilePath "go" -ArgumentList "tool", "pprof", "-http=:$Port", $ProfileFile -PassThru -NoNewWindow

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
    Cleanup
}



