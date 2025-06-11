package main

import (
	"fmt"
	"log"
	"os"

	"apache2buddy/internal/analysis"
	"apache2buddy/internal/config"
	"apache2buddy/internal/logs"
	"apache2buddy/internal/output"
	"apache2buddy/internal/process"
	"apache2buddy/internal/status"
	"apache2buddy/internal/system"
)

func main() {
	fmt.Println("Apache2Buddy Go - Enhanced Version")
	fmt.Println("==================================")

	// Check root access
	if os.Geteuid() != 0 {
		log.Fatal("This script must be run as root")
	}

	// Check required commands exist
	if err := system.CheckRequiredCommands(); err != nil {
		log.Fatalf("Missing required commands: %v", err)
	}

	// Get system memory
	sysInfo, err := system.GetInfo()
	if err != nil {
		log.Fatalf("Failed to get system info: %v", err)
	}

	fmt.Printf("System: %d MB total, %d MB available\n", sysInfo.TotalMemoryMB, sysInfo.AvailableMemoryMB)

	// Parse Apache configuration with enhanced version detection
	apacheConfig, err := config.ParseWithVersion()
	if err != nil {
		log.Printf("Warning: Could not parse Apache config: %v", err)
		apacheConfig = config.GetDefaults()
	}

	fmt.Printf("Apache: %s v%s, %s MPM, MaxClients/MaxRequestWorkers: %d\n",
		apacheConfig.ServerName, apacheConfig.Version, apacheConfig.MPMModel, apacheConfig.GetCurrentMaxClients())

	// Check for control panels
	if controlPanel := system.DetectControlPanels(); controlPanel != "" {
		fmt.Printf("⚠️  Control Panel Detected: %s - Be careful modifying config files manually!\n", controlPanel)
	}

	// Detect additional services
	system.DetectServices(sysInfo)
	system.DetectPHPFPM(sysInfo, apacheConfig.MPMModel) // Enhanced PHP-FPM detection

	totalOtherMemory := system.GetTotalOtherServicesMemory(sysInfo)
	if totalOtherMemory > 0 {
		for service, memory := range sysInfo.OtherServices {
			fmt.Printf("Service detected: %s using %d MB\n", service, memory)
		}
		fmt.Printf("Total other services: %d MB\n", totalOtherMemory)
		sysInfo.AvailableMemoryMB -= totalOtherMemory
		fmt.Printf("Remaining for Apache: %d MB\n", sysInfo.AvailableMemoryMB)
	}

	// Get Apache status information (mod_status)
	statusInfo, err := status.GetApacheStatus()
	if err != nil {
		log.Printf("Note: Could not get Apache status info: %v", err)
		log.Printf("Consider enabling mod_status with ExtendedStatus On for better analysis")
	} else {
		fmt.Printf("Apache Status: %d active workers, %.1f requests/sec\n",
			statusInfo.ActiveWorkers, statusInfo.RequestsPerSec)
	}

	// Find Apache processes
	processes, err := process.FindApacheProcesses()
	if err != nil {
		log.Fatalf("Failed to find Apache processes: %v", err)
	}

	if len(processes) == 0 {
		log.Fatal("No Apache worker processes found. Is Apache running?")
	}

	fmt.Printf("Found %d Apache worker processes\n", len(processes))

	// Get virtual host count
	vhostCount := config.GetVirtualHostCount(apacheConfig.ConfigPath)
	if vhostCount > 0 {
		fmt.Printf("Virtual hosts configured: %d\n", vhostCount)
	}

	// Check Apache logs for issues
	fmt.Printf("Analyzing Apache logs...\n")
	logAnalysis := logs.AnalyzeApacheLogs()
	if logAnalysis.MaxClientsExceeded > 0 {
		fmt.Printf("⚠️  MaxRequestWorkers exceeded %d times in logs\n", logAnalysis.MaxClientsExceeded)
	}

	// Calculate memory statistics and enhanced recommendations
	fmt.Printf("Calculating recommendations...\n")
	memStats := analysis.CalculateMemoryStats(processes)
	recommendations := analysis.GenerateEnhancedRecommendations(sysInfo, memStats, apacheConfig, statusInfo, vhostCount)

	// Display enhanced results
	fmt.Printf("Generating report...\n")
	output.DisplayEnhancedResults(sysInfo, memStats, apacheConfig, recommendations, statusInfo, logAnalysis)

	// Create log entry for historical tracking
	fmt.Printf("Creating log entry...\n")
	if err := logs.CreateLogEntry(sysInfo, memStats, apacheConfig, recommendations); err != nil {
		fmt.Printf("Note: Could not create log entry: %v\n", err)
	}

	fmt.Printf("Analysis complete!\n")

	// Exit with status code based on recommendations
	switch recommendations.Status {
	case "OK":
		os.Exit(0)
	case "WARNING":
		os.Exit(1)
	case "CRITICAL":
		os.Exit(2)
	default:
		os.Exit(0)
	}
}
