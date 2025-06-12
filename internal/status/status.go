package status

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"apache2buddy-go/internal/debug"
)

// ApacheStatus contains comprehensive Apache server status information
// Fields include basic metrics, extended status data, and detailed worker states
type ApacheStatus struct {
	// Basic worker counts
	ActiveWorkers int
	IdleWorkers   int

	// Performance metrics
	RequestsPerSec float64
	BytesPerSec    float64
	TotalAccesses  int
	TotalKBytes    int
	Uptime         string
	CPUUsage       float64

	// Extended status information
	ExtendedEnabled bool
	AvgRequestTime  float64
	CPULoadPercent  float64
	Load1Min        float64
	Load5Min        float64
	Load15Min       float64

	// Compatibility aliases
	TotalRequests     int     // Alias for TotalAccesses
	TotalTrafficKB    int     // Alias for TotalKBytes
	BytesPerReq       float64 // Bytes per request
	ServerVersion     string  // Apache server version
	TotalSlots        int     // Total worker slots available
	WorkersProcessing int     // Alias for ActiveWorkers

	// Detailed worker states (from scoreboard)
	WorkersRestarting int          // Workers restarting
	WorkersWaiting    int          // Workers waiting for connections
	WorkersWriting    int          // Workers writing/sending responses
	WorkersReading    int          // Workers reading requests
	WorkersKeepalive  int          // Workers in keepalive state
	WorkersClosing    int          // Workers closing connections
	WorkersLogging    int          // Workers logging
	WorkersFinishing  int          // Workers finishing gracefully
	OpenSlots         int          // Open/available worker slots
	UniqueClients     int          // Number of unique client connections
	TopClients        []ClientInfo // Top clients by activity
}

// ClientInfo represents information about a client connection
type ClientInfo struct {
	IP       string
	Requests int
	Bytes    int64
	Status   string
}

func GetApacheStatus() (*ApacheStatus, error) {
	// Try common mod_status URLs
	urls := []string{
		"http://localhost/server-status?auto",
		"http://127.0.0.1/server-status?auto",
		"http://localhost:80/server-status?auto",
	}

	var status *ApacheStatus
	var err error

	for _, url := range urls {
		if status, err = fetchStatus(url); err == nil {
			break
		}
	}

	if status == nil {
		return nil, fmt.Errorf("mod_status not accessible - enable mod_status with ExtendedStatus On")
	}

	// Try to get detailed worker status from HTML page
	if htmlContent, err := GetDetailedStatus(); err == nil {
		workerStats := ParseWorkerStatus(htmlContent)
		status.WorkersWaiting = workerStats["waiting"]
		status.WorkersReading = workerStats["reading"]
		status.WorkersWriting = workerStats["sending"]
		status.WorkersKeepalive = workerStats["keepalive"]
		status.WorkersRestarting = workerStats["starting"]
		status.WorkersClosing = workerStats["closing"]
		status.WorkersLogging = workerStats["logging"]
		status.WorkersFinishing = workerStats["graceful_finish"]
		status.OpenSlots = workerStats["open_slot"]

		// Update UniqueClients if found in HTML
		if uniqueClients := workerStats["unique_clients"]; uniqueClients > 0 {
			status.UniqueClients = uniqueClients
		}

		// Parse top clients from HTML content
		status.TopClients = parseTopClients(htmlContent)
	} else {
		// Fallback: provide reasonable defaults based on active/idle workers
		status.WorkersWaiting = status.IdleWorkers
		status.WorkersReading = 0
		status.WorkersWriting = status.ActiveWorkers
		status.WorkersKeepalive = 0
		status.WorkersRestarting = 0
		status.WorkersClosing = 0
		status.WorkersLogging = 0
		status.WorkersFinishing = 0
		status.OpenSlots = 0
		status.TopClients = make([]ClientInfo, 0) // Empty slice as fallback
	}

	return status, nil
}

func fetchStatus(url string) (*ApacheStatus, error) {
	client := &http.Client{
		Timeout: 5 * time.Second, // 5 second timeout
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			debug.Error(err, "closing response body")
		}
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseStatus(string(body))
}

