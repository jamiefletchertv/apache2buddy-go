package debug

import (
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	DebugEnabled = false
	debugLogger  *log.Logger
)

func init() {
	// Create a debug logger that prefixes with [DEBUG]
	debugLogger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
}

// Enable turns on debug mode
func Enable() {
	DebugEnabled = true
	Printf("Debug mode enabled")
}

// Disable turns off debug mode
func Disable() {
	DebugEnabled = false
}

// IsEnabled returns whether debug mode is active
func IsEnabled() bool {
	return DebugEnabled
}

// Printf prints debug information if debug mode is enabled
func Printf(format string, args ...interface{}) {
	if DebugEnabled {
		debugLogger.Printf(format, args...)
	}
}

// Println prints debug information if debug mode is enabled
func Println(args ...interface{}) {
	if DebugEnabled {
		debugLogger.Println(args...)
	}
}

// Section prints a debug section header
func Section(title string) {
	if DebugEnabled {
		separator := strings.Repeat("-", len(title)+8)
		debugLogger.Printf("\n%s", separator)
		debugLogger.Printf("--- %s ---", title)
		debugLogger.Printf("%s\n", separator)
	}
}

// Timer helps measure execution time of operations
type Timer struct {
	name      string
	startTime time.Time
}

// StartTimer creates and starts a new timer
func StartTimer(name string) *Timer {
	if DebugEnabled {
		debugLogger.Printf("Starting timer: %s", name)
	}
	return &Timer{
		name:      name,
		startTime: time.Now(),
	}
}

// Stop ends the timer and prints the elapsed time
func (t *Timer) Stop() {
	if DebugEnabled {
		elapsed := time.Since(t.startTime)
		debugLogger.Printf("Timer '%s' completed in %v", t.name, elapsed)
	}
}

// DumpSystemInfo prints detailed system debugging information
func DumpSystemInfo() {
	if !DebugEnabled {
		return
	}

	Section("SYSTEM DEBUG INFO")

	// Go runtime info
	Printf("Go Version: %s", runtime.Version())
	Printf("GOOS: %s", runtime.GOOS)
	Printf("GOARCH: %s", runtime.GOARCH)
	Printf("NumCPU: %d", runtime.NumCPU())
	Printf("NumGoroutine: %d", runtime.NumGoroutine())

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	Printf("Alloc: %d KB", m.Alloc/1024)
	Printf("TotalAlloc: %d KB", m.TotalAlloc/1024)
	Printf("Sys: %d KB", m.Sys/1024)

	// Process info
	Printf("PID: %d", os.Getpid())
	Printf("PPID: %d", os.Getppid())
	Printf("UID: %d", os.Getuid())
	Printf("GID: %d", os.Getgid())

	// Environment
	Printf("USER: %s", os.Getenv("USER"))
	Printf("HOME: %s", os.Getenv("HOME"))
	Printf("PATH: %s", os.Getenv("PATH"))
}

// DumpFileInfo prints debug info about a file
func DumpFileInfo(filepath string) {
	if !DebugEnabled {
		return
	}

	Printf("Checking file: %s", filepath)

	if info, err := os.Stat(filepath); err != nil {
		Printf("File stat error: %v", err)
	} else {
		Printf("File exists: size=%d, mode=%s, modtime=%s",
			info.Size(), info.Mode(), info.ModTime())
	}

	if info, err := os.Lstat(filepath); err != nil {
		Printf("Lstat error: %v", err)
	} else if info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(filepath); err != nil {
			Printf("Readlink error: %v", err)
		} else {
			Printf("Symlink target: %s", target)
		}
	}
}

// DumpCommandOutput prints debug info about command execution
func DumpCommandOutput(command string, args []string, output []byte, err error) {
	if !DebugEnabled {
		return
	}

	Printf("Command: %s %s", command, strings.Join(args, " "))
	if err != nil {
		Printf("Command error: %v", err)
	} else {
		Printf("Command success, output length: %d bytes", len(output))
		if len(output) > 0 {
			// Show first 200 characters of output
			preview := string(output)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			Printf("Command output preview: %s", preview)
		}
	}
}

// DumpSlice prints debug info about a slice
func DumpSlice(name string, slice interface{}) {
	if !DebugEnabled {
		return
	}

	Printf("Slice '%s': %+v", name, slice)
}

// DumpStruct prints debug info about a struct
func DumpStruct(name string, obj interface{}) {
	if !DebugEnabled {
		return
	}

	Printf("Struct '%s': %+v", name, obj)
}

// DumpMap prints debug info about a map
func DumpMap(name string, m interface{}) {
	if !DebugEnabled {
		return
	}

	Printf("Map '%s': %+v", name, m)
}

// Error prints an error with debug context
func Error(err error, context string) {
	if DebugEnabled && err != nil {
		debugLogger.Printf("ERROR in %s: %v", context, err)

		// Print stack trace for debugging
		buf := make([]byte, 1024)
		n := runtime.Stack(buf, false)
		Printf("Stack trace:\n%s", buf[:n])
	}
}

// Warn prints a warning message
func Warn(format string, args ...interface{}) {
	if DebugEnabled {
		debugLogger.Printf("WARNING: "+format, args...)
	}
}

// Info prints an informational debug message
func Info(format string, args ...interface{}) {
	if DebugEnabled {
		debugLogger.Printf("INFO: "+format, args...)
	}
}

// Trace prints function entry/exit information
func Trace(funcName string) func() {
	if DebugEnabled {
		Printf("ENTER: %s", funcName)
		start := time.Now()
		return func() {
			Printf("EXIT: %s (took %v)", funcName, time.Since(start))
		}
	}
	return func() {} // No-op if debug disabled
}
