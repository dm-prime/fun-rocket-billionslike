# Trace script for billionslike3
# Collects execution trace and opens timeline view

param(
    [int]$Seconds = 5
)

Write-Host "Collecting execution trace for $Seconds seconds from http://localhost:6060..." -ForegroundColor Cyan

# Download the trace
$traceFile = "trace.out"
curl -s "http://localhost:6060/debug/pprof/trace?seconds=$Seconds" -o $traceFile

if (-not (Test-Path $traceFile)) {
    Write-Host "ERROR: Failed to download trace. Is the game running?" -ForegroundColor Red
    exit 1
}

Write-Host "Trace collected: $traceFile" -ForegroundColor Green
Write-Host "Opening trace viewer (will open browser automatically)..." -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop the trace viewer." -ForegroundColor Yellow

# Cleanup function
function Cleanup-Trace {
    param([string]$file)
    if (Test-Path $file) {
        Remove-Item $file -Force -ErrorAction SilentlyContinue
        Write-Host "`nCleaned up $file" -ForegroundColor Yellow
    }
}

# Set up cleanup handlers
$cleanupAction = { Cleanup-Trace -file $traceFile }
Register-EngineEvent PowerShell.Exiting -Action $cleanupAction | Out-Null
trap { Cleanup-Trace -file $traceFile; break }

# Open trace viewer (this will open a browser automatically and start a web server)
try {
    $traceProcess = Start-Process -FilePath "go" -ArgumentList "tool", "trace", $traceFile -PassThru -NoNewWindow
    
    Write-Host "Trace viewer is running. Browser should open automatically." -ForegroundColor Green
    
    # Wait for trace process to exit (or user presses Ctrl+C)
    Wait-Process -Id $traceProcess.Id -ErrorAction SilentlyContinue
} catch {
    Write-Host "ERROR: Failed to open trace viewer. Make sure Go is installed." -ForegroundColor Red
    Cleanup-Trace -file $traceFile
    exit 1
} finally {
    # Ensure process is stopped
    if (-not $traceProcess.HasExited) {
        Stop-Process -Id $traceProcess.Id -Force -ErrorAction SilentlyContinue
    }
    
    # Cleanup trace file
    Cleanup-Trace -file $traceFile
}

