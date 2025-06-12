package analysis

import (
	"apache2buddy-go/internal/config"
	"apache2buddy-go/internal/process"
	"apache2buddy-go/internal/system"
)

type MemoryStats struct {
	SmallestMB   float64
	LargestMB    float64
	AverageMB    float64
	TotalMB      float64
	ProcessCount int
}

type Recommendations struct {
	CurrentMaxClients     int
	RecommendedMaxClients int
	MinRecommended        int
	MaxRecommended        int
	Status                string
	Message               string
	UtilizationPercent    float64
	VHostWarning          bool
	MPMNote               string
}

func CalculateMemoryStats(processes []process.ProcessInfo) *MemoryStats {
	if len(processes) == 0 {
		return &MemoryStats{
			ProcessCount: 0,
		}
	}

	stats := &MemoryStats{
		SmallestMB:   processes[0].MemoryMB,
		LargestMB:    processes[0].MemoryMB,
		ProcessCount: len(processes),
	}

	var totalMemory float64

	for _, proc := range processes {
		totalMemory += proc.MemoryMB

		if proc.MemoryMB < stats.SmallestMB {
			stats.SmallestMB = proc.MemoryMB
		}

		if proc.MemoryMB > stats.LargestMB {
			stats.LargestMB = proc.MemoryMB
		}
	}

	stats.TotalMB = totalMemory
	stats.AverageMB = totalMemory / float64(len(processes))

	return stats
}

func GenerateRecommendations(sysInfo *system.SystemInfo, memStats *MemoryStats, config *config.ApacheConfig) *Recommendations {
	if memStats.ProcessCount == 0 {
		return &Recommendations{
			Status:  "ERROR",
			Message: "No processes to analyze",
		}
	}

	// Use largest process memory for conservative calculation
	largestMB := memStats.LargestMB

	// Calculate recommendation based on remaining available memory
	recommendedMaxClients := int(float64(sysInfo.AvailableMemoryMB) / largestMB * 0.9) // 90% safety margin

	// Get actual current MaxClients from config
	currentMaxClients := config.GetCurrentMaxClients()

	// Calculate utilization
	potentialUsage := float64(currentMaxClients) * largestMB
	utilizationPercent := (potentialUsage / float64(sysInfo.AvailableMemoryMB)) * 100

	// Determine status
	var status, message string
	if currentMaxClients <= recommendedMaxClients {
		status = "OK"
		message = "Configuration appears acceptable"
	} else {
		status = "HIGH"
		message = "Consider reducing MaxClients/MaxRequestWorkers"
	}

	return &Recommendations{
		CurrentMaxClients:     currentMaxClients,
		RecommendedMaxClients: recommendedMaxClients,
		Status:                status,
		Message:               message,
		UtilizationPercent:    utilizationPercent,
	}
}

// GenerateEnhancedRecommendations provides comprehensive analysis like original apache2buddy.pl
func GenerateEnhancedRecommendations(sysInfo *system.SystemInfo, memStats *MemoryStats, config *config.ApacheConfig, statusInfo interface{}, vhostCount int) *Recommendations {
	if memStats.ProcessCount == 0 {
		return &Recommendations{
			Status:  "ERROR",
			Message: "No processes to analyze",
		}
	}

	// Use largest process memory for conservative calculation
	largestMB := memStats.LargestMB

	// Calculate range recommendations (90-100% of remaining RAM)
	maxRecommended := int(float64(sysInfo.AvailableMemoryMB) / largestMB)       // 100%
	minRecommended := int(float64(sysInfo.AvailableMemoryMB) / largestMB * 0.9) // 90%

	// Get actual current MaxClients from config
	currentMaxClients := config.GetCurrentMaxClients()

	// Calculate utilization
	potentialUsage := float64(currentMaxClients) * largestMB
	utilizationPercent := (potentialUsage / float64(sysInfo.AvailableMemoryMB)) * 100

	// Enhanced status determination
	var status, message, mpmNote string
	var vhostWarning bool

	// Check virtual host vs MaxClients relationship
	if vhostCount > maxRecommended {
		vhostWarning = true
	}

	// MPM-specific notes (like original apache2buddy.pl)
	if config.MPMModel == "worker" || config.MPMModel == "event" {
		mpmNote = "Apache is running in " + config.MPMModel + " mode. Check manually for backend processes such as PHP-FPM and pm.max_children."
	}

	// Determine overall status
	if currentMaxClients <= minRecommended {
		status = "OK"
		message = "Configuration appears acceptable"
	} else if currentMaxClients <= maxRecommended {
		status = "WARNING"
		message = "Configuration is on the high side but acceptable"
	} else {
		status = "CRITICAL"
		message = "Consider reducing MaxClients/MaxRequestWorkers to avoid memory issues"
	}

	return &Recommendations{
		CurrentMaxClients:     currentMaxClients,
		RecommendedMaxClients: minRecommended, // Conservative recommendation
		MinRecommended:        minRecommended,
		MaxRecommended:        maxRecommended,
		Status:                status,
		Message:               message,
		UtilizationPercent:    utilizationPercent,
		VHostWarning:          vhostWarning,
		MPMNote:               mpmNote,
	}
}
