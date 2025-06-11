package process

import (
	"testing"
)

func TestProcessInfo_Validation(t *testing.T) {
	tests := []struct {
		name    string
		process ProcessInfo
		valid   bool
	}{
		{
			name: "valid process",
			process: ProcessInfo{
				PID:      1234,
				User:     "www-data",
				MemoryMB: 25.5,
			},
			valid: true,
		},
		{
			name: "invalid PID",
			process: ProcessInfo{
				PID:      0,
				User:     "www-data",
				MemoryMB: 25.5,
			},
			valid: false,
		},
		{
			name: "negative memory",
			process: ProcessInfo{
				PID:      1234,
				User:     "www-data",
				MemoryMB: -5.0,
			},
			valid: false,
		},
		{
			name: "empty user",
			process: ProcessInfo{
				PID:      1234,
				User:     "",
				MemoryMB: 25.5,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic validation logic
			isValid := tt.process.PID > 0 && 
					  tt.process.User != "" &&
					  tt.process.MemoryMB >= 0

			if isValid != tt.valid {
				t.Errorf("Process validation = %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestIsApacheProcess(t *testing.T) {
	tests := []struct {
		name string
		comm string
		want bool
	}{
		{
			name: "httpd process",
			comm: "httpd",
			want: true,
		},
		{
			name: "apache2 process",
			comm: "apache2",
			want: true,
		},
		{
			name: "httpd.worker process",
			comm: "httpd.worker",
			want: true,
		},
		{
			name: "httpd-prefork process",
			comm: "httpd-prefork",
			want: true,
		},
		{
			name: "nginx process",
			comm: "nginx",
			want: false,
		},
		{
			name: "mysql process",
			comm: "mysqld",
			want: false,
		},
		{
			name: "empty command",
			comm: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isApacheProcess(tt.comm)
			if got != tt.want {
				t.Errorf("isApacheProcess(%s) = %v, want %v", tt.comm, got, tt.want)
			}
		})
	}
}

func TestIsApacheCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "httpd command",
			command: "/usr/sbin/httpd -D FOREGROUND",
			want:    true,
		},
		{
			name:    "apache2 command",
			command: "/usr/sbin/apache2 -k start",
			want:    true,
		},
		{
			name:    "local apache",
			command: "/usr/local/apache2/bin/httpd",
			want:    true,
		},
		{
			name:    "nginx command",
			command: "/usr/sbin/nginx -g daemon off;",
			want:    false,
		},
		{
			name:    "apache2buddy command - should be excluded",
			command: "./apache2buddy-go -debug",
			want:    false,
		},
		{
			name:    "mysql command",
			command: "/usr/sbin/mysqld",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isApacheCommand(tt.command)
			if got != tt.want {
				t.Errorf("isApacheCommand(%s) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestParseAuxFormat(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    int // expected number of processes
		wantErr bool
	}{
		{
			name: "standard ps aux output",
			output: `USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root      1234  0.0  0.1  12345  6789 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND
www-data  1235  0.0  0.2  12345  7890 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND
www-data  1236  0.0  0.1  12345  6543 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND`,
			want:    2, // Only www-data processes, not root
			wantErr: false,
		},
		{
			name: "busybox ps output",
			output: `  PID USER     TIME  COMMAND
 1234 root     0:00 /usr/sbin/httpd -D FOREGROUND
 1235 www-data 0:00 /usr/sbin/httpd -D FOREGROUND
 1236 www-data 0:00 /usr/sbin/httpd -D FOREGROUND`,
			want:    2, // Only www-data processes
			wantErr: false,
		},
		{
			name: "mixed processes",
			output: `USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root      1234  0.0  0.1  12345  6789 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND
www-data  1235  0.0  0.2  12345  7890 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND
mysql     1236  0.0  0.1  12345  6543 ?        S    10:00   0:00 /usr/sbin/mysqld
www-data  1237  0.0  0.1  12345  6543 ?        S    10:00   0:00 /usr/sbin/httpd -D FOREGROUND`,
			want:    2, // Only www-data httpd processes
			wantErr: false,
		},
		{
			name: "no apache processes",
			output: `USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
mysql     1236  0.0  0.1  12345  6543 ?        S    10:00   0:00 /usr/sbin/mysqld
redis     1237  0.0  0.1  12345  6543 ?        S    10:00   0:00 /usr/bin/redis-server`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "empty output",
			output:  "",
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAuxFormat(tt.output)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAuxFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if len(got) != tt.want {
				t.Errorf("parseAuxFormat() returned %d processes, want %d", len(got), tt.want)
			}
			
			// Validate that all returned processes are valid
			for _, proc := range got {
				if proc.PID <= 0 {
					t.Errorf("Invalid PID: %d", proc.PID)
				}
				if proc.User == "" {
					t.Errorf("Empty user for PID %d", proc.PID)
				}
				if proc.User == "root" {
					t.Errorf("Root process should be filtered out: PID %d", proc.PID)
				}
				if proc.MemoryMB < 0 {
					t.Errorf("Negative memory for PID %d: %f", proc.PID, proc.MemoryMB)
				}
			}
		})
	}
}

func TestParsePmapOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   float64
	}{
		{
			name: "standard pmap output",
			output: `1234:   /usr/sbin/httpd -D FOREGROUND
mapped: 123456K    writeable/private: 123456K    shared: 12345K`,
			want: 120.5625, // 123456K converted to MB
		},
		{
			name: "alternative format",
			output: `1234:   /usr/sbin/httpd -D FOREGROUND
mapped: 123456K    writable-private: 123456K    shared: 12345K`,
			want: 120.5625, // 123456K converted to MB
		},
		{
			name: "no writeable memory",
			output: `1234:   /usr/sbin/httpd -D FOREGROUND
mapped: 123456K    shared: 12345K`,
			want: 0,
		},
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePmapOutput(tt.output)
			if got != tt.want {
				t.Errorf("parsePmapOutput() = %f, want %f", got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkIsApacheProcess(b *testing.B) {
	commands := []string{"httpd", "apache2", "nginx", "mysqld", "httpd.worker"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cmd := range commands {
			isApacheProcess(cmd)
		}
	}
}

func BenchmarkIsApacheCommand(b *testing.B) {
	commands := []string{
		"/usr/sbin/httpd -D FOREGROUND",
		"/usr/sbin/apache2 -k start",
		"/usr/sbin/nginx -g daemon off;",
		"./apache2buddy-go -debug",
		"/usr/local/apache2/bin/httpd",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cmd := range commands {
			isApacheCommand(cmd)
		}
	}
}