func parseStatus(content string) (*ApacheStatus, error) {
	status := &ApacheStatus{}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "BusyWorkers":
			status.ActiveWorkers, _ = strconv.Atoi(value)
			status.WorkersProcessing = status.ActiveWorkers // Set alias
		case "IdleWorkers":
			status.IdleWorkers, _ = strconv.Atoi(value)
		case "ReqPerSec":
			status.RequestsPerSec, _ = strconv.ParseFloat(value, 64)
		case "BytesPerSec":
			status.BytesPerSec, _ = strconv.ParseFloat(value, 64)
		case "Total Accesses":
			status.TotalAccesses, _ = strconv.Atoi(value)
			status.TotalRequests = status.TotalAccesses // Set alias
		case "Total kBytes":
			status.TotalKBytes, _ = strconv.Atoi(value)
			status.TotalTrafficKB = status.TotalKBytes // Set alias
		case "Uptime":
			status.Uptime = value
		case "CPULoad":
			status.CPUUsage, _ = strconv.ParseFloat(value, 64)
			status.CPULoadPercent = status.CPUUsage // Set alias for compatibility
		case "Load1":
			status.Load1Min, _ = strconv.ParseFloat(value, 64)
		case "Load5":
			status.Load5Min, _ = strconv.ParseFloat(value, 64)
		case "Load15":
			status.Load15Min, _ = strconv.ParseFloat(value, 64)
		case "DurationPerReq":
			status.AvgRequestTime, _ = strconv.ParseFloat(value, 64)
		case "BytesPerReq":
			status.BytesPerReq, _ = strconv.ParseFloat(value, 64)
		case "ServerVersion":
			status.ServerVersion = value
		case "ConnsTotal":
			// Some Apache versions report total connections
			status.UniqueClients, _ = strconv.Atoi(value)
		case "ConnsAsyncWriting":
			// Additional connection info that might be useful
		case "ConnsAsyncKeepAlive":
			// Additional connection info that might be useful
		case "ConnsAsyncClosing":
			// Additional connection info that might be useful
		}
	}

	// Check if ExtendedStatus is enabled by looking for extended metrics
	if status.Load1Min > 0 || status.Load5Min > 0 || status.Load15Min > 0 ||
		status.AvgRequestTime > 0 {
		status.ExtendedEnabled = true
	}

	// Calculate derived values
	if status.BytesPerReq == 0 && status.TotalAccesses > 0 && status.TotalKBytes > 0 {
		// Calculate bytes per request if not provided directly
		totalBytes := float64(status.TotalKBytes) * 1024 // Convert KB to bytes
		status.BytesPerReq = totalBytes / float64(status.TotalAccesses)
	}

	// Calculate total slots (active + idle workers)
	status.TotalSlots = status.ActiveWorkers + status.IdleWorkers

	// If UniqueClients not provided, estimate from active workers (rough approximation)
	if status.UniqueClients == 0 {
		status.UniqueClients = status.ActiveWorkers
	}

	// Try to get server version if not provided in status
	if status.ServerVersion == "" {
		status.ServerVersion = detectServerVersion()
	}

	// If we don't have extended load data, try to parse from system load average
	if !status.ExtendedEnabled {
		parseSystemLoad(status)
	}

	// Validate we got meaningful data
	if status.ActiveWorkers == 0 && status.IdleWorkers == 0 {
		return nil, fmt.Errorf("no worker data in mod_status output")
	}

	return status, nil
}

// parseSystemLoad attempts to get system load average if not available from mod_status
func parseSystemLoad(status *ApacheStatus) {
	// Try to read /proc/loadavg for system load
	content := ""
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		content = string(data)
	}

	if content != "" {
		fields := strings.Fields(content)
		if len(fields) >= 3 {
			status.Load1Min, _ = strconv.ParseFloat(fields[0], 64)
			status.Load5Min, _ = strconv.ParseFloat(fields[1], 64)
			status.Load15Min, _ = strconv.ParseFloat(fields[2], 64)
		}
	}
}

// detectServerVersion attempts to get Apache server version
func detectServerVersion() string {
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
				// Extract version info
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return parts[2] // e.g., "Apache/2.4.41"
				}
			}
		}
	}

	return "Unknown"
}

// GetDetailedStatus gets the full HTML status page for additional analysis
func GetDetailedStatus() (string, error) {
	urls := []string{
		"http://localhost/server-status",
		"http://127.0.0.1/server-status",
	}

	for _, url := range urls {
		if content, err := fetchStatusHTML(url); err == nil {
			return content, nil
		}
	}

	return "", fmt.Errorf("detailed status not accessible")
}

func fetchStatusHTML(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			debug.Error(err, "closing response body")
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ParseWorkerStatus extracts worker status from HTML page
func ParseWorkerStatus(htmlContent string) map[string]int {
	workerStats := make(map[string]int)

	// Look for the scoreboard in various formats
	patterns := []string{
		`<pre>([._WSRKDCLGIJPkpOoNMcm\s]+)</pre>`,
		`Scoreboard Key:.*?<pre>([._WSRKDCLGIJPkpOoNMcm\s]+)</pre>`,
		`<tt>([._WSRKDCLGIJPkpOoNMcm\s]+)</tt>`,
	}

	var scoreboard string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(htmlContent)
		if len(matches) > 1 {
			scoreboard = matches[1]
			break
		}
	}

	if scoreboard != "" {
		for _, char := range scoreboard {
			switch char {
			case '_':
				workerStats["waiting"]++
			case 'S':
				workerStats["starting"]++
			case 'R':
				workerStats["reading"]++
			case 'W':
				workerStats["sending"]++
			case 'K':
				workerStats["keepalive"]++
			case 'D':
				workerStats["dns_lookup"]++
			case 'C':
				workerStats["closing"]++
			case 'L':
				workerStats["logging"]++
			case 'G':
				workerStats["graceful_finish"]++
			case 'I':
				workerStats["idle_cleanup"]++
			case 'J':
				workerStats["idle_cleanup"]++
			case 'P':
				workerStats["graceful_finish"]++
			case 'k':
				workerStats["keepalive"]++
			case 'p':
				workerStats["graceful_finish"]++
			case 'O':
				workerStats["open_slot"]++
			case 'o':
				workerStats["open_slot"]++
			case 'N':
				workerStats["open_slot"]++
			case 'M':
				workerStats["open_slot"]++
			case 'c':
				workerStats["closing"]++
			case 'm':
				workerStats["graceful_finish"]++
			case '.':
				workerStats["open_slot"]++
			case ' ':
				// Skip spaces
				continue
			default:
				// Handle any other characters as open slots
				workerStats["open_slot"]++
			}
		}
	}

	// Also try to extract additional connection info from the HTML
	if uniqueClients := extractUniqueClients(htmlContent); uniqueClients > 0 {
		workerStats["unique_clients"] = uniqueClients
	}

	return workerStats
}

