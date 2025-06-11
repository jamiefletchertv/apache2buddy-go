package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"apache2buddy-go/internal/analysis"
	"apache2buddy-go/internal/config"
	"apache2buddy-go/internal/logs"
	"apache2buddy-go/internal/status"
	"apache2buddy-go/internal/system"
)

// captureOutput captures stdout during function execution for testing
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestDisplayEnhancedResults_BasicOutput(t *testing.T) {
	// Prepare test data
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     2048,
		AvailableMemoryMB: 1500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		SmallestMB:   20.5,
		LargestMB:    35.2,
		AverageMB:    27.8,
		TotalMB:      278.0,
		ProcessCount: 10,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 150,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
		ConfigPath:        "/etc/apache2/apache2.conf",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     150,
		RecommendedMaxClients: 120,
		Status:                "WARNING",
		Message:               "Consider reducing MaxRequestWorkers",
		UtilizationPercent:    85.0,
	}

	statusInfo := &status.ApacheStatus{
		ActiveWorkers: 8,
		IdleWorkers:   12,
	}

	logAnalysis := &logs.LogAnalysis{
		MaxClientsExceeded: 2,
		PHPFatalErrors:     1,
		AnalyzedLines:      1000,
	}

	// Capture output
	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, statusInfo, logAnalysis)
	})

	// Verify key sections are present
	tests := []string{
		"Server Version: Apache 2.4.41",
		"Server MPM: prefork",
		"Total RAM: 2048 MB",
		"Available RAM: 1500 MB",
		"Current MaxRequestWorkers: 150",
		"Apache processes found: 10",
		"Memory usage per process: 20.5 MB (smallest), 27.8 MB (average), 35.2 MB (largest)",
		"Active workers: 8, Idle workers: 12",
		"⚠️  RESULT: Your Apache configuration could be improved.",
		"Consider reducing MaxRequestWorkers to 120",
		"Configuration file: /etc/apache2/apache2.conf",
		"MaxRequestWorkers 120",
		"⚠️  Log analysis shows MaxRequestWorkers was exceeded 2 times",
		"⚠️  Found 1 PHP Fatal Errors in logs",
		"Analysis completed. Check /var/log/apache2buddy-go.log for historical data.",
	}

	for _, expected := range tests {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestDisplayEnhancedResults_OKStatus(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     4096,
		AvailableMemoryMB: 3500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		SmallestMB:   25.0,
		LargestMB:    40.0,
		AverageMB:    32.5,
		TotalMB:      325.0,
		ProcessCount: 10,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 80,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
		ConfigPath:        "/etc/apache2/apache2.conf",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     80,
		RecommendedMaxClients: 87,
		Status:                "OK",
		Message:               "Configuration appears acceptable",
		UtilizationPercent:    35.0,
	}

	statusInfo := &status.ApacheStatus{
		ActiveWorkers: 5,
		IdleWorkers:   10,
	}

	logAnalysis := &logs.LogAnalysis{
		AnalyzedLines: 500,
	}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, statusInfo, logAnalysis)
	})

	// Should show OK status and no configuration changes needed
	if !strings.Contains(output, "✓ RESULT: Your Apache configuration appears to be optimal.") {
		t.Error("Output should show OK result for optimal configuration")
	}

	// Should not show MaxRequestWorkers configuration section when OK
	configSectionCount := strings.Count(output, "To implement changes, edit your Apache configuration:")
	if configSectionCount > 0 {
		t.Error("Should not show configuration changes for OK status")
	}
}

func TestDisplayEnhancedResults_CriticalStatus(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     1024,
		AvailableMemoryMB: 800,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		SmallestMB:   45.0,
		LargestMB:    60.0,
		AverageMB:    52.5,
		TotalMB:      525.0,
		ProcessCount: 10,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 256,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
		ConfigPath:        "/etc/apache2/apache2.conf",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     256,
		RecommendedMaxClients: 13,
		Status:                "CRITICAL",
		Message:               "Configuration needs immediate attention",
		UtilizationPercent:    150.0,
	}

	logAnalysis := &logs.LogAnalysis{
		MaxClientsExceeded: 10,
		PHPFatalErrors:     5,
		AnalyzedLines:      2000,
	}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, nil, logAnalysis)
	})

	// Should show critical status
	if !strings.Contains(output, "🔥 RESULT: Your Apache configuration needs immediate attention!") {
		t.Error("Output should show CRITICAL result")
	}

	// Should show configuration changes
	if !strings.Contains(output, "Reduce MaxRequestWorkers to 13") {
		t.Error("Output should show specific reduction recommendation")
	}

	// Should show log analysis warnings
	if !strings.Contains(output, "MaxRequestWorkers was exceeded 10 times") {
		t.Error("Output should show MaxClients exceeded warnings")
	}
	if !strings.Contains(output, "Found 5 PHP Fatal Errors") {
		t.Error("Output should show PHP fatal error count")
	}
}

