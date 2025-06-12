package analysis

import (
	"testing"

	"apache2buddy-go/internal/config"
	"apache2buddy-go/internal/process"
	"apache2buddy-go/internal/system"
)

func TestCalculateMemoryStats(t *testing.T) {
	tests := []struct {
		name      string
		processes []process.ProcessInfo
		want      *MemoryStats
	}{
		{
			name:      "empty processes",
			processes: []process.ProcessInfo{},
			want: &MemoryStats{
				ProcessCount: 0,
			},
		},
		{
			name: "single process",
			processes: []process.ProcessInfo{
				{PID: 1234, User: "www-data", MemoryMB: 25.5},
			},
			want: &MemoryStats{
				SmallestMB:   25.5,
				LargestMB:    25.5,
				AverageMB:    25.5,
				TotalMB:      25.5,
				ProcessCount: 1,
			},
		},
		{
			name: "multiple processes",
			processes: []process.ProcessInfo{
				{PID: 1234, User: "www-data", MemoryMB: 20.0},
				{PID: 1235, User: "www-data", MemoryMB: 30.0},
				{PID: 1236, User: "www-data", MemoryMB: 25.0},
			},
			want: &MemoryStats{
				SmallestMB:   20.0,
				LargestMB:    30.0,
				AverageMB:    25.0,
				TotalMB:      75.0,
				ProcessCount: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMemoryStats(tt.processes)
			if got.ProcessCount != tt.want.ProcessCount {
				t.Errorf("ProcessCount = %d, want %d", got.ProcessCount, tt.want.ProcessCount)
			}
			if tt.want.ProcessCount > 0 {
				if got.SmallestMB != tt.want.SmallestMB {
					t.Errorf("SmallestMB = %f, want %f", got.SmallestMB, tt.want.SmallestMB)
				}
				if got.LargestMB != tt.want.LargestMB {
					t.Errorf("LargestMB = %f, want %f", got.LargestMB, tt.want.LargestMB)
				}
				if got.AverageMB != tt.want.AverageMB {
					t.Errorf("AverageMB = %f, want %f", got.AverageMB, tt.want.AverageMB)
				}
				if got.TotalMB != tt.want.TotalMB {
					t.Errorf("TotalMB = %f, want %f", got.TotalMB, tt.want.TotalMB)
				}
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name     string
		sysInfo  *system.SystemInfo
		memStats *MemoryStats
		config   *config.ApacheConfig
		want     string // expected status
	}{
		{
			name: "no processes",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{ProcessCount: 0},
			config:   &config.ApacheConfig{MaxRequestWorkers: 256},
			want:     "ERROR",
		},
		{
			name: "safe configuration",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{
				ProcessCount: 10,
				LargestMB:    30.0,
				AverageMB:    25.0,
			},
			config: &config.ApacheConfig{MaxRequestWorkers: 40},
			want:   "OK",
		},
		{
			name: "high memory usage",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{
				ProcessCount: 10,
				LargestMB:    30.0,
				AverageMB:    25.0,
			},
			config: &config.ApacheConfig{MaxRequestWorkers: 100},
			want:   "HIGH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRecommendations(tt.sysInfo, tt.memStats, tt.config)
			if got.Status != tt.want {
				t.Errorf("Status = %s, want %s", got.Status, tt.want)
			}
		})
	}
}

func TestGenerateEnhancedRecommendations(t *testing.T) {
	tests := []struct {
		name       string
		sysInfo    *system.SystemInfo
		memStats   *MemoryStats
		config     *config.ApacheConfig
		vhostCount int
		wantStatus string
	}{
		{
			name: "optimal configuration",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     4096,
				AvailableMemoryMB: 3500,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{
				ProcessCount: 15,
				LargestMB:    35.0,
				AverageMB:    30.0,
			},
			config: &config.ApacheConfig{
				MaxRequestWorkers: 90,
				MPMModel:          "prefork",
			},
			vhostCount: 5,
			wantStatus: "OK",
		},
		{
			name: "warning configuration",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{
				ProcessCount: 10,
				LargestMB:    40.0,
				AverageMB:    35.0,
			},
			config: &config.ApacheConfig{
				MaxRequestWorkers: 35,
				MPMModel:          "prefork",
			},
			vhostCount: 3,
			wantStatus: "WARNING",
		},
		{
			name: "critical configuration",
			sysInfo: &system.SystemInfo{
				TotalMemoryMB:     1024,
				AvailableMemoryMB: 800,
				OtherServices:     make(map[string]int),
			},
			memStats: &MemoryStats{
				ProcessCount: 20,
				LargestMB:    50.0,
				AverageMB:    45.0,
			},
			config: &config.ApacheConfig{
				MaxRequestWorkers: 256,
				MPMModel:          "prefork",
			},
			vhostCount: 10,
			wantStatus: "CRITICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateEnhancedRecommendations(tt.sysInfo, tt.memStats, tt.config, nil, tt.vhostCount)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", got.Status, tt.wantStatus)
			}
			if got.CurrentMaxClients != tt.config.MaxRequestWorkers {
				t.Errorf("CurrentMaxClients = %d, want %d", got.CurrentMaxClients, tt.config.MaxRequestWorkers)
			}
			if got.RecommendedMaxClients <= 0 {
				t.Errorf("RecommendedMaxClients should be positive, got %d", got.RecommendedMaxClients)
			}
		})
	}
}
