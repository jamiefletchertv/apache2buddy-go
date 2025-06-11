package output

import (
	"fmt"

	"apache2buddy/internal/analysis"
	"apache2buddy/internal/config"
	"apache2buddy/internal/logs"
	"apache2buddy/internal/status"
	"apache2buddy/internal/system"
)

func DisplayEnhancedResults(sysInfo *system.SystemInfo, memStats *analysis.MemoryStats, config *config.ApacheConfig, recommendations *analysis.Recommendations, statusInfo *status.ApacheStatus, logAnalysis *logs.LogAnalysis) {
	// System Info
	fmt.Println("=== SYSTEM INFO ===")
	fmt.Printf("Total Memory: %d MB\n", sysInfo.TotalMemoryMB)
	fmt.Printf("Available Memory: %d MB\n", sysInfo.AvailableMemoryMB)

	// Apache Config
	fmt.Println("=== APACHE CONFIG ===")
	fmt.Printf("MPM Model: %s\n", config.MPMModel)
	fmt.Printf("Config File: %s\n", config.ConfigPath)

	// Apache Processes
	fmt.Println("=== APACHE PROCESSES ===")
	fmt.Printf("Memory usage - Smallest: %.1f MB, Average: %.1f MB, Largest: %.1f MB\n",
		memStats.SmallestMB, memStats.AverageMB, memStats.LargestMB)
	fmt.Printf("Total Apache memory: %.1f MB\n", memStats.TotalMB)

	// Enhanced Recommendations
	fmt.Println("=== ENHANCED RECOMMENDATIONS ===")
	fmt.Printf("Status: %s\n", recommendations.Status)
	fmt.Printf("Message: %s\n", recommendations.Message)
	fmt.Printf("Current MaxClients: %d\n", recommendations.CurrentMaxClients)
	fmt.Printf("Recommended Range: %d - %d\n", recommendations.MinRecommended, recommendations.MaxRecommended)
	fmt.Printf("Conservative Recommendation: %d\n", recommendations.RecommendedMaxClients)
	
	// Calculate memory utilization
	if sysInfo.AvailableMemoryMB > 0 {
		utilization := (float64(recommendations.RecommendedMaxClients) * memStats.LargestMB) / float64(sysInfo.AvailableMemoryMB) * 100
		fmt.Printf("Memory Utilization: %.1f%%\n", utilization)
	}

	// Status-based advice
	switch recommendations.Status {
	case "OK":
		fmt.Printf("✓ Your Apache configuration looks good\n")
	case "WARNING":
		fmt.Printf("⚠️  Consider tuning your Apache configuration\n")
	case "CRITICAL":
		fmt.Printf("🔥 Critical: Apache configuration needs immediate attention\n")
	}

	// Apache Status (mod_status)
	fmt.Println("=== APACHE STATUS (mod_status) ===")
	if statusInfo != nil {
		// Basic information
		fmt.Printf("Active Workers: %d\n", statusInfo.ActiveWorkers)
		fmt.Printf("Requests/sec: %.3f\n", statusInfo.RequestsPerSec)
		
		// Extended information if available
		if statusInfo.ExtendedEnabled {
			fmt.Printf("✓ ExtendedStatus is enabled\n")
			
			// Performance metrics
			if statusInfo.AvgRequestTime > 0 {
				fmt.Printf("Average Request Time: %.2f ms\n", statusInfo.AvgRequestTime)
			}
			if statusInfo.CPULoadPercent > 0 {
				fmt.Printf("CPU Load: %.3f%%\n", statusInfo.CPULoadPercent)
			}
			if statusInfo.Load1Min > 0 {
				fmt.Printf("Load Averages: %.2f (1m), %.2f (5m), %.2f (15m)\n", 
					statusInfo.Load1Min, statusInfo.Load5Min, statusInfo.Load15Min)
			}
			
			// Traffic statistics
			if statusInfo.TotalRequests > 0 {
				fmt.Printf("Total Requests: %d\n", statusInfo.TotalRequests)
			}
			if statusInfo.TotalTrafficKB > 0 {
				fmt.Printf("Total Traffic: %d KB\n", statusInfo.TotalTrafficKB)
			}
			if statusInfo.BytesPerReq > 0 {
				fmt.Printf("Bytes per Request: %.0f B\n", statusInfo.BytesPerReq)
			}
			
			// Server information
			if statusInfo.Uptime != "" {
				fmt.Printf("Uptime: %s\n", statusInfo.Uptime)
			}
			if statusInfo.ServerVersion != "" {
				fmt.Printf("Server: %s\n", statusInfo.ServerVersion)
			}
			
			// Worker state analysis
			if statusInfo.TotalSlots > 0 {
				fmt.Printf("\n--- Worker Analysis ---\n")
				fmt.Printf("Total Worker Slots: %d\n", statusInfo.TotalSlots)
				if statusInfo.WorkersProcessing > 0 {
					fmt.Printf("Processing Requests: %d\n", statusInfo.WorkersProcessing)
				}
				if statusInfo.WorkersRestarting > 0 {
					fmt.Printf("Gracefully Restarting: %d\n", statusInfo.WorkersRestarting)
				}
				if statusInfo.WorkersWaiting > 0 {
					fmt.Printf("Waiting for Connection: %d\n", statusInfo.WorkersWaiting)
				}
				if statusInfo.WorkersWriting > 0 {
					fmt.Printf("Sending Replies: %d\n", statusInfo.WorkersWriting)
				}
				if statusInfo.WorkersReading > 0 {
					fmt.Printf("Reading Requests: %d\n", statusInfo.WorkersReading)
				}
				if statusInfo.WorkersKeepalive > 0 {
					fmt.Printf("Keepalive: %d\n", statusInfo.WorkersKeepalive)
				}
				if statusInfo.WorkersClosing > 0 {
					fmt.Printf("Closing Connections: %d\n", statusInfo.WorkersClosing)
				}
				if statusInfo.WorkersLogging > 0 {
					fmt.Printf("Logging: %d\n", statusInfo.WorkersLogging)
				}
				if statusInfo.WorkersFinishing > 0 {
					fmt.Printf("Gracefully Finishing: %d\n", statusInfo.WorkersFinishing)
				}
				if statusInfo.OpenSlots > 0 {
					fmt.Printf("Open Slots: %d\n", statusInfo.OpenSlots)
				}
				
				// Utilization percentage
				usedSlots := statusInfo.TotalSlots - statusInfo.OpenSlots
				if statusInfo.TotalSlots > 0 {
					utilization := float64(usedSlots) / float64(statusInfo.TotalSlots) * 100
					fmt.Printf("Worker Utilization: %.1f%% (%d/%d slots)\n", utilization, usedSlots, statusInfo.TotalSlots)
				}
			}
			
			// Client analysis
			if statusInfo.UniqueClients > 0 {
				fmt.Printf("\n--- Client Analysis ---\n")
				fmt.Printf("Unique Clients: %d\n", statusInfo.UniqueClients)
				if len(statusInfo.TopClients) > 0 {
					fmt.Printf("Most Active Clients:\n")
					for _, client := range statusInfo.TopClients {
						fmt.Printf("  %s\n", client)
					}
				}
			}
		} else {
			fmt.Printf("⚠️  Basic mod_status only (ExtendedStatus Off)\n")
			fmt.Printf("Tip: Enable 'ExtendedStatus On' for detailed metrics\n")
		}
	} else {
		fmt.Printf("⚠️  mod_status not accessible\n")
		fmt.Printf("Consider enabling mod_status:\n")
		fmt.Printf("  LoadModule status_module modules/mod_status.so\n")
		fmt.Printf("  <Location \"/server-status\">\n")
		fmt.Printf("    SetHandler server-status\n")
		fmt.Printf("    Require local\n")
		fmt.Printf("  </Location>\n")
		fmt.Printf("  ExtendedStatus On\n")
	}

	// Log Analysis
	fmt.Println("=== LOG ANALYSIS ===")
	if logAnalysis.AnalyzedLines > 0 {
		fmt.Printf("Analyzed %d log lines\n", logAnalysis.AnalyzedLines)
		if logAnalysis.MaxClientsExceeded > 0 {
			fmt.Printf("⚠️  MaxRequestWorkers exceeded: %d times\n", logAnalysis.MaxClientsExceeded)
		}
		if logAnalysis.PHPFatalErrors > 0 {
			fmt.Printf("⚠️  PHP Fatal Errors: %d\n", logAnalysis.PHPFatalErrors)
		}
		if len(logAnalysis.RecentErrors) > 0 {
			fmt.Printf("Recent errors:\n")
			for _, err := range logAnalysis.RecentErrors {
				fmt.Printf("  %s\n", err)
			}
		}
		if logAnalysis.MaxClientsExceeded == 0 && logAnalysis.PHPFatalErrors == 0 {
			fmt.Printf("✓ No critical issues found in logs\n")
		}
	} else {
		fmt.Printf("Note: Log analysis completed\n")
	}
}
