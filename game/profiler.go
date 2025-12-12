package game

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// Profiler handles automatic performance profiling
type Profiler struct {
	mu                sync.Mutex
	isProfiling       bool
	lastCaptureTime   time.Time
	captureCooldown   time.Duration
	profilesDir       string
	captureDuration   time.Duration
	
	// Continuous CPU profiling
	cpuProfileFile    *os.File
	cpuProfileActive  bool
	cpuProfileStartTime time.Time
}

// NewProfiler creates a new profiler instance
// Continuous CPU profiling is disabled by default to avoid performance overhead
func NewProfiler() *Profiler {
	profilesDir := "profiles"
	os.MkdirAll(profilesDir, 0755)
	
	p := &Profiler{
		captureCooldown: 10 * time.Second, // Don't capture more than once every 10 seconds
		profilesDir:     profilesDir,
		captureDuration: 5 * time.Second,   // Capture 5 seconds of data
		cpuProfileActive: false,
	}
	
	// Disabled continuous CPU profiling to reduce overhead
	// Uncomment the line below to enable profiling:
	// p.StartContinuousCPUProfile()
	
	return p
}

// StartContinuousCPUProfile starts continuous CPU profiling in the background
// This avoids the stop-the-world pause that happens when starting profiling on-demand
func (p *Profiler) StartContinuousCPUProfile() error {
	p.mu.Lock()
	if p.cpuProfileActive {
		p.mu.Unlock()
		return fmt.Errorf("CPU profiling already active")
	}
	
	// Create a temporary file for continuous profiling
	// We'll save it when FPS drops are detected
	tempFile, err := os.CreateTemp(p.profilesDir, "cpu-profile-temp-*.prof")
	if err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to create temp profile file: %w", err)
	}
	
	// Start CPU profiling
	if err := pprof.StartCPUProfile(tempFile); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		p.mu.Unlock()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	
	p.cpuProfileFile = tempFile
	p.cpuProfileActive = true
	p.cpuProfileStartTime = time.Now()
	p.mu.Unlock()
	
	return nil
}

// StopContinuousCPUProfile stops continuous CPU profiling and saves the profile
func (p *Profiler) StopContinuousCPUProfile(reason string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.cpuProfileActive {
		return "", fmt.Errorf("CPU profiling not active")
	}
	
	// Stop profiling
	pprof.StopCPUProfile()
	
	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("fps-drop-%s-%s", timestamp, reason)
	profilePath := filepath.Join(p.profilesDir, baseName+".cpu.prof")
	
	// Close temp file
	tempPath := p.cpuProfileFile.Name()
	p.cpuProfileFile.Close()
	
	// Rename temp file to final location
	if err := os.Rename(tempPath, profilePath); err != nil {
		// If rename fails, try copying
		if copyErr := p.copyFile(tempPath, profilePath); copyErr != nil {
			os.Remove(tempPath)
			return "", fmt.Errorf("failed to save profile: %w", err)
		}
		os.Remove(tempPath)
	}
	
	p.cpuProfileActive = false
	
	// Restart continuous profiling
	go func() {
		if err := p.StartContinuousCPUProfile(); err != nil {
			fmt.Printf("Warning: Failed to restart CPU profiling: %v\n", err)
		}
	}()
	
	return profilePath, nil
}

// copyFile copies a file from src to dst
func (p *Profiler) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// CaptureProfile captures CPU profile and trace when FPS drops
func (p *Profiler) CaptureProfile(reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check cooldown to avoid capturing too frequently
	if time.Since(p.lastCaptureTime) < p.captureCooldown {
		return fmt.Errorf("capture on cooldown (last capture was %v ago)", time.Since(p.lastCaptureTime))
	}
	
	if p.isProfiling {
		return fmt.Errorf("already profiling")
	}
	
	p.isProfiling = true
	p.lastCaptureTime = time.Now()
	
	// Generate timestamped filename
	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("fps-drop-%s-%s", timestamp, reason)
	
	// Capture in a goroutine to avoid blocking the game
	go func() {
		defer func() {
			p.mu.Lock()
			p.isProfiling = false
			p.mu.Unlock()
		}()
		
		// Capture CPU profile and trace in parallel
		var wg sync.WaitGroup
		wg.Add(2)
		
		go func() {
			defer wg.Done()
			err := p.captureCPUProfile(baseName)
			if err != nil {
				fmt.Printf("Error capturing CPU profile: %v\n", err)
			}
		}()
		
		go func() {
			defer wg.Done()
			err := p.captureTrace(baseName)
			if err != nil {
				fmt.Printf("Error capturing trace: %v\n", err)
			}
		}()
		
		// Wait for both captures to complete
		wg.Wait()
		
		// Analyze the profile
		p.analyzeProfile(baseName)
	}()
	
	return nil
}

