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

	"apache2buddy/internal/debug"
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
	defer debug.Trace("ApacheConfig.GetCurrentMaxClients")()
	
	if c.MaxRequestWorkers > 0 {
		debug.Printf("Using MaxRequestWorkers: %d", c.MaxRequestWorkers)
		return c.MaxRequestWorkers
	}
	if c.MaxClients > 0 {
		debug.Printf("Using MaxClients: %d", c.MaxClients)
		return c.MaxClients
	}
	debug.Printf("Using default: 256")
	return 256 // Default
}

func Parse() (*ApacheConfig, error) {
	defer debug.Trace("config.Parse")()
	
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

	debug.Printf("Searching for Apache config files...")
	var configPath string
	for _, path := range configPaths {
		debug.DumpFileInfo(path)
		if _, err := os.Stat(path); err == nil {
			configPath = path
			debug.Printf("Found Apache config: %s", path)
			break
		}
	}

	if configPath == "" {
		debug.Error(fmt.Errorf("no config file found"), "config file search")
		return config, fmt.Errorf("Apache config file not found")
	}

	config.ConfigPath = configPath

	// Parse config file
	if err := parseConfigFile(config, configPath); err != nil {
		debug.Error(err, "config file parsing")
		return config, err
	}

	// Detect MPM model
	mpm, err := detectMPMModel()
	if err == nil {
		config.MPMModel = mpm
		debug.Printf("Detected MPM model: %s", mpm)
	} else {
		debug.Warn("Could not detect MPM model: %v", err)
	}

	debug.DumpStruct("ParsedConfig", config)
	return config, nil
}

func parseConfigFile(config *ApacheConfig, filePath string) error {
	defer debug.Trace("parseConfigFile")()
	debug.Printf("Parsing config file: %s", filePath)
	
	file, err := os.Open(filePath)
	if err != nil {
		debug.Error(err, "opening config file")
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inMPMSection := false
	currentMPM := ""
	lineCount := 0
	directivesFound := 0

	for scanner.Scan() {
		lineCount++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		debug.Printf("Processing line %d: %s", lineCount, line)

		// Handle Include directives (basic support)
		if strings.HasPrefix(line, "Include ") || strings.HasPrefix(line, "IncludeOptional ") {
			includePath := extractIncludePath(line, filepath.Dir(filePath))
			debug.Printf("Found Include directive: %s -> %s", line, includePath)
			if includePath != "" && !strings.Contains(includePath, "*") {
				debug.Printf("Recursively parsing include: %s", includePath)
				parseConfigFile(config, includePath) // Recursive include
			} else if strings.Contains(includePath, "*") {
				debug.Printf("Skipping wildcard include: %s", includePath)
				// TODO: Handle wildcard includes properly
			}
			continue
		}

		// Handle MPM sections - fix the logic
		if strings.Contains(line, "<IfModule") {
			if strings.Contains(line, "mpm_prefork") && !strings.Contains(line, "!mpm_prefork") {
				inMPMSection = true
				currentMPM = "prefork"
				debug.Printf("Entering prefork MPM section")
			} else if strings.Contains(line, "mpm_worker") && !strings.Contains(line, "!mpm_worker") {
				inMPMSection = true
				currentMPM = "worker"
				debug.Printf("Entering worker MPM section")
			} else if strings.Contains(line, "mpm_event") && !strings.Contains(line, "!mpm_event") {
				inMPMSection = true
				currentMPM = "event"
				debug.Printf("Entering event MPM section")
			} else if strings.Contains(line, "!mpm_") {
				debug.Printf("Skipping negative MPM condition: %s", line)
			}
			continue
		}

		if strings.Contains(line, "</IfModule>") {
			if inMPMSection {
				debug.Printf("Exiting MPM section: %s", currentMPM)
			}
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
			debug.Printf("Could not parse value as integer: %s = %s", directive, fields[1])
			continue
		}

		directivesFound++
		debug.Printf("Found directive: %s = %d (in MPM: %s, inMPMSection: %t)", directive, value, currentMPM, inMPMSection)

		switch directive {
		case "MaxClients":
			config.MaxClients = value
			if inMPMSection {
				config.MPMModel = currentMPM
			}
			debug.Printf("Set MaxClients to %d", value)
		case "MaxRequestWorkers":
			config.MaxRequestWorkers = value
			if inMPMSection {
				config.MPMModel = currentMPM
			}
			debug.Printf("Set MaxRequestWorkers to %d", value)
		case "ServerLimit":
			config.ServerLimit = value
			debug.Printf("Set ServerLimit to %d", value)
		case "ThreadsPerChild":
			config.ThreadsPerChild = value
			debug.Printf("Set ThreadsPerChild to %d", value)
		}
	}

	debug.Printf("Config parsing complete: %d lines processed, %d directives found", lineCount, directivesFound)
	
	// If we didn't find MaxClients/MaxRequestWorkers, try to detect default values
	if config.MaxClients == 0 && config.MaxRequestWorkers == 0 {
		debug.Warn("No MaxClients or MaxRequestWorkers found in config file")
		debug.Printf("This could mean:")
		debug.Printf("1. Values are in included files not being parsed")
		debug.Printf("2. Using compiled-in defaults")
		debug.Printf("3. Values are set by the system package configuration")
		
		// Try to get defaults from Apache itself
		if defaults := tryGetApacheDefaults(config.MPMModel); defaults > 0 {
			debug.Printf("Using detected Apache defaults: %d", defaults)
			config.MaxRequestWorkers = defaults
		}
	}
	
	return scanner.Err()
}

// tryGetApacheDefaults attempts to get default values from Apache configuration
func tryGetApacheDefaults(mpmModel string) int {
	defer debug.Trace("tryGetApacheDefaults")()
	
	// Try to get defaults from httpd -V output
	commands := [][]string{
		{"httpd", "-V"},
		{"apache2", "-V"},
	}

	for _, cmd := range commands {
		debug.Printf("Trying to get defaults from: %s %s", cmd[0], strings.Join(cmd[1:], " "))
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		debug.DumpCommandOutput(cmd[0], cmd[1:], output, err)
		
		if err != nil {
			continue
		}

		outputStr := string(output)
		
		// Look for server config defaults
		if strings.Contains(outputStr, "DEFAULT_PIDLOG") {
			debug.Printf("Found Apache build configuration")
			
			// Common defaults based on MPM
			switch mpmModel {
			case "prefork":
				debug.Printf("Using prefork default: 256")
				return 256
			case "worker", "event":
				debug.Printf("Using worker/event default: 400")
				return 400
			default:
				debug.Printf("Using generic default: 256")
				return 256
			}
		}
	}

	debug.Printf("Could not determine Apache defaults")
	return 0
}

func extractIncludePath(line, baseDir string) string {
	re := regexp.MustCompile(`(?:Include(?:Optional)?)\s+(\S+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		path := matches[1]
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		debug.Printf("Extracted include path: %s", path)
		return path
	}
	return ""
}

func detectMPMModel() (string, error) {
	defer debug.Trace("detectMPMModel")()
	
	commands := [][]string{
		{"apache2ctl", "-M"},
		{"httpd", "-M"},
		{"apachectl", "-M"},
	}

	for _, cmd := range commands {
		debug.Printf("Trying command: %s %s", cmd[0], strings.Join(cmd[1:], " "))
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		debug.DumpCommandOutput(cmd[0], cmd[1:], output, err)
		
		if err != nil {
			debug.Printf("Command failed: %v", err)
			continue
		}

		outputStr := strings.ToLower(string(output))
		if strings.Contains(outputStr, "mpm_prefork") {
			debug.Printf("Detected MPM: prefork")
			return "prefork", nil
		} else if strings.Contains(outputStr, "mpm_worker") {
			debug.Printf("Detected MPM: worker")
			return "worker", nil
		} else if strings.Contains(outputStr, "mpm_event") {
			debug.Printf("Detected MPM: event")
			return "event", nil
		}
	}

	debug.Warn("Could not detect MPM model from any command")
	return "prefork", fmt.Errorf("could not detect MPM model")
}

func ParseWithVersion() (*ApacheConfig, error) {
	defer debug.Trace("ParseWithVersion")()
	
	config, err := Parse()
	if err != nil {
		return config, err
	}

	// Detect Apache version
	version, serverName, err := detectApacheVersion()
	if err == nil {
		config.Version = version
		config.ServerName = serverName
		debug.Printf("Detected Apache version: %s %s", serverName, version)
	} else {
		debug.Warn("Could not detect Apache version: %v", err)
	}

	return config, nil
}

func GetVirtualHostCount(configPath string) int {
	defer debug.Trace("GetVirtualHostCount")()
	
	if configPath == "" {
		debug.Printf("No config path provided")
		return 0
	}

	debug.Printf("Counting VirtualHost directives in: %s", configPath)
	
	// Count VirtualHost directives
	cmd := exec.Command("grep", "-c", "<VirtualHost", configPath)
	output, err := cmd.Output()
	debug.DumpCommandOutput("grep", []string{"-c", "<VirtualHost", configPath}, output, err)
	
	if err != nil {
		debug.Printf("grep command failed: %v", err)
		return 0
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		debug.Printf("Could not parse vhost count: %v", err)
		return 0
	}

	debug.Printf("Found %d virtual hosts", count)
	return count
}

func detectApacheVersion() (string, string, error) {
	defer debug.Trace("detectApacheVersion")()
	
	commands := [][]string{
		{"apache2", "-v"},
		{"httpd", "-v"},
		{"apache2ctl", "-v"},
		{"apachectl", "-v"},
	}

	for _, cmd := range commands {
		debug.Printf("Trying version detection command: %s %s", cmd[0], strings.Join(cmd[1:], " "))
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		debug.DumpCommandOutput(cmd[0], cmd[1:], output, err)
		
		if err != nil {
			debug.Printf("Version command failed: %v", err)
			continue
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			debug.Printf("Checking version line: %s", line)
			if strings.Contains(line, "Server version:") {
				// Extract version and server name
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					serverInfo := parts[2] // e.g., "Apache/2.4.41"
					versionParts := strings.Split(serverInfo, "/")
					if len(versionParts) == 2 {
						debug.Printf("Parsed version: %s %s", versionParts[0], versionParts[1])
						return versionParts[1], versionParts[0], nil
					}
				}
			}
		}
	}

	debug.Warn("Could not detect Apache version from any command")
	return "unknown", "Apache", fmt.Errorf("could not detect Apache version")
}

func GetDefaults() *ApacheConfig {
	defer debug.Trace("GetDefaults")()
	debug.Printf("Using default Apache configuration")
	
	return &ApacheConfig{
		MaxClients:        256,
		MaxRequestWorkers: 256,
		MPMModel:          "prefork",
		Version:           "2.4",
		ServerName:        "Apache",
	}
}
