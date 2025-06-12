package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"apache2buddy-go/internal/debug"
)

type SystemInfo struct {
	TotalMemoryMB     int
	AvailableMemoryMB int
	OtherServices     map[string]int // service name -> memory MB
}

func CheckRequiredCommands() error {
	commands := []string{"ps", "pmap"}
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("command '%s' not found", cmd)
		}
	}
	return nil
}

func GetInfo() (*SystemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc/meminfo: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			debug.Error(err, "closing /proc/meminfo")
		}
	}()

	var totalKB, availableKB, freeKB, buffersKB, cachedKB int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			totalKB, _ = strconv.Atoi(fields[1])
		case "MemAvailable:":
			availableKB, _ = strconv.Atoi(fields[1])
		case "MemFree:":
			freeKB, _ = strconv.Atoi(fields[1])
		case "Buffers:":
			buffersKB, _ = strconv.Atoi(fields[1])
		case "Cached:":
			cachedKB, _ = strconv.Atoi(fields[1])
		}
	}

	// Calculate available memory if not provided
	if availableKB == 0 {
		availableKB = freeKB + buffersKB + cachedKB
	}

	if totalKB == 0 {
		return nil, fmt.Errorf("could not determine total memory")
	}

	return &SystemInfo{
		TotalMemoryMB:     totalKB / 1024,
		AvailableMemoryMB: availableKB / 1024,
		OtherServices:     make(map[string]int),
	}, nil
}

func DetectServices(sysInfo *SystemInfo) {
	services := map[string][]string{
		"MySQL":     {"mysqld", "mariadb"},
		"Redis":     {"redis-server"},
		"Memcached": {"memcached"},
		"PHP-FPM":   {"php-fpm", "php5-fpm", "php7.0-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm"},
		"Nginx":     {"nginx"},
		"Varnish":   {"varnishd"},
		"Java":      {"java"},
		"Postfix":   {"postfix", "master"},
	}

	for serviceName, processNames := range services {
		totalMemory := 0.0
		for _, procName := range processNames {
			if memory, err := getServiceMemory(procName); err == nil {
				totalMemory += memory
			}
		}
		if totalMemory > 0 {
			sysInfo.OtherServices[serviceName] = int(totalMemory)
		}
	}
}

func getServiceMemory(serviceName string) (float64, error) {
	cmd := exec.Command("ps", "-C", serviceName, "-o", "rss=", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return 0, fmt.Errorf("no processes found")
	}

	var totalMemoryKB float64
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if memKB, err := strconv.ParseFloat(line, 64); err == nil {
			totalMemoryKB += memKB
		}
	}

	return totalMemoryKB / 1024, nil // Convert to MB
}

func DetectControlPanels() string {
	controlPanels := map[string]string{
		"/usr/local/cpanel":      "cPanel",
		"/usr/local/psa":         "Plesk",
		"/etc/webmin":            "Webmin",
		"/usr/local/directadmin": "DirectAdmin",
	}

	for path, name := range controlPanels {
		if _, err := os.Stat(path); err == nil {
			return name
		}
	}

	return ""
}

func DetectPHPFPM(sysInfo *SystemInfo, mpmModel string) {
	// Enhanced PHP-FPM detection with MPM-specific warnings
	phpfpmMemory := 0.0
	phpfpmProcesses := []string{"php-fpm", "php5-fpm", "php7.0-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm"}

	for _, procName := range phpfpmProcesses {
		if memory, err := getServiceMemory(procName); err == nil {
			phpfpmMemory += memory
		}
	}

	if phpfpmMemory > 0 {
		sysInfo.OtherServices["PHP-FPM"] = int(phpfpmMemory)

		// Add special note for worker/event MPM
		if mpmModel == "worker" || mpmModel == "event" {
			sysInfo.OtherServices["PHP-FPM-Note"] = -1 // Special marker for display
		}
	}
}

func GetTotalOtherServicesMemory(sysInfo *SystemInfo) int {
	total := 0
	for service, memory := range sysInfo.OtherServices {
		if service != "PHP-FPM-Note" { // Skip special markers
			total += memory
		}
	}
	return total
}
