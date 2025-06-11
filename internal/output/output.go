package output

import (
	"fmt"
	"os/exec"
	"strings"

	"apache2buddy/internal/analysis"
	"apache2buddy/internal/config"
	"apache2buddy/internal/debug"
	"apache2buddy/internal/logs"
	"apache2buddy/internal/status"
	"apache2buddy/internal/system"
)

func DisplayEnhancedResults(sysInfo *system.SystemInfo, memStats *analysis.MemoryStats, config *config.ApacheConfig, recommendations *analysis.Recommendations, statusInfo *status.ApacheStatus, logAnalysis *logs.LogAnalysis) {
	fmt.Println()
	fmt.Println("apache2buddy - Enhanced Go Version")
	fmt.Println("==================================")
	fmt.Println()

	// Server Information Section
	fmt.Printf("Server Version: %s %s\n", config.ServerName, config.Version)
	fmt.Printf("Server MPM: %s\n", config.MPMModel)
	fmt.Printf("Server Built: %s\n", detectServerBuilt())
	fmt.Println()

	// System Memory Info
	fmt.Printf("Total RAM: %d MB\n", sysInfo.TotalMemoryMB)
	fmt.Printf("Available RAM: %d MB\n", sysInfo.AvailableMemoryMB)

	// Other Services (if any)
	totalOtherMemory := system.GetTotalOtherServicesMemory(sysInfo)
	if totalOtherMemory > 0 {
		fmt.Printf("RAM used by other services: %d MB\n", totalOtherMemory)
		fmt.Printf("RAM available for Apache: %d MB\n", sysInfo.AvailableMemoryMB)
	}
	fmt.Println()

	// Current Configuration
	fmt.Printf("Current MaxRequestWorkers: %d\n", config.GetCurrentMaxClients())
	if config.ServerLimit > 0 {
		fmt.Printf("Current ServerLimit: %d\n", config.ServerLimit)
	}
	fmt.Println()

	// Process Analysis
	if memStats.ProcessCount > 0 {
		fmt.Printf("Apache processes found: %d\n", memStats.ProcessCount)
		fmt.Printf("Memory usage per process: %.1f MB (smallest), %.1f MB (average), %.1f MB (largest)\n",
			memStats.SmallestMB, memStats.AverageMB, memStats.LargestMB)
		fmt.Println()
	}

	// Apache Status (if available)
	if statusInfo != nil {
		fmt.Printf("Active workers: %d, Idle workers: %d\n", statusInfo.ActiveWorkers, statusInfo.IdleWorkers)

		if statusInfo.RequestsPerSec > 0 {
			fmt.Printf("Requests per second: %.3f\n", statusInfo.RequestsPerSec)
		}

		if statusInfo.ExtendedEnabled && statusInfo.Load1Min > 0 {
			fmt.Printf("System load: %.2f (1min), %.2f (5min), %.2f (15min)\n",
				statusInfo.Load1Min, statusInfo.Load5Min, statusInfo.Load15Min)
		}
		fmt.Println()
	}

	// Memory Analysis and Recommendations
	currentMemoryUsage := float64(config.GetCurrentMaxClients()) * memStats.LargestMB
	currentUtilization := (currentMemoryUsage / float64(sysInfo.AvailableMemoryMB)) * 100

	fmt.Printf("Current memory usage: %.1f MB (%.1f%% of available)\n",
		currentMemoryUsage, currentUtilization)

	if recommendations.RecommendedMaxClients != recommendations.CurrentMaxClients {
		recommendedMemoryUsage := float64(recommendations.RecommendedMaxClients) * memStats.LargestMB
		recommendedUtilization := (recommendedMemoryUsage / float64(sysInfo.AvailableMemoryMB)) * 100

		fmt.Printf("Recommended MaxRequestWorkers: %d\n", recommendations.RecommendedMaxClients)
		fmt.Printf("Projected memory usage: %.1f MB (%.1f%% of available)\n",
			recommendedMemoryUsage, recommendedUtilization)
	}
	fmt.Println()

	// Status and Recommendations
	switch recommendations.Status {
	case "OK":
		fmt.Printf("✓ RESULT: Your Apache configuration appears to be optimal.\n")
	case "WARNING":
		fmt.Printf("⚠️  RESULT: Your Apache configuration could be improved.\n")
		if recommendations.RecommendedMaxClients < recommendations.CurrentMaxClients {
			fmt.Printf("Consider reducing MaxRequestWorkers to %d to prevent memory issues.\n", recommendations.RecommendedMaxClients)
		} else {
			fmt.Printf("Consider increasing MaxRequestWorkers to %d for better performance.\n", recommendations.RecommendedMaxClients)
		}
	case "CRITICAL":
		fmt.Printf("🔥 RESULT: Your Apache configuration needs immediate attention!\n")
		fmt.Printf("Reduce MaxRequestWorkers to %d to prevent memory issues.\n", recommendations.RecommendedMaxClients)
	}

	// MPM-specific notes
	if recommendations.MPMNote != "" {
		fmt.Printf("\nNote: %s\n", recommendations.MPMNote)
	}

	// Log Analysis Issues
	if logAnalysis.AnalyzedLines > 0 && (logAnalysis.MaxClientsExceeded > 0 || logAnalysis.PHPFatalErrors > 0) {
		fmt.Println()
		if logAnalysis.MaxClientsExceeded > 0 {
			fmt.Printf("⚠️  Log analysis shows MaxRequestWorkers was exceeded %d times.\n", logAnalysis.MaxClientsExceeded)
		}
		if logAnalysis.PHPFatalErrors > 0 {
			fmt.Printf("⚠️  Found %d PHP Fatal Errors in logs.\n", logAnalysis.PHPFatalErrors)
		}
	}

	// Configuration suggestions
	fmt.Println()
	fmt.Printf("Configuration file: %s\n", config.ConfigPath)
	if recommendations.Status != "OK" {
		fmt.Printf("\nTo implement changes, edit your Apache configuration:\n")
		fmt.Printf("<%s %s_module>\n", "IfModule", config.MPMModel)
		fmt.Printf("    MaxRequestWorkers %d\n", recommendations.RecommendedMaxClients)
		if config.MPMModel == "prefork" && recommendations.RecommendedMaxClients > 256 {
			fmt.Printf("    ServerLimit %d\n", recommendations.RecommendedMaxClients)
		}
		fmt.Printf("</%s>\n", "IfModule")
		fmt.Printf("\nThen restart Apache to apply changes.\n")
	}

	// Debug Information (only shown in debug mode)
	if debug.IsEnabled() {
		showDebugInformation(sysInfo, memStats, config, recommendations, statusInfo, logAnalysis)
	}

	fmt.Println()
	fmt.Printf("Analysis completed. Check /var/log/apache2buddy.log for historical data.\n")
}

