package logger

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// testWriter is a custom io.Writer that captures output
type testWriter struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *testWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestNew(t *testing.T) {
	logger := New(true)
	if logger == nil {
		t.Fatal("New() returned nil")
	}
	if !logger.debugMode {
		t.Error("New() did not set debugMode correctly")
	}
}

func TestInfo(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Info("Test info message")
	if !strings.Contains(writer.String(), "Test info message") {
		t.Errorf("Info() did not output the expected message")
	}
}

func TestSuccess(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Success("Test success message")
	if !strings.Contains(writer.String(), "Test success message") {
		t.Errorf("Success() did not output the expected message")
	}
}

func TestError(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Error("Test error message")
	if !strings.Contains(writer.String(), "Test error message") {
		t.Errorf("Error() did not output the expected message")
	}
}

func TestDebug(t *testing.T) {
	logger := New(true)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Debug("Test debug message")
	if !strings.Contains(writer.String(), "Test debug message") {
		t.Errorf("Debug() did not output the expected message when in debug mode")
	}

	logger = New(false)
	writer = &testWriter{}
	logger.SetOutput(writer)

	logger.Debug("Test debug message")
	if writer.String() != "" {
		t.Errorf("Debug() output a message when not in debug mode")
	}
}

func TestPrint(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Print("Test print message")
	if !strings.Contains(writer.String(), "Test print message") {
		t.Errorf("Print() did not output the expected message")
	}
}

func TestResult(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.Result("Test result message")
	if !strings.Contains(writer.String(), "Test result message") {
		t.Errorf("Result() did not output the expected message")
	}
}

func TestCostBreakdown(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.CostBreakdown("Test cost breakdown message")
	if !strings.Contains(writer.String(), "Test cost breakdown message") {
		t.Errorf("CostBreakdown() did not output the expected message")
	}
}

func TestStartThinking(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	done := make(chan bool)
	go func() {
		logger.StartThinking()
		time.Sleep(500 * time.Millisecond)
		done <- true
	}()

	<-done
	logger.StopThinking()

	output := writer.String()
	if !strings.Contains(output, "🤔") {
		t.Errorf("StartThinking() did not output the expected thinking indicator")
	}
}

func TestStopThinking(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.StartThinking()
	time.Sleep(500 * time.Millisecond)
	logger.StopThinking()

	time.Sleep(100 * time.Millisecond)
	output := writer.String()
	finalChar := output[len(output)-1]
	if finalChar != '\r' {
		t.Errorf("StopThinking() did not clear the thinking indicator as expected. Last character: %c", finalChar)
	}
}

func TestMultipleStartThinking(t *testing.T) {
	logger := New(false)
	writer := &testWriter{}
	logger.SetOutput(writer)

	logger.StartThinking()
	time.Sleep(100 * time.Millisecond)
	logger.StartThinking() // Second call should not start another goroutine
	time.Sleep(500 * time.Millisecond)
	logger.StopThinking()

	output := writer.String()
	count := strings.Count(output, "🤔")
	if count > 7 { // Allow for some variation due to timing
		t.Errorf("Multiple StartThinking() calls resulted in too many thinking indicators. Count: %d", count)
	}
}

func TestStopThinkingWithoutStart(t *testing.T) {
	logger := New(false)
	// This should not panic or cause any errors
	logger.StopThinking()
}
