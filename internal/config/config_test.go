package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApacheConfig_GetCurrentMaxClients(t *testing.T) {
	tests := []struct {
		name   string
		config *ApacheConfig
		want   int
	}{
		{
			name: "MaxRequestWorkers set",
			config: &ApacheConfig{
				MaxRequestWorkers: 150,
				MaxClients:        100,
			},
			want: 150,
		},
		{
			name: "Only MaxClients set",
			config: &ApacheConfig{
				MaxRequestWorkers: 0,
				MaxClients:        100,
			},
			want: 100,
		},
		{
			name: "Neither set",
			config: &ApacheConfig{
				MaxRequestWorkers: 0,
				MaxClients:        0,
			},
			want: 256, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCurrentMaxClients()
			if got != tt.want {
				t.Errorf("GetCurrentMaxClients() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseConfigFile(t *testing.T) {
	// Create temporary config files for testing
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		filename string
		want     *ApacheConfig
		wantErr  bool
	}{
		{
			name: "basic prefork config",
			content: `
<IfModule mpm_prefork_module>
    MaxRequestWorkers 150
    ServerLimit 150
</IfModule>`,
			filename: "apache_prefork.conf",
			want: &ApacheConfig{
				MaxRequestWorkers: 150,
				ServerLimit:       150,
				MPMModel:          "prefork",
			},
		},
		{
			name: "worker config",
			content: `
<IfModule mpm_worker_module>
    MaxRequestWorkers 400
    ThreadsPerChild 25
    ServerLimit 16
</IfModule>`,
			filename: "apache_worker.conf",
			want: &ApacheConfig{
				MaxRequestWorkers: 400,
				ThreadsPerChild:   25,
				ServerLimit:       16,
				MPMModel:          "worker",
			},
		},
		{
			name: "legacy MaxClients",
			content: `
<IfModule mpm_prefork_module>
    MaxClients 256
    ServerLimit 256
</IfModule>`,
			filename: "apache_legacy.conf",
			want: &ApacheConfig{
				MaxClients:  256,
				ServerLimit: 256,
				MPMModel:    "prefork",
			},
		},
		{
			name: "mixed directives",
			content: `
LoadModule rewrite_module modules/mod_rewrite.so
<IfModule mpm_prefork_module>
    StartServers 8
    MaxRequestWorkers 200
    MinSpareServers 5
    MaxSpareServers 20
</IfModule>
ServerName example.com`,
			filename: "apache_mixed.conf",
			want: &ApacheConfig{
				MaxRequestWorkers: 200,
				MPMModel:          "prefork",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test config file
			configPath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			config := &ApacheConfig{MPMModel: "prefork"} // Default
			err = parseConfigFile(config, configPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config.MaxRequestWorkers != tt.want.MaxRequestWorkers {
					t.Errorf("MaxRequestWorkers = %d, want %d", config.MaxRequestWorkers, tt.want.MaxRequestWorkers)
				}
				if config.MaxClients != tt.want.MaxClients {
					t.Errorf("MaxClients = %d, want %d", config.MaxClients, tt.want.MaxClients)
				}
				if config.ServerLimit != tt.want.ServerLimit {
					t.Errorf("ServerLimit = %d, want %d", config.ServerLimit, tt.want.ServerLimit)
				}
				if config.ThreadsPerChild != tt.want.ThreadsPerChild {
					t.Errorf("ThreadsPerChild = %d, want %d", config.ThreadsPerChild, tt.want.ThreadsPerChild)
				}
				if tt.want.MPMModel != "" && config.MPMModel != tt.want.MPMModel {
					t.Errorf("MPMModel = %s, want %s", config.MPMModel, tt.want.MPMModel)
				}
			}
		})
	}
}

func TestGetDefaults(t *testing.T) {
	config := GetDefaults()

	if config.MaxClients != 256 {
		t.Errorf("Default MaxClients = %d, want 256", config.MaxClients)
	}
	if config.MaxRequestWorkers != 256 {
		t.Errorf("Default MaxRequestWorkers = %d, want 256", config.MaxRequestWorkers)
	}
	if config.MPMModel != "prefork" {
		t.Errorf("Default MPMModel = %s, want prefork", config.MPMModel)
	}
	if config.Version != "2.4" {
		t.Errorf("Default Version = %s, want 2.4", config.Version)
	}
	if config.ServerName != "Apache" {
		t.Errorf("Default ServerName = %s, want Apache", config.ServerName)
	}
}

func TestExtractIncludePath(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		baseDir string
		want    string
	}{
		{
			name:    "absolute path",
			line:    "Include /etc/apache2/sites-enabled/*.conf",
			baseDir: "/etc/apache2",
			want:    "/etc/apache2/sites-enabled/*.conf",
		},
		{
			name:    "relative path",
			line:    "Include conf.d/*.conf",
			baseDir: "/etc/apache2",
			want:    "/etc/apache2/conf.d/*.conf",
		},
		{
			name:    "IncludeOptional",
			line:    "IncludeOptional /etc/apache2/mods-enabled/*.load",
			baseDir: "/etc/apache2",
			want:    "/etc/apache2/mods-enabled/*.load",
		},
		{
			name:    "no match",
			line:    "LoadModule rewrite_module modules/mod_rewrite.so",
			baseDir: "/etc/apache2",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIncludePath(tt.line, tt.baseDir)
			if got != tt.want {
				t.Errorf("extractIncludePath() = %s, want %s", got, tt.want)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkParseConfigFile(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench.conf")

	content := `
<IfModule mpm_prefork_module>
    StartServers 8
    MinSpareServers 5
    MaxSpareServers 20
    MaxRequestWorkers 256
    MaxConnectionsPerChild 10000
</IfModule>

<IfModule mpm_worker_module>
    StartServers 2
    MinSpareThreads 25
    MaxSpareThreads 75
    ThreadsPerChild 25
    MaxRequestWorkers 150
    MaxConnectionsPerChild 10000
</IfModule>
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &ApacheConfig{MPMModel: "prefork"}
		_ = parseConfigFile(config, configPath)
	}
}

func BenchmarkGetCurrentMaxClients(b *testing.B) {
	config := &ApacheConfig{
		MaxRequestWorkers: 150,
		MaxClients:        100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.GetCurrentMaxClients()
	}
}
