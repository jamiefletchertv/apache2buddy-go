package config

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ApacheConfig struct {
	MaxClients        int
	MaxRequestWorkers int
	ServerLimit       int
	ThreadsPerChild   int
	MPMModel          string
	ConfigPath        string
	Version           string
	ServerName        string
}

func (c *ApacheConfig) GetCurrentMaxClients() int {
	if c.MaxRequestWorkers > 0 {
		return c.MaxRequestWorkers
	}
	if c.MaxClients > 0 {
		return c.MaxClients
	}
	return 256 // Default
}

func Parse() (*ApacheConfig, error) {
	config := &ApacheConfig{
		MPMModel: "prefork", // Default
	}

	// Find Apache config file
	configPaths := []string{
		"/etc/apache2/apache2.conf",
		"/etc/httpd/conf/httpd.conf",
		"/usr/local/apache2/conf/httpd.conf",
		"/etc/httpd/httpd.conf",
		"/etc/apache2/httpd.conf",
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return config, fmt.Errorf("Apache config file not found")
	}

	config.ConfigPath = configPath

	// Parse config file
	if err := parseConfigFile(config, configPath); err != nil {
		return config, err
	}

	// Detect MPM model
	mpm, err := detectMPMModel()
	if err == nil {
		config.MPMModel = mpm
	}

	return config, nil
}

func parseConfigFile(config *ApacheConfig, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inMPMSection := false
	currentMPM := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Handle Include directives (basic support)
		if strings.HasPrefix(line, "Include ") {
			includePath := extractIncludePath(line, filepath.Dir(filePath))
			if includePath != "" && !strings.Contains(includePath, "*") {
				parseConfigFile(config, includePath) // Recursive include
			}
			continue
		}

		// Handle MPM sections
		if strings.Contains(line, "<IfModule") {
			if strings.Contains(line, "mpm_prefork") {
				inMPMSection = true
				currentMPM = "prefork"
			} else if strings.Contains(line, "mpm_worker") {
				inMPMSection = true
				currentMPM = "worker"
			} else if strings.Contains(line, "mpm_event") {
				inMPMSection = true
				currentMPM = "event"
			}
			continue
		}

		if strings.Contains(line, "</IfModule>") {
			inMPMSection = false
			currentMPM = ""
			continue
		}

		// Parse directives
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		directive := fields[0]
		value, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		switch directive {
		case "MaxClients":
			config.MaxClients = value
			if inMPMSection {
				config.MPMModel = currentMPM
			}
		case "MaxRequestWorkers":
			config.MaxRequestWorkers = value
			if inMPMSection {
				config.MPMModel = currentMPM
			}
		case "ServerLimit":
			config.ServerLimit = value
		case "ThreadsPerChild":
			config.ThreadsPerChild = value
		}
	}

	return scanner.Err()
}

func extractIncludePath(line, baseDir string) string {
	re := regexp.MustCompile(`Include\s+(\S+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		path := matches[1]
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		return path
	}
	return ""
}

func detectMPMModel() (string, error) {
	commands := [][]string{
		{"apache2ctl", "-M"},
		{"httpd", "-M"},
		{"apachectl", "-M"},
	}

	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			continue
		}

		outputStr := strings.ToLower(string(output))
		if strings.Contains(outputStr, "mpm_prefork") {
			return "prefork", nil
		} else if strings.Contains(outputStr, "mpm_worker") {
			return "worker", nil
		} else if strings.Contains(outputStr, "mpm_event") {
			return "event", nil
		}
	}

	return "prefork", fmt.Errorf("could not detect MPM model")
}

func ParseWithVersion() (*ApacheConfig, error) {
	config, err := Parse()
	if err != nil {
		return config, err
	}

	// Detect Apache version
	version, serverName, err := detectApacheVersion()
	if err == nil {
		config.Version = version
		config.ServerName = serverName
	}

	return config, nil
}

func GetVirtualHostCount(configPath string) int {
	if configPath == "" {
		return 0
	}

	// Count VirtualHost directives
	cmd := exec.Command("grep", "-c", "<VirtualHost", configPath)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}

	return count
}

func detectApacheVersion() (string, string, error) {
	commands := [][]string{
		{"apache2", "-v"},
		{"httpd", "-v"},
		{"apache2ctl", "-v"},
		{"apachectl", "-v"},
	}

	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Server version:") {
				// Extract version and server name
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					serverInfo := parts[2] // e.g., "Apache/2.4.41"
					versionParts := strings.Split(serverInfo, "/")
					if len(versionParts) == 2 {
						return versionParts[1], versionParts[0], nil
					}
				}
			}
		}
	}

	return "unknown", "Apache", fmt.Errorf("could not detect Apache version")
}

func GetDefaults() *ApacheConfig {
	return &ApacheConfig{
		MaxClients:        256,
		MaxRequestWorkers: 256,
		MPMModel:          "prefork",
		Version:           "2.4",
		ServerName:        "Apache",
	}
}
