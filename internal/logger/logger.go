package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Logger provides simple, colorful logging for Klama
type Logger struct {
	debugMode bool
	thinking  bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	out       io.Writer
}

// New creates a new Logger instance
func New(debugMode bool) *Logger {
	return &Logger{
		debugMode: debugMode,
		stopChan:  make(chan struct{}),
		out:       os.Stdout,
	}
}

// SetOutput sets the output destination for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
}

// Info prints an informational message
func (l *Logger) Info(format string, a ...interface{}) {
	fmt.Fprintf(l.out, "%s %s\n", color.BlueString("ℹ"), fmt.Sprintf(format, a...))
}

// Success prints a success message
func (l *Logger) Success(format string, a ...interface{}) {
	fmt.Fprintf(l.out, "%s %s\n", color.GreenString("✔"), fmt.Sprintf(format, a...))
}

// Error prints an error message
func (l *Logger) Error(format string, a ...interface{}) {
	fmt.Fprintf(l.out, "%s %s\n", color.RedString("✖"), fmt.Sprintf(format, a...))
}

// Debug prints a debug message (only in debug mode)
func (l *Logger) Debug(format string, a ...interface{}) {
	if l.debugMode {
		fmt.Fprintf(l.out, "%s %s\n", color.YellowString("➤"), fmt.Sprintf(format, a...))
	}
}

// Print prints a plain message without any prefix
func (l *Logger) Print(format string, a ...interface{}) {
	fmt.Fprintf(l.out, format+"\n", a...)
}

// Result prints a result message with a distinct symbol
func (l *Logger) Result(format string, a ...interface{}) {
	fmt.Fprintf(l.out, "%s %s\n", "🔍", fmt.Sprintf(format, a...))
}

// CostBreakdown prints a cost breakdown message with a distinct symbol
func (l *Logger) CostBreakdown(format string, a ...interface{}) {
	fmt.Fprintf(l.out, "%s %s\n", "💰", fmt.Sprintf(format, a...))
}

// StartThinking starts the thinking indication
func (l *Logger) StartThinking() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.thinking {
		return
	}
	l.thinking = true
	l.stopChan = make(chan struct{})
	l.wg.Add(1)

	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		chars := []rune(`-\|/`)
		for {
			select {
			case <-l.stopChan:
				fmt.Fprint(l.out, "\r \r") // Clear the thinking indication
				return
			case <-ticker.C:
				fmt.Fprintf(l.out, "\r%s %c", "🤔", chars[i%len(chars)])
				i++
			}
		}
	}()
}

// StopThinking stops the thinking indication
func (l *Logger) StopThinking() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.thinking {
		return
	}
	close(l.stopChan)
	l.wg.Wait()
	l.thinking = false
}

// EmptyLogger returns a logger that discards all output
func EmptyLogger() *Logger {
	return &Logger{
		out: io.Discard,
	}
}
