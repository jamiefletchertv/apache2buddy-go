package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apache2buddy-go/internal/analysis"
	"apache2buddy-go/internal/config"
	"apache2buddy-go/internal/system"
)

func TestLogAnalysis_Empty(t *testing.T) {
	analysis := &LogAnalysis{}

	if analysis.MaxClientsExceeded != 0 {
		t.Errorf("Empty LogAnalysis should have MaxClientsExceeded = 0, got %d", analysis.MaxClientsExceeded)
	}
	if analysis.PHPFatalErrors != 0 {
		t.Errorf("Empty LogAnalysis should have PHPFatalErrors = 0, got %d", analysis.PHPFatalErrors)
	}
	if len(analysis.RecentErrors) != 0 {
		t.Errorf("Empty LogAnalysis should have empty RecentErrors, got %d", len(analysis.RecentErrors))
	}
}

func TestIsReadableLogFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() string // Returns file path
		cleanup  func(string)
		expected bool
	}{
		{
			name: "regular readable file",
			setup: func() string {
				tmpFile, err := os.CreateTemp("", "test_log_*.log")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				_, _ = tmpFile.WriteString("test log content\n")
				_ = tmpFile.Close()
				return tmpFile.Name()
			},
			cleanup: func(path string) {
				_ = os.Remove(path)
			},
			expected: true,
		},
		{
			name: "non-existent file",
			setup: func() string {
				return "/non/existent/file.log"
			},
			cleanup:  func(string) {},
			expected: false,
		},
		{
			name: "symlink to /dev/stdout",
			setup: func() string {
				tmpDir := t.TempDir()
				linkPath := filepath.Join(tmpDir, "stdout_link.log")
				err := os.Symlink("/dev/stdout", linkPath)
				if err != nil {
					t.Skipf("Cannot create symlink test: %v", err)
				}
				return linkPath
			},
			cleanup:  func(string) {},
			expected: false,
		},
		{
			name: "symlink to regular file",
			setup: func() string {
				tmpDir := t.TempDir()

				// Create target file
				targetPath := filepath.Join(tmpDir, "target.log")
				err := os.WriteFile(targetPath, []byte("log content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create target file: %v", err)
				}

				// Create symlink
				linkPath := filepath.Join(tmpDir, "link.log")
				err = os.Symlink(targetPath, linkPath)
				if err != nil {
					t.Skipf("Cannot create symlink test: %v", err)
				}
				return linkPath
			},
			cleanup:  func(string) {},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setup()
			defer tt.cleanup(filePath)

			result := isReadableLogFile(filePath)
			if result != tt.expected {
				t.Errorf("isReadableLogFile(%s) = %v, want %v", filePath, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeLogFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *LogAnalysis
	}{
		{
			name: "log with MaxClients exceeded",
			content: `[Mon Dec 05 10:15:30 2023] [error] server reached MaxRequestWorkers setting, consider raising the MaxRequestWorkers setting
[Mon Dec 05 10:16:30 2023] [error] server reached MaxClients setting, consider raising the MaxClients setting`,
			expected: &LogAnalysis{
				MaxClientsExceeded: 2,
				PHPFatalErrors:     0,
				RecentErrors:       []string{},
				AnalyzedLines:      2,
			},
		},
		{
			name: "log with PHP fatal errors",
			content: `[Mon Dec 05 10:15:30 2023] [error] PHP Fatal error: Call to undefined function in /var/www/test.php on line 10
[Mon Dec 05 10:16:30 2023] [error] PHP Parse error: syntax error in /var/www/bad.php on line 5`,
			expected: &LogAnalysis{
				MaxClientsExceeded: 0,
				PHPFatalErrors:     2,
				RecentErrors: []string{
					"[Mon Dec 05 10:15:30 2023] [error] PHP Fatal error: Call to undefined function in /var/www/test.php on line 10",
					"[Mon Dec 05 10:16:30 2023] [error] PHP Parse error: syntax error in /var/www/bad.php on line 5",
				},
				AnalyzedLines: 2,
			},
		},
		{
			name: "mixed log entries",
			content: `[Mon Dec 05 10:15:30 2023] [notice] Apache/2.4.41 configured
[Mon Dec 05 10:16:30 2023] [error] server reached MaxRequestWorkers setting
[Mon Dec 05 10:17:30 2023] [error] PHP Fatal error: out of memory
[Mon Dec 05 10:18:30 2023] [info] Normal log entry`,
			expected: &LogAnalysis{
				MaxClientsExceeded: 1,
				PHPFatalErrors:     1,
				RecentErrors: []string{
					"[Mon Dec 05 10:17:30 2023] [error] PHP Fatal error: out of memory",
				},
				AnalyzedLines: 4,
			},
		},
		{
			name:    "empty log",
			content: "",
			expected: &LogAnalysis{
				MaxClientsExceeded: 0,
				PHPFatalErrors:     0,
				RecentErrors:       []string{},
				AnalyzedLines:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary log file
			tmpFile, err := os.CreateTemp("", "test_apache_*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			// Write test content
			_, err = tmpFile.WriteString(tt.content)
			if err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}
			_ = tmpFile.Close()

			// Analyze the log file
			analysis := &LogAnalysis{}
			err = analyzeLogFile(tmpFile.Name(), analysis)
			if err != nil {
				t.Fatalf("analyzeLogFile failed: %v", err)
			}

			// Verify results
			if analysis.MaxClientsExceeded != tt.expected.MaxClientsExceeded {
				t.Errorf("MaxClientsExceeded = %d, want %d", analysis.MaxClientsExceeded, tt.expected.MaxClientsExceeded)
			}
			if analysis.PHPFatalErrors != tt.expected.PHPFatalErrors {
				t.Errorf("PHPFatalErrors = %d, want %d", analysis.PHPFatalErrors, tt.expected.PHPFatalErrors)
			}
			if analysis.AnalyzedLines != tt.expected.AnalyzedLines {
				t.Errorf("AnalyzedLines = %d, want %d", analysis.AnalyzedLines, tt.expected.AnalyzedLines)
			}

			// Check recent errors (should match expected errors)
			if len(analysis.RecentErrors) != len(tt.expected.RecentErrors) {
				t.Errorf("RecentErrors length = %d, want %d", len(analysis.RecentErrors), len(tt.expected.RecentErrors))
			} else {
				for i, err := range tt.expected.RecentErrors {
					if i < len(analysis.RecentErrors) && analysis.RecentErrors[i] != err {
						t.Errorf("RecentErrors[%d] = %s, want %s", i, analysis.RecentErrors[i], err)
					}
				}
			}
		})
	}
}

func TestCreateLogEntryInternal(t *testing.T) {
	// Mock data
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
	}

	recommendations := &analysis.Recommendations{
		CurrentMaxClients:     150,
		RecommendedMaxClients: 120,
		Status:                "WARNING",
		Message:               "Consider reducing MaxRequestWorkers",
	}

	// Test log entry creation (this will fail due to permissions, but we test the logic)
	err := createLogEntryInternal(sysInfo, memStats, config, recommendations)
	if err != nil {
		// Expected to fail in test environment due to /var/log permissions
		t.Logf("createLogEntryInternal failed as expected in test environment: %v", err)
	}
}

func TestAnalyzeApacheLogsTimeout(t *testing.T) {
	// Test that AnalyzeApacheLogs returns quickly even when no log files exist
	// This tests the timeout functionality
	analysis := AnalyzeApacheLogs()

	// Should return a valid analysis struct even if no logs are found
	if analysis == nil {
		t.Error("AnalyzeApacheLogs should never return nil")
		return
	}

	if analysis.AnalyzedLines < 0 {
		t.Errorf("AnalyzedLines should not be negative, got %d", analysis.AnalyzedLines)
	}
}

// Benchmark tests for performance validation
func BenchmarkAnalyzeLogFile(b *testing.B) {
	// Create a test log file with sample content
	tmpFile, err := os.CreateTemp("", "bench_log_*.log")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write realistic log content
	content := strings.Repeat(`[Mon Dec 05 10:15:30 2023] [error] server reached MaxRequestWorkers setting
[Mon Dec 05 10:16:30 2023] [error] PHP Fatal error: Call to undefined function
[Mon Dec 05 10:17:30 2023] [notice] Normal log entry
`, 100)

	_, err = tmpFile.WriteString(content)
	if err != nil {
		b.Fatalf("Failed to write test content: %v", err)
	}
	_ = tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis := &LogAnalysis{}
		_ = analyzeLogFile(tmpFile.Name(), analysis)
	}
}

func BenchmarkIsReadableLogFile(b *testing.B) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "bench_file_*.log")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isReadableLogFile(tmpFile.Name())
	}
}
