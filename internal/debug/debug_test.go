package debug

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEnableDisable(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Test Enable
	Enable()
	if !IsEnabled() {
		t.Error("Debug should be enabled after calling Enable()")
	}

	// Test Disable
	Disable()
	if IsEnabled() {
		t.Error("Debug should be disabled after calling Disable()")
	}
}

func TestPrintf(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	// Test with debug disabled
	Disable()
	Printf("This should not appear")
	if buf.Len() > 0 {
		t.Error("Printf should not output when debug is disabled")
	}

	// Test with debug enabled
	Enable()
	buf.Reset()
	Printf("Test message: %s", "hello")
	output := buf.String()
	if !strings.Contains(output, "Test message: hello") {
		t.Errorf("Printf output should contain test message, got: %s", output)
	}
}

func TestSection(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()
	Section("TEST SECTION")
	output := buf.String()

	if !strings.Contains(output, "TEST SECTION") {
		t.Errorf("Section output should contain section title, got: %s", output)
	}
	if !strings.Contains(output, "---") {
		t.Errorf("Section output should contain separators, got: %s", output)
	}
}

func TestTimer(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	timer := StartTimer("Test Operation")
	if timer == nil {
		t.Error("StartTimer should return a non-nil timer")
	}

	// Small delay to ensure measurable time
	time.Sleep(1 * time.Millisecond)

	timer.Stop()
	output := buf.String()

	if !strings.Contains(output, "Starting timer: Test Operation") {
		t.Errorf("Timer output should contain start message, got: %s", output)
	}
	if !strings.Contains(output, "Timer 'Test Operation' completed") {
		t.Errorf("Timer output should contain completion message, got: %s", output)
	}
}

func TestTrace(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	func() {
		defer Trace("TestFunction")()
		// Function body
	}()

	output := buf.String()

	if !strings.Contains(output, "ENTER: TestFunction") {
		t.Errorf("Trace output should contain enter message, got: %s", output)
	}
	if !strings.Contains(output, "EXIT: TestFunction") {
		t.Errorf("Trace output should contain exit message, got: %s", output)
	}
}

func TestDumpMethods(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	// Test DumpStruct
	testStruct := struct {
		Name string
		Age  int
	}{"Test", 25}

	DumpStruct("TestStruct", testStruct)
	output := buf.String()
	if !strings.Contains(output, "TestStruct") {
		t.Errorf("DumpStruct should include struct name, got: %s", output)
	}

	// Test DumpSlice
	buf.Reset()
	testSlice := []string{"a", "b", "c"}
	DumpSlice("TestSlice", testSlice)
	output = buf.String()
	if !strings.Contains(output, "TestSlice") {
		t.Errorf("DumpSlice should include slice name, got: %s", output)
	}

	// Test DumpMap
	buf.Reset()
	testMap := map[string]int{"key1": 1, "key2": 2}
	DumpMap("TestMap", testMap)
	output = buf.String()
	if !strings.Contains(output, "TestMap") {
		t.Errorf("DumpMap should include map name, got: %s", output)
	}
}

func TestError(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	testErr := &testError{"test error"}
	Error(testErr, "test context")
	output := buf.String()

	if !strings.Contains(output, "ERROR in test context") {
		t.Errorf("Error output should contain context, got: %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("Error output should contain error message, got: %s", output)
	}
}

func TestWarnInfo(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	// Test Warn
	Warn("Warning message: %s", "test")
	output := buf.String()
	if !strings.Contains(output, "WARNING: Warning message: test") {
		t.Errorf("Warn output should contain warning, got: %s", output)
	}

	// Test Info
	buf.Reset()
	Info("Info message: %s", "test")
	output = buf.String()
	if !strings.Contains(output, "INFO: Info message: test") {
		t.Errorf("Info output should contain info, got: %s", output)
	}
}

func TestDumpFileInfo(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	// Test with existing file
	tempFile, err := os.CreateTemp("", "test_debug_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	DumpFileInfo(tempFile.Name())
	output := buf.String()
	if !strings.Contains(output, "Checking file:") {
		t.Errorf("DumpFileInfo should log file check, got: %s", output)
	}

	// Test with non-existing file
	buf.Reset()
	DumpFileInfo("/non/existent/file")
	output = buf.String()
	if !strings.Contains(output, "File stat error:") {
		t.Errorf("DumpFileInfo should log error for non-existent file, got: %s", output)
	}
}

func TestDumpCommandOutput(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	Enable()

	// Test successful command
	DumpCommandOutput("test", []string{"-v"}, []byte("test output"), nil)
	output := buf.String()
	if !strings.Contains(output, "Command: test -v") {
		t.Errorf("DumpCommandOutput should log command, got: %s", output)
	}
	if !strings.Contains(output, "Command success") {
		t.Errorf("DumpCommandOutput should log success, got: %s", output)
	}

	// Test failed command
	buf.Reset()
	testErr := &testError{"command failed"}
	DumpCommandOutput("test", []string{"-v"}, nil, testErr)
	output = buf.String()
	if !strings.Contains(output, "Command error: command failed") {
		t.Errorf("DumpCommandOutput should log error, got: %s", output)
	}
}

func TestDebugDisabledState(t *testing.T) {
	// Save original state
	originalState := DebugEnabled
	defer func() {
		DebugEnabled = originalState
	}()

	// Ensure debug is disabled
	Disable()

	// Capture output
	var buf bytes.Buffer
	debugLogger = log.New(&buf, "[DEBUG] ", log.LstdFlags)

	// Test that various functions don't output when disabled
	Printf("Test message")
	Section("Test Section")
	Warn("Warning")
	Info("Info")
	DumpStruct("TestStruct", struct{}{})
	DumpSlice("TestSlice", []string{})
	DumpMap("TestMap", map[string]int{})
	DumpFileInfo("/tmp")
	DumpCommandOutput("test", []string{}, []byte("output"), nil)

	if buf.Len() > 0 {
		t.Errorf("No output should be produced when debug is disabled, got: %s", buf.String())
	}
}

// Helper type for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// Benchmark tests
func BenchmarkPrintf(b *testing.B) {
	Enable()
	defer Disable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Printf("Benchmark message %d", i)
	}
}

func BenchmarkTimer(b *testing.B) {
	Enable()
	defer Disable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer := StartTimer("Benchmark")
		timer.Stop()
	}
}

func BenchmarkTrace(b *testing.B) {
	Enable()
	defer Disable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() {
			defer Trace("BenchmarkFunction")()
		}()
	}
}