func TestDisplayEnhancedResults_WithOtherServices(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     2048,
		AvailableMemoryMB: 1200,
		OtherServices: map[string]int{
			"MySQL":   200,
			"Redis":   50,
			"PHP-FPM": 100,
		},
	}

	memStats := &analysis.MemoryStats{
		ProcessCount: 8,
		LargestMB:    30.0,
		AverageMB:    25.0,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 100,
		MPMModel:          "worker",
		ServerName:        "Apache",
		Version:           "2.4.41",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:  100,
		Status:             "WARNING",
		MPMNote:            "Apache is running in worker mode. Check manually for backend processes such as PHP-FPM and pm.max_children.",
	}

	logAnalysis := &logs.LogAnalysis{}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, nil, logAnalysis)
	})

	// Should show other services memory usage
	if !strings.Contains(output, "RAM used by other services: 350 MB") {
		t.Error("Output should show other services memory usage")
	}
	if !strings.Contains(output, "RAM available for Apache: 1200 MB") {
		t.Error("Output should show RAM available for Apache")
	}

	// Should show MPM-specific note
	if !strings.Contains(output, "Apache is running in worker mode") {
		t.Error("Output should show MPM-specific note")
	}
}

func TestDisplayEnhancedResults_WithExtendedStatus(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     2048,
		AvailableMemoryMB: 1500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		ProcessCount: 10,
		LargestMB:    30.0,
		AverageMB:    25.0,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 100,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients: 100,
		Status:            "OK",
	}

	statusInfo := &status.ApacheStatus{
		ActiveWorkers:     8,
		IdleWorkers:       12,
		RequestsPerSec:    2.5,
		ExtendedEnabled:   true,
		Load1Min:          0.8,
		Load5Min:          0.6,
		Load15Min:         0.4,
	}

	logAnalysis := &logs.LogAnalysis{}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, statusInfo, logAnalysis)
	})

	// Should show extended status information
	if !strings.Contains(output, "Requests per second: 2.500") {
		t.Error("Output should show requests per second")
	}
	if !strings.Contains(output, "System load: 0.80 (1min), 0.60 (5min), 0.40 (15min)") {
		t.Error("Output should show system load averages")
	}
}

func TestDetectServerBuilt(t *testing.T) {
	// Test that function doesn't crash and returns a string
	built := detectServerBuilt()
	
	if built == "" {
		t.Error("detectServerBuilt should return a non-empty string")
	}
	
	// In test environment, should return "Unknown"
	if built != "Unknown" {
		t.Logf("Detected server built: %s", built)
	}
}

func TestDisplayEnhancedResults_NoLogAnalysis(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     2048,
		AvailableMemoryMB: 1500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		ProcessCount: 5,
		LargestMB:    25.0,
		AverageMB:    22.0,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 80,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients: 80,
		Status:            "OK",
	}

	// Test with nil statusInfo and empty logAnalysis
	logAnalysis := &logs.LogAnalysis{
		AnalyzedLines: 0,
	}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, nil, logAnalysis)
	})

	// Should not show log analysis section when no lines analyzed
	logAnalysisCount := strings.Count(output, "Log analysis shows")
	if logAnalysisCount > 0 {
		t.Error("Should not show log analysis when no lines were analyzed")
	}
}

func TestDisplayEnhancedResults_ServerLimitConfiguration(t *testing.T) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     4096,
		AvailableMemoryMB: 3500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		ProcessCount: 10,
		LargestMB:    20.0,
		AverageMB:    18.0,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 150,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     150,
		RecommendedMaxClients: 300, // Higher than 256, should trigger ServerLimit
		Status:                "WARNING",
	}

	logAnalysis := &logs.LogAnalysis{}

	output := captureOutput(func() {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, nil, logAnalysis)
	})

	// Should show ServerLimit configuration when recommended > 256 for prefork
	if !strings.Contains(output, "ServerLimit 300") {
		t.Error("Output should show ServerLimit configuration for high MaxRequestWorkers in prefork mode")
	}
}

// Benchmark test for performance validation
func BenchmarkDisplayEnhancedResults(b *testing.B) {
	sysInfo := &system.SystemInfo{
		TotalMemoryMB:     2048,
		AvailableMemoryMB: 1500,
		OtherServices:     make(map[string]int),
	}

	memStats := &analysis.MemoryStats{
		SmallestMB:   20.5,
		LargestMB:    35.2,
		AverageMB:    27.8,
		TotalMB:      278.0,
		ProcessCount: 10,
	}

	config := &config.ApacheConfig{
		MaxRequestWorkers: 150,
		MPMModel:          "prefork",
		ServerName:        "Apache",
		Version:           "2.4.41",
		ConfigPath:        "/etc/apache2/apache2.conf",
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     150,
		RecommendedMaxClients: 120,
		Status:                "WARNING",
		Message:               "Consider reducing MaxRequestWorkers",
	}

	statusInfo := &status.ApacheStatus{
		ActiveWorkers: 8,
		IdleWorkers:   12,
	}

	logAnalysis := &logs.LogAnalysis{
		AnalyzedLines: 1000,
	}

	// Redirect output to discard for benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DisplayEnhancedResults(sysInfo, memStats, config, recommendations, statusInfo, logAnalysis)
	}
}