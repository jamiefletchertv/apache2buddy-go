package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"apache2buddy-go/internal/analysis"
	"apache2buddy-go/internal/config"
	"apache2buddy-go/internal/debug"
	"apache2buddy-go/internal/logs"
	"apache2buddy-go/internal/output"
	"apache2buddy-go/internal/process"
	"apache2buddy-go/internal/status"
	"apache2buddy-go/internal/system"
)

func main() {
	// Parse command line flags
	var (
		debugFlag   = flag.Bool("debug", false, "Enable debug mode for detailed troubleshooting")
		helpFlag    = flag.Bool("help", false, "Show help information")
		versionFlag = flag.Bool("version", false, "Show version information")
		historyFlag = flag.Int("history", 0, "Show last N entries from apache2buddy log file")
	)
	flag.Parse()

	// Handle help flag
	if *helpFlag {
		showHelp()
		return
	}

	// Handle version flag
	if *versionFlag {
		showVersion()
		return
	}

	// Handle history flag
	if *historyFlag > 0 {
		showHistory(*historyFlag)
		return
	}

	// Enable debug mode if requested
	if *debugFlag {
		debug.Enable()
		debug.DumpSystemInfo()
	}

	fmt.Println("Apache2Buddy Go")
	fmt.Println("==================================")

	// Debug timing
	totalTimer := debug.StartTimer("Total Analysis")
	defer totalTimer.Stop()

	// Check root access
	debug.Info("Checking root access")
	if os.Geteuid() != 0 {
		debug.Error(fmt.Errorf("not running as root (uid=%d)", os.Geteuid()), "root check")
		log.Fatal("This script must be run as root")
	}
	debug.Info("Running as root (uid=0)")

	// Check required commands exist
	debug.Section("CHECKING REQUIRED COMMANDS")
	cmdTimer := debug.StartTimer("Command Check")
	if err := system.CheckRequiredCommands(); err != nil {
		debug.Error(err, "command check")
		log.Fatalf("Missing required commands: %v", err)
	}
	cmdTimer.Stop()
	debug.Info("All required commands found")

	// Get system memory
	debug.Section("GATHERING SYSTEM INFORMATION")
	sysTimer := debug.StartTimer("System Info")
	sysInfo, err := system.GetInfo()
	if err != nil {
		debug.Error(err, "system info")
		log.Fatalf("Failed to get system info: %v", err)
	}
	sysTimer.Stop()
	debug.DumpStruct("SystemInfo", sysInfo)

	// Parse Apache configuration with enhanced version detection
	debug.Section("PARSING APACHE CONFIGURATION")
	configTimer := debug.StartTimer("Config Parse")
	apacheConfig, err := config.ParseWithVersion()
	if err != nil {
		debug.Warn("Could not parse Apache config: %v", err)
		// Only show warning in debug mode, not in normal output
		if !debug.IsEnabled() {
			fmt.Printf("Warning: Could not parse Apache config, using defaults\n")
		}
		apacheConfig = config.GetDefaults()
		debug.Info("Using default Apache configuration")
	}
	configTimer.Stop()
	debug.DumpStruct("ApacheConfig", apacheConfig)

	// Check for control panels
	debug.Info("Checking for control panels")
	if controlPanel := system.DetectControlPanels(); controlPanel != "" {
		fmt.Printf("⚠️  Control Panel Detected: %s - Be careful modifying config files manually!\n", controlPanel)
		debug.Info("Control panel detected: %s", controlPanel)
	} else {
		debug.Info("No control panel detected")
	}

	// Detect additional services
	debug.Section("DETECTING SERVICES")
	serviceTimer := debug.StartTimer("Service Detection")
	system.DetectServices(sysInfo)
	system.DetectPHPFPM(sysInfo, apacheConfig.MPMModel) // Enhanced PHP-FPM detection
	serviceTimer.Stop()
	debug.DumpMap("DetectedServices", sysInfo.OtherServices)

	totalOtherMemory := system.GetTotalOtherServicesMemory(sysInfo)
	if totalOtherMemory > 0 {
		// Only show this in debug mode or if significant
		if debug.IsEnabled() || totalOtherMemory > 100 {
			fmt.Printf("Other services using %d MB RAM\n", totalOtherMemory)
		}
		sysInfo.AvailableMemoryMB -= totalOtherMemory
		debug.Info("Adjusted available memory after services: %d MB", sysInfo.AvailableMemoryMB)
	}

	// Get Apache status information (mod_status)
	debug.Section("RETRIEVING APACHE STATUS")
	statusTimer := debug.StartTimer("Apache Status")
	statusInfo, err := status.GetApacheStatus()
	if err != nil {
		debug.Warn("Could not get Apache status info: %v", err)
		// Only show this warning in debug mode
		if debug.IsEnabled() {
			log.Printf("Note: Could not get Apache status info: %v", err)
			log.Printf("Consider enabling mod_status with ExtendedStatus On for better analysis")
		}
	} else {
		debug.DumpStruct("ApacheStatus", statusInfo)
	}
	statusTimer.Stop()

	// Find Apache processes
	debug.Section("FINDING APACHE PROCESSES")
	processTimer := debug.StartTimer("Process Discovery")
	processes, err := process.FindApacheProcesses()
	if err != nil {
		debug.Error(err, "process discovery")
		log.Fatalf("Failed to find Apache processes: %v", err)
	}
	processTimer.Stop()

	if len(processes) == 0 {
		debug.Error(fmt.Errorf("no processes found"), "process discovery")
		log.Fatal("No Apache worker processes found. Is Apache running?")
	}

	debug.Info("Found %d Apache worker processes", len(processes))
	debug.DumpSlice("ApacheProcesses", processes)

	// Get virtual host count
	debug.Info("Counting virtual hosts")
	vhostCount := config.GetVirtualHostCount(apacheConfig.ConfigPath)
	debug.Info("Virtual hosts found: %d", vhostCount)

	// Check Apache logs for issues
	debug.Section("ANALYZING APACHE LOGS")
	logTimer := debug.StartTimer("Log Analysis")
	logAnalysis := logs.AnalyzeApacheLogs()
	logTimer.Stop()
	debug.DumpStruct("LogAnalysis", logAnalysis)

	// Calculate memory statistics and enhanced recommendations
	debug.Section("CALCULATING RECOMMENDATIONS")
	memTimer := debug.StartTimer("Memory Analysis")
	memStats := analysis.CalculateMemoryStats(processes)
	recommendations := analysis.GenerateEnhancedRecommendations(sysInfo, memStats, apacheConfig, statusInfo, vhostCount)
	memTimer.Stop()

	debug.DumpStruct("MemoryStats", memStats)
	debug.DumpStruct("Recommendations", recommendations)

	// Display enhanced results (this handles all the main output)
	debug.Section("GENERATING REPORT")
	reportTimer := debug.StartTimer("Report Generation")
	output.DisplayEnhancedResults(sysInfo, memStats, apacheConfig, recommendations, statusInfo, logAnalysis)
	reportTimer.Stop()

	// Create log entry for historical tracking
	debug.Info("Creating log entry")
	logEntryTimer := debug.StartTimer("Log Entry Creation")
	if err := logs.CreateLogEntry(sysInfo, memStats, apacheConfig, recommendations); err != nil {
		debug.Warn("Could not create log entry: %v", err)
		if debug.IsEnabled() {
			fmt.Printf("Note: Could not create log entry: %v\n", err)
		}
	}
	logEntryTimer.Stop()

	// Exit with status code based on recommendations
	debug.Info("Exiting with status based on recommendations: %s", recommendations.Status)
	switch recommendations.Status {
	case "OK":
		debug.Info("Exiting with code 0 (OK)")
		os.Exit(0)
	case "WARNING":
		debug.Info("Exiting with code 1 (WARNING)")
		os.Exit(1)
	case "CRITICAL":
		debug.Info("Exiting with code 2 (CRITICAL)")
		os.Exit(2)
	default:
		debug.Info("Exiting with code 0 (default)")
		os.Exit(0)
	}
}

func showHelp() {
	fmt.Println("Apache2Buddy Go")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  apache2buddy [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -debug         Enable debug mode for detailed troubleshooting output")
	fmt.Println("  -help          Show this help information")
	fmt.Println("  -version       Show version information")
	fmt.Println("  -history N     Show last N entries from apache2buddy log file")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  Analyzes Apache HTTP Server configuration and provides tuning recommendations")
	fmt.Println("  based on current memory usage and system resources.")
	fmt.Println()
	fmt.Println("REQUIREMENTS:")
	fmt.Println("  - Must be run as root")
	fmt.Println("  - Requires 'ps' and 'pmap' commands")
	fmt.Println("  - Apache must be running")
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println("  - Exit code 0: Configuration OK")
	fmt.Println("  - Exit code 1: Configuration needs tuning (WARNING)")
	fmt.Println("  - Exit code 2: Configuration critical (CRITICAL)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  sudo ./apache2buddy-go                    # Normal analysis")
	fmt.Println("  sudo ./apache2buddy-go -debug             # Debug mode with detailed output")
	fmt.Println("  sudo ./apache2buddy-go -history 10        # Show last 10 log entries")
	fmt.Println()
	fmt.Println("LOG FILE:")
	fmt.Println("  Historical data is logged to /var/log/apache2buddy-go.log")
	fmt.Println()
}

func showHistory(count int) {
	fmt.Printf("apache2buddy-go Historical Log (last %d entries)\n", count)
	fmt.Println(strings.Repeat("=", 60))

	entries, err := logs.GetRecentLogEntries(count)
	if err != nil {
		fmt.Printf("Error reading log file: %v\n", err)
		fmt.Println("Make sure /var/log/apache2buddy-go.log exists and is readable.")
		return
	}

	if len(entries) == 0 {
		fmt.Println("No log entries found.")
		fmt.Println("Run apache2buddy-go at least once to generate log entries.")
		return
	}

	for _, entry := range entries {
		fmt.Println(entry)
	}

	fmt.Printf("\nShowing %d of %d available entries.\n", len(entries), len(entries))
	fmt.Println("Use -history with a larger number to see more entries.")
}

func showVersion() {
	fmt.Println("apache2buddy-go")
	fmt.Println("Version: 1.0.0")
	fmt.Println("Based on apache2buddy.pl by Richard Petersen")
	fmt.Println("Go implementation with enhanced features")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  - Memory usage analysis")
	fmt.Println("  - Apache configuration parsing")
	fmt.Println("  - mod_status integration")
	fmt.Println("  - Log analysis")
	fmt.Println("  - Service detection")
	fmt.Println("  - Historical logging")
	fmt.Println("  - Debug mode")
	fmt.Println()
}
