package logs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"apache2buddy/internal/analysis"
	"apache2buddy/internal/config"
	"apache2buddy/internal/system"
)

type LogAnalysis struct {
	MaxClientsExceeded int
	PHPFatalErrors     int
	RecentErrors       []string
	AnalyzedLines      int
}

func AnalyzeApacheLogs() *LogAnalysis {
	analysis := &LogAnalysis{}

	// Common Apache log paths
	logPaths := []string{
		"/var/log/apache2/error.log",
		"/var/log/httpd/error_log",
		"/var/log/apache2/error_log",
		"/usr/local/apache2/logs/error_log",
	}

	for _, logPath := range logPaths {
		// Check if log file exists and is not a device/pipe
		if !isReadableLogFile(logPath) {
			continue
		}

		// Set a timeout for log analysis to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to analyze logs with timeout
		done := make(chan error, 1)
		go func() {
			done <- analyzeLogFile(logPath, analysis)
		}()

		// Wait for completion or timeout
		select {
		case err := <-done:
			if err == nil {
				return analysis // Successfully analyzed a log file
			}
		case <-ctx.Done():
			// Timeout occurred, continue to next log file
			continue
		}
	}

	return analysis
}

func isReadableLogFile(logPath string) bool {
	// Check if file exists
	fileInfo, err := os.Lstat(logPath)
	if err != nil {
		return false
	}

	// If it's a symlink, check where it points
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(logPath)
		if err != nil {
			return false
		}

		// Skip if symlinked to stdout/stderr/dev devices
		if linkTarget == "/dev/stdout" || linkTarget == "/dev/stderr" || 
		   linkTarget == "/dev/null" || strings.HasPrefix(linkTarget, "/dev/") {
			fmt.Printf("Note: Apache logs are redirected to %s (containerized setup) - log analysis skipped\n", linkTarget)
			return false
		}
	}

	// Check if it's a regular file we can read
	if !fileInfo.Mode().IsRegular() && fileInfo.Mode()&os.ModeSymlink == 0 {
		return false
	}

	// Try to open the file briefly to see if it's readable
	file, err := os.Open(logPath)
	if err != nil {
		return false
	}
	file.Close()

	return true
}

func analyzeLogFile(logPath string, analysis *LogAnalysis) error {
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linesRead := 0
	maxLines := 1000 // Reduced from 10000 to prevent hanging on huge logs

	for scanner.Scan() && linesRead < maxLines {
		line := scanner.Text()
		linesRead++

		// Check for MaxRequestWorkers/MaxClients exceeded
		if strings.Contains(line, "server reached MaxRequestWorkers") ||
			strings.Contains(line, "server reached MaxClients") {
			analysis.MaxClientsExceeded++
		}

		// Check for PHP fatal errors
		if strings.Contains(line, "PHP Fatal error") ||
			strings.Contains(line, "PHP Parse error") {
			analysis.PHPFatalErrors++
			if len(analysis.RecentErrors) < 5 {
				analysis.RecentErrors = append(analysis.RecentErrors, line)
			}
		}

		// Add a small delay every 100 lines to prevent CPU hogging
		if linesRead%100 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	analysis.AnalyzedLines = linesRead
	return scanner.Err()
}

func CreateLogEntry(sysInfo *system.SystemInfo, memStats *analysis.MemoryStats, config *config.ApacheConfig, recommendations *analysis.Recommendations) error {
	// Add timeout for log creation too
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- createLogEntryInternal(sysInfo, memStats, config, recommendations)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("log creation timed out")
	}
}

func createLogEntryInternal(sysInfo *system.SystemInfo, memStats *analysis.MemoryStats, config *config.ApacheConfig, recommendations *analysis.Recommendations) error {
	logFile := "/var/log/apache2buddy.log"

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Format("2006/01/02 15:04:05")

	// Format: Date Uptime Model Memory MaxClients Recommended Smallest Avg Largest
	logEntry := fmt.Sprintf(`%s Memory: "%d MB" MaxClients: "%d" Recommended: "%d" Status: "%s" Smallest: "%.2f MB" Avg: "%.2f MB" Largest: "%.2f MB" MPM: "%s"`+"\n",
		timestamp,
		sysInfo.AvailableMemoryMB,
		config.GetCurrentMaxClients(),
		recommendations.RecommendedMaxClients,
		recommendations.Status,
		memStats.SmallestMB,
		memStats.AverageMB,
		memStats.LargestMB,
		config.MPMModel,
	)

	_, err = file.WriteString(logEntry)
	return err
}

func GetRecentLogEntries(count int) ([]string, error) {
	// Add timeout for reading log entries
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan []string, 1)
	errChan := make(chan error, 1)

	go func() {
		logFile := "/var/log/apache2buddy.log"

		file, err := os.Open(logFile)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		// Return last 'count' lines
		if len(lines) > count {
			done <- lines[len(lines)-count:]
		} else {
			done <- lines
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	select {
	case lines := <-done:
		return lines, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("reading log entries timed out")
	}
}