// captureCPUProfile captures a CPU profile
func (p *Profiler) captureCPUProfile(baseName string) error {
	profilePath := filepath.Join(p.profilesDir, baseName+".cpu.prof")
	
	file, err := os.Create(profilePath)
	if err != nil {
		return fmt.Errorf("failed to create profile file: %w", err)
	}
	defer file.Close()
	
	// Start CPU profiling
	if err := pprof.StartCPUProfile(file); err != nil {
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	
	// Profile for the specified duration
	time.Sleep(p.captureDuration)
	
	// Stop profiling
	pprof.StopCPUProfile()
	
	fmt.Printf("CPU profile saved to: %s\n", profilePath)
	return nil
}

// captureTrace captures an execution trace
func (p *Profiler) captureTrace(baseName string) error {
	tracePath := filepath.Join(p.profilesDir, baseName+".trace")
	
	file, err := os.Create(tracePath)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	defer file.Close()
	
	// Start tracing
	if err := trace.Start(file); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}
	
	// Trace for the specified duration
	time.Sleep(p.captureDuration)
	
	// Stop tracing
	trace.Stop()
	
	fmt.Printf("Trace saved to: %s\n", tracePath)
	return nil
}

// analyzeProfile analyzes the captured profile and prints a summary
func (p *Profiler) analyzeProfile(baseName string) {
	profilePath := filepath.Join(p.profilesDir, baseName+".cpu.prof")
	
	// Check if file exists and get its size
	info, err := os.Stat(profilePath)
	if err != nil {
		fmt.Printf("Warning: Could not analyze profile: %v\n", err)
		return
	}
	
	fmt.Printf("\n=== Performance Analysis: %s ===\n", baseName)
	fmt.Printf("Profile file: %s (%.2f KB)\n", profilePath, float64(info.Size())/1024)
	fmt.Printf("\nTo view detailed analysis, run:\n")
	fmt.Printf("  go tool pprof -http=:8080 %s\n", profilePath)
	fmt.Printf("  (Flame graph will be available at http://localhost:8080/ui/flamegraph)\n")
	
	// Get memory stats at analysis time
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\nMemory stats at capture time:\n")
	fmt.Printf("  Alloc: %d KB\n", m.Alloc/1024)
	fmt.Printf("  TotalAlloc: %d KB\n", m.TotalAlloc/1024)
	fmt.Printf("  Sys: %d KB\n", m.Sys/1024)
	fmt.Printf("  NumGC: %d\n", m.NumGC)
	fmt.Printf("  HeapObjects: %d\n", m.HeapObjects)
	
	// GC pause statistics
	if m.NumGC > 0 {
		avgPauseNs := m.PauseTotalNs / uint64(m.NumGC)
		fmt.Printf("  GC Pause Total: %.2f ms\n", float64(m.PauseTotalNs)/1e6)
		fmt.Printf("  GC Pause Avg: %.2f ms\n", float64(avgPauseNs)/1e6)
		if m.NumGC < 256 {
			fmt.Printf("  GC Pause Max: %.2f ms\n", float64(m.PauseNs[m.NumGC-1])/1e6)
		} else {
			fmt.Printf("  GC Pause Max: %.2f ms\n", float64(m.PauseNs[(m.NumGC+255)%256])/1e6)
		}
	}
	
	fmt.Printf("\n=== End Analysis ===\n\n")
}

// CaptureProfileSync saves the current continuous CPU profile when FPS drop is detected
// This avoids stop-the-world pauses by using already-running profiling
func (p *Profiler) CaptureProfileSync(reason string, duration time.Duration) error {
	// Save the current continuous CPU profile (captures data from before the drop)
	profilePath, err := p.StopContinuousCPUProfile(reason)
	if err != nil {
		return fmt.Errorf("failed to save CPU profile: %w", err)
	}
	
	// Extract base name for analysis
	baseName := filepath.Base(profilePath)
	baseName = baseName[:len(baseName)-len(".cpu.prof")]
	
	// Analyze the profile
	p.analyzeProfile(baseName)
	
	fmt.Printf("CPU profile saved to: %s\n", profilePath)
	fmt.Printf("Note: Trace capture skipped to avoid stop-the-world pause\n")
	
	return nil
}


// IsProfiling returns whether a profile capture is currently in progress
func (p *Profiler) IsProfiling() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isProfiling
}

