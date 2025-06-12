package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID      int
	User     string
	MemoryMB float64
}

func FindApacheProcesses() ([]ProcessInfo, error) {
	// For Alpine/BusyBox, use simple ps and grep approach
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps command failed: %v", err)
	}

	return parseAuxFormat(string(output))
}

func parseAuxFormat(output string) ([]ProcessInfo, error) {
	var processes []ProcessInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header line
		if strings.Contains(line, "USER") && strings.Contains(line, "PID") {
			continue
		}

		// Parse ps aux output
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		var user string
		var pid int
		var command string
		var err error

		// Handle different ps formats
		if len(fields) >= 11 {
			// Standard GNU ps aux format: USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND
			user = fields[0]
			pid, err = strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			command = strings.Join(fields[10:], " ")
		} else if len(fields) >= 4 {
			// Alpine BusyBox ps aux format: PID USER TIME COMMAND
			pid, err = strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			user = fields[1]
			command = strings.Join(fields[3:], " ")
		} else {
			continue
		}

		// Check if this is an Apache process based on command
		if !isApacheCommand(command) {
			continue
		}

		// Skip root processes (master processes)
		if user == "root" {
			continue
		}

		// Get memory for this process
		memory, err := getProcessMemory(pid)
		if err != nil {
			continue
		}

		processes = append(processes, ProcessInfo{
			PID:      pid,
			User:     user,
			MemoryMB: memory,
		})
	}

	return processes, nil
}

func isApacheProcess(comm string) bool {
	apacheNames := []string{"httpd", "apache2", "httpd.worker", "httpd-prefork"}
	for _, name := range apacheNames {
		if comm == name {
			return true
		}
	}
	return false
}

func isApacheCommand(command string) bool {
	// More comprehensive Apache detection, but exclude our own script
	if strings.Contains(command, "apache2buddy") {
		return false
	}

	apacheIndicators := []string{
		"httpd",
		"apache2",
		"/usr/sbin/httpd",
		"/usr/sbin/apache2",
		"/usr/local/apache2/bin/httpd",
	}

	for _, indicator := range apacheIndicators {
		if strings.Contains(command, indicator) {
			return true
		}
	}
	return false
}

func getProcessMemory(pid int) (float64, error) {
	// Alpine/BusyBox doesn't have pmap, so let's try different approaches

	// Method 1: Try reading /proc/PID/status directly
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	if data, err := os.ReadFile(statusFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if memKB, err := strconv.ParseFloat(fields[1], 64); err == nil {
						return memKB / 1024, nil // Convert to MB
					}
				}
			}
		}
	}

	// Method 2: Try ps with simpler format for BusyBox
	cmd := exec.Command("ps", "-o", "pid,rss")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if linePid, err := strconv.Atoi(fields[0]); err == nil && linePid == pid {
					if rssKB, err := strconv.ParseFloat(fields[1], 64); err == nil {
						return rssKB / 1024, nil
					}
				}
			}
		}
	}

	// Method 3: Default fallback - assume 10MB per process (conservative)
	return 10.0, nil
}

func parsePmapOutput(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "writeable/private") ||
			strings.Contains(strings.ToLower(line), "writable-private") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasSuffix(field, "K") {
					memStr := strings.TrimSuffix(field, "K")
					if memKB, err := strconv.ParseFloat(memStr, 64); err == nil {
						return memKB / 1024 // Convert to MB
					}
				}
			}
		}
	}
	return 0
}