// detectServerBuilt tries to get the Apache build date
func detectServerBuilt() string {
	commands := [][]string{
		{"httpd", "-v"},
		{"apache2", "-v"},
	}

	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Server built:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return "Unknown"
}

// showDebugInformation displays detailed technical information when debug mode is enabled
func showDebugInformation(sysInfo *system.SystemInfo, memStats *analysis.MemoryStats, config *config.ApacheConfig, recommendations *analysis.Recommendations, statusInfo *status.ApacheStatus, logAnalysis *logs.LogAnalysis) {
	debug.Section("DETAILED DEBUG INFORMATION")

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("DEBUG INFORMATION (detailed technical data)")
	fmt.Println(strings.Repeat("=", 60))

	// Detailed System Information
	fmt.Println("\n=== DETAILED SYSTEM INFO ===")
	fmt.Printf("Total Memory: %d MB\n", sysInfo.TotalMemoryMB)
	fmt.Printf("Available Memory: %d MB\n", sysInfo.AvailableMemoryMB)
	fmt.Printf("Other Services Memory: %d MB\n", system.GetTotalOtherServicesMemory(sysInfo))

	if len(sysInfo.OtherServices) > 0 {
		fmt.Printf("Service Breakdown:\n")
		for service, memory := range sysInfo.OtherServices {
			if service != "PHP-FPM-Note" {
				fmt.Printf("  - %s: %d MB\n", service, memory)
			}
		}
	}

	// Detailed Apache Configuration
	fmt.Println("\n=== DETAILED APACHE CONFIG ===")
	fmt.Printf("Server Name: %s\n", config.ServerName)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("MPM Model: %s\n", config.MPMModel)
	fmt.Printf("Config File: %s\n", config.ConfigPath)
	fmt.Printf("MaxClients (legacy): %d\n", config.MaxClients)
	fmt.Printf("MaxRequestWorkers: %d\n", config.MaxRequestWorkers)
	fmt.Printf("ServerLimit: %d\n", config.ServerLimit)
	fmt.Printf("ThreadsPerChild: %d\n", config.ThreadsPerChild)
	fmt.Printf("Effective MaxClients: %d\n", config.GetCurrentMaxClients())

	// Detailed Process Analysis
	fmt.Println("\n=== DETAILED PROCESS ANALYSIS ===")
	fmt.Printf("Process Count: %d\n", memStats.ProcessCount)
	fmt.Printf("Smallest Worker: %.2f MB\n", memStats.SmallestMB)
	fmt.Printf("Average Worker: %.2f MB\n", memStats.AverageMB)
	fmt.Printf("Largest Worker: %.2f MB\n", memStats.LargestMB)
	fmt.Printf("Total Memory Used: %.2f MB\n", memStats.TotalMB)

	// Detailed Recommendations Analysis
	fmt.Println("\n=== DETAILED RECOMMENDATIONS ===")
	fmt.Printf("Current MaxClients: %d\n", recommendations.CurrentMaxClients)
	fmt.Printf("Recommended MaxClients: %d\n", recommendations.RecommendedMaxClients)
	fmt.Printf("Min Recommended: %d\n", recommendations.MinRecommended)
	fmt.Printf("Max Recommended: %d\n", recommendations.MaxRecommended)
	fmt.Printf("Status: %s\n", recommendations.Status)
	fmt.Printf("Message: %s\n", recommendations.Message)
	fmt.Printf("Utilization Percent: %.2f%%\n", recommendations.UtilizationPercent)
	fmt.Printf("VHost Warning: %t\n", recommendations.VHostWarning)
	if recommendations.MPMNote != "" {
		fmt.Printf("MPM Note: %s\n", recommendations.MPMNote)
	}

	// Detailed Apache Status (mod_status)
	fmt.Println("\n=== DETAILED APACHE STATUS ===")
	if statusInfo != nil {
		fmt.Printf("Active Workers: %d\n", statusInfo.ActiveWorkers)
		fmt.Printf("Idle Workers: %d\n", statusInfo.IdleWorkers)
		fmt.Printf("Total Slots: %d\n", statusInfo.TotalSlots)
		fmt.Printf("Requests Per Second: %.6f\n", statusInfo.RequestsPerSec)
		fmt.Printf("Bytes Per Second: %.2f\n", statusInfo.BytesPerSec)
		fmt.Printf("Total Accesses: %d\n", statusInfo.TotalAccesses)
		fmt.Printf("Total KBytes: %d\n", statusInfo.TotalKBytes)
		fmt.Printf("Uptime: %s\n", statusInfo.Uptime)
		fmt.Printf("CPU Usage: %.6f\n", statusInfo.CPUUsage)
		fmt.Printf("Extended Enabled: %t\n", statusInfo.ExtendedEnabled)
		fmt.Printf("Avg Request Time: %.6f\n", statusInfo.AvgRequestTime)
		fmt.Printf("CPU Load Percent: %.6f\n", statusInfo.CPULoadPercent)
		fmt.Printf("Load 1Min: %.2f\n", statusInfo.Load1Min)
		fmt.Printf("Load 5Min: %.2f\n", statusInfo.Load5Min)
		fmt.Printf("Load 15Min: %.2f\n", statusInfo.Load15Min)
		fmt.Printf("Bytes Per Request: %.2f\n", statusInfo.BytesPerReq)
		fmt.Printf("Server Version: %s\n", statusInfo.ServerVersion)
		fmt.Printf("Unique Clients: %d\n", statusInfo.UniqueClients)

		// Worker state details
		fmt.Printf("\nWorker State Breakdown:\n")
		fmt.Printf("  Processing: %d\n", statusInfo.WorkersProcessing)
		fmt.Printf("  Restarting: %d\n", statusInfo.WorkersRestarting)
		fmt.Printf("  Waiting: %d\n", statusInfo.WorkersWaiting)
		fmt.Printf("  Writing: %d\n", statusInfo.WorkersWriting)
		fmt.Printf("  Reading: %d\n", statusInfo.WorkersReading)
		fmt.Printf("  Keepalive: %d\n", statusInfo.WorkersKeepalive)
		fmt.Printf("  Closing: %d\n", statusInfo.WorkersClosing)
		fmt.Printf("  Logging: %d\n", statusInfo.WorkersLogging)
		fmt.Printf("  Finishing: %d\n", statusInfo.WorkersFinishing)
		fmt.Printf("  Open Slots: %d\n", statusInfo.OpenSlots)

		// Top clients
		if len(statusInfo.TopClients) > 0 {
			fmt.Printf("\nTop Clients:\n")
			for i, client := range statusInfo.TopClients {
				fmt.Printf("  %d. IP: %s, Requests: %d, Bytes: %d, Status: %s\n",
					i+1, client.IP, client.Requests, client.Bytes, client.Status)
			}
		}
	} else {
		fmt.Printf("mod_status not accessible\n")
	}

	// Detailed Log Analysis
	fmt.Println("\n=== DETAILED LOG ANALYSIS ===")
	fmt.Printf("Analyzed Lines: %d\n", logAnalysis.AnalyzedLines)
	fmt.Printf("MaxClients Exceeded: %d\n", logAnalysis.MaxClientsExceeded)
	fmt.Printf("PHP Fatal Errors: %d\n", logAnalysis.PHPFatalErrors)

	if len(logAnalysis.RecentErrors) > 0 {
		fmt.Printf("Recent Errors (%d):\n", len(logAnalysis.RecentErrors))
		for i, err := range logAnalysis.RecentErrors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}

	// Memory Calculations Debug
	fmt.Println("\n=== MEMORY CALCULATION DEBUG ===")
	if memStats.ProcessCount > 0 {
		currentMemoryUsage := float64(config.GetCurrentMaxClients()) * memStats.LargestMB
		recommendedMemoryUsage := float64(recommendations.RecommendedMaxClients) * memStats.LargestMB

		fmt.Printf("Current Config Memory Usage:\n")
		fmt.Printf("  MaxClients: %d\n", config.GetCurrentMaxClients())
		fmt.Printf("  × Largest Process: %.2f MB\n", memStats.LargestMB)
		fmt.Printf("  = Total Usage: %.2f MB\n", currentMemoryUsage)
		fmt.Printf("  / Available: %d MB\n", sysInfo.AvailableMemoryMB)
		fmt.Printf("  = Utilization: %.1f%%\n", (currentMemoryUsage/float64(sysInfo.AvailableMemoryMB))*100)

		fmt.Printf("\nRecommended Config Memory Usage:\n")
		fmt.Printf("  Recommended MaxClients: %d\n", recommendations.RecommendedMaxClients)
		fmt.Printf("  × Largest Process: %.2f MB\n", memStats.LargestMB)
		fmt.Printf("  = Total Usage: %.2f MB\n", recommendedMemoryUsage)
		fmt.Printf("  / Available: %d MB\n", sysInfo.AvailableMemoryMB)
		fmt.Printf("  = Utilization: %.1f%%\n", (recommendedMemoryUsage/float64(sysInfo.AvailableMemoryMB))*100)

		fmt.Printf("\nMemory Safety Calculations:\n")
		fmt.Printf("  Available Memory: %d MB\n", sysInfo.AvailableMemoryMB)
		fmt.Printf("  Max Theoretical (100%%): %d workers\n", int(float64(sysInfo.AvailableMemoryMB)/memStats.LargestMB))
		fmt.Printf("  Conservative (90%%): %d workers\n", int(float64(sysInfo.AvailableMemoryMB)/memStats.LargestMB*0.9))
	}

	fmt.Println(strings.Repeat("=", 60))
}
