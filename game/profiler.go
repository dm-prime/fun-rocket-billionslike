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
}

// NewProfiler creates a new profiler instance
func NewProfiler() *Profiler {
	profilesDir := "profiles"
	os.MkdirAll(profilesDir, 0755)
	
	return &Profiler{
		captureCooldown: 10 * time.Second, // Don't capture more than once every 10 seconds
		profilesDir:     profilesDir,
		captureDuration: 5 * time.Second,   // Capture 5 seconds of data
	}
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
	fmt.Printf("\n=== End Analysis ===\n\n")
}

// CaptureProfileSync captures CPU profile and trace synchronously (blocks until complete)
// This is used when the game is about to exit, so we can ensure data is written
func (p *Profiler) CaptureProfileSync(reason string, duration time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Generate timestamped filename
	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("fps-drop-%s-%s", timestamp, reason)
	
	// Capture CPU profile and trace in parallel
	var wg sync.WaitGroup
	var cpuErr, traceErr error
	wg.Add(2)
	
	go func() {
		defer wg.Done()
		cpuErr = p.captureCPUProfileSync(baseName, duration)
	}()
	
	go func() {
		defer wg.Done()
		traceErr = p.captureTraceSync(baseName, duration)
	}()
	
	// Wait for both captures to complete
	wg.Wait()
	
	// Analyze the profile
	p.analyzeProfile(baseName)
	
	if cpuErr != nil {
		return cpuErr
	}
	if traceErr != nil {
		return traceErr
	}
	return nil
}

// captureCPUProfileSync captures a CPU profile synchronously
func (p *Profiler) captureCPUProfileSync(baseName string, duration time.Duration) error {
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
	time.Sleep(duration)
	
	// Stop profiling
	pprof.StopCPUProfile()
	
	fmt.Printf("CPU profile saved to: %s\n", profilePath)
	return nil
}

// captureTraceSync captures an execution trace synchronously
func (p *Profiler) captureTraceSync(baseName string, duration time.Duration) error {
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
	time.Sleep(duration)
	
	// Stop tracing
	trace.Stop()
	
	fmt.Printf("Trace saved to: %s\n", tracePath)
	return nil
}

// IsProfiling returns whether a profile capture is currently in progress
func (p *Profiler) IsProfiling() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isProfiling
}

