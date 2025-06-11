package system

import (
	"testing"
)

func TestSystemInfo_TotalOtherServicesMemory(t *testing.T) {
	tests := []struct {
		name     string
		sysInfo  *SystemInfo
		expected int
	}{
		{
			name: "no services",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			expected: 0,
		},
		{
			name: "single service",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices: map[string]int{
					"MySQL": 200,
				},
			},
			expected: 200,
		},
		{
			name: "multiple services",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     4096,
				AvailableMemoryMB: 3000,
				OtherServices: map[string]int{
					"MySQL":     300,
					"Redis":     50,
					"PHP-FPM":   150,
					"Memcached": 100,
				},
			},
			expected: 600,
		},
		{
			name: "services with special markers",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices: map[string]int{
					"MySQL":        200,
					"PHP-FPM":      150,
					"PHP-FPM-Note": -1, // Special marker should be ignored
				},
			},
			expected: 350,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTotalOtherServicesMemory(tt.sysInfo)
			if result != tt.expected {
				t.Errorf("GetTotalOtherServicesMemory() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDetectControlPanels(t *testing.T) {
	// This test is environment-dependent, so we mainly test the function exists
	// and returns a string (empty or with a control panel name)
	result := DetectControlPanels()
	
	// Should return a string (could be empty if no control panels are detected)
	if result != "" {
		t.Logf("Control panel detected: %s", result)
	} else {
		t.Log("No control panel detected")
	}
}

func TestSystemInfoValidation(t *testing.T) {
	tests := []struct {
		name    string
		sysInfo *SystemInfo
		valid   bool
	}{
		{
			name: "valid system info",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			valid: true,
		},
		{
			name: "available memory greater than total",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     1024,
				AvailableMemoryMB: 2048,
				OtherServices:     make(map[string]int),
			},
			valid: false,
		},
		{
			name: "zero total memory",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     0,
				AvailableMemoryMB: 1500,
				OtherServices:     make(map[string]int),
			},
			valid: false,
		},
		{
			name: "negative memory values",
			sysInfo: &SystemInfo{
				TotalMemoryMB:     -1,
				AvailableMemoryMB: -500,
				OtherServices:     make(map[string]int),
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic validation logic
			isValid := tt.sysInfo.TotalMemoryMB > 0 && 
					  tt.sysInfo.AvailableMemoryMB >= 0 &&
					  tt.sysInfo.AvailableMemoryMB <= tt.sysInfo.TotalMemoryMB

			if isValid != tt.valid {
				t.Errorf("System info validation = %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestServiceMemoryCalculation(t *testing.T) {
	// Test the logic for calculating service memory
	tests := []struct {
		name     string
		services map[string]int
		expected int
	}{
		{
			name:     "empty services",
			services: map[string]int{},
			expected: 0,
		},
		{
			name: "normal services",
			services: map[string]int{
				"MySQL":   200,
				"Redis":   50,
				"PHP-FPM": 150,
			},
			expected: 400,
		},
		{
			name: "services with zero memory",
			services: map[string]int{
				"MySQL": 200,
				"Redis": 0,
				"Nginx": 100,
			},
			expected: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysInfo := &SystemInfo{
				TotalMemoryMB:     2048,
				AvailableMemoryMB: 1500,
				OtherServices:     tt.services,
			}
			
			result := GetTotalOtherServicesMemory(sysInfo)
			if result != tt.expected {
				t.Errorf("Service memory calculation = %d, want %d", result, tt.expected)
			}
		})
	}
}

// Benchmark test for performance
func BenchmarkGetTotalOtherServicesMemory(b *testing.B) {
	sysInfo := &SystemInfo{
		TotalMemoryMB:     4096,
		AvailableMemoryMB: 3000,
		OtherServices: map[string]int{
			"MySQL":     300,
			"Redis":     50,
			"PHP-FPM":   150,
			"Memcached": 100,
			"Nginx":     80,
			"Postfix":   30,
			"Java":      500,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTotalOtherServicesMemory(sysInfo)
	}
}

func BenchmarkDetectControlPanels(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectControlPanels()
	}
}