// extractUniqueClients tries to extract unique client count from HTML content
func extractUniqueClients(htmlContent string) int {
	// Look for patterns like "X requests being processed, Y idle workers"
	// or connection information in the HTML
	patterns := []string{
		`(\d+)\s+requests\s+being\s+processed`,
		`Async\s+connections:\s+total:\s+(\d+)`,
		`ConnsTotal:\s+(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(htmlContent)
		if len(matches) > 1 {
			if count, err := strconv.Atoi(matches[1]); err == nil {
				return count
			}
		}
	}

	return 0
}

// parseTopClients extracts client information from Apache status HTML
func parseTopClients(htmlContent string) []ClientInfo {
	var clients []ClientInfo
	clientMap := make(map[string]*ClientInfo)

	// Look for client information in various formats
	// Pattern 1: Look for IP addresses in request lines
	ipPattern := regexp.MustCompile(`(?:GET|POST|PUT|DELETE|HEAD|OPTIONS)\s+.*?(\d+\.\d+\.\d+\.\d+)`)
	matches := ipPattern.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			ip := match[1]
			if client, exists := clientMap[ip]; exists {
				client.Requests++
			} else {
				clientMap[ip] = &ClientInfo{
					IP:       ip,
					Requests: 1,
					Bytes:    0,
					Status:   "Active",
				}
			}
		}
	}

	// Pattern 2: Look for table rows with client data
	tablePattern := regexp.MustCompile(`<tr><td[^>]*>(\d+\.\d+\.\d+\.\d+)</td><td[^>]*>(\d+)</td><td[^>]*>(\d+)</td>`)
	tableMatches := tablePattern.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range tableMatches {
		if len(match) > 3 {
			ip := match[1]
			requests, _ := strconv.Atoi(match[2])
			bytes, _ := strconv.ParseInt(match[3], 10, 64)

			clientMap[ip] = &ClientInfo{
				IP:       ip,
				Requests: requests,
				Bytes:    bytes,
				Status:   "Active",
			}
		}
	}

	// Pattern 3: Simple IP extraction from any part of the status page
	if len(clientMap) == 0 {
		simpleIPPattern := regexp.MustCompile(`\b(\d+\.\d+\.\d+\.\d+)\b`)
		ipMatches := simpleIPPattern.FindAllStringSubmatch(htmlContent, -1)

		ipCount := make(map[string]int)
		for _, match := range ipMatches {
			if len(match) > 0 {
				ip := match[1]
				// Skip local/reserved IPs for this analysis
				if !isLocalIP(ip) {
					ipCount[ip]++
				}
			}
		}

		for ip, count := range ipCount {
			clientMap[ip] = &ClientInfo{
				IP:       ip,
				Requests: count,
				Bytes:    0,
				Status:   "Detected",
			}
		}
	}

	// Convert map to slice
	for _, client := range clientMap {
		clients = append(clients, *client)
	}

	// Sort by request count (descending) and limit to top 10
	if len(clients) > 1 {
		// Simple bubble sort for small datasets
		for i := 0; i < len(clients)-1; i++ {
			for j := 0; j < len(clients)-i-1; j++ {
				if clients[j].Requests < clients[j+1].Requests {
					clients[j], clients[j+1] = clients[j+1], clients[j]
				}
			}
		}
	}

	// Limit to top 10 clients
	if len(clients) > 10 {
		clients = clients[:10]
	}

	return clients
}

// isLocalIP checks if an IP address is local/reserved
func isLocalIP(ip string) bool {
	localPatterns := []string{
		"127.",     // Loopback
		"10.",      // Private Class A
		"192.168.", // Private Class C
		"172.1",    // Private Class B (172.16-172.31)
		"172.2",
		"172.3",
		"169.254.", // Link-local
		"::1",      // IPv6 loopback
		"fe80::",   // IPv6 link-local
	}

	for _, pattern := range localPatterns {
		if strings.HasPrefix(ip, pattern) {
			return true
		}
	}

	return false
}
