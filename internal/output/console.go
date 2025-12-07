package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ConsoleOutput handles outputting transcriptions to the console
type ConsoleOutput struct {
	mu            sync.Mutex
	writer        io.Writer
	showTimestamp bool
	showMetadata  bool
}

// ConsoleConfig configures console output behavior
type ConsoleConfig struct {
	// ShowTimestamp prefixes each line with a timestamp
	ShowTimestamp bool

	// ShowMetadata displays additional metadata (confidence, etc.)
	ShowMetadata bool

	// Writer is the output destination (default: os.Stdout)
	Writer io.Writer
}

// NewConsoleOutput creates a new console output handler
func NewConsoleOutput(config ConsoleConfig) *ConsoleOutput {
	writer := config.Writer
	if writer == nil {
		writer = os.Stdout
	}

	return &ConsoleOutput{
		writer:        writer,
		showTimestamp: config.ShowTimestamp,
		showMetadata:  config.ShowMetadata,
	}
}

// DefaultConsoleOutput creates a console output with default settings
func DefaultConsoleOutput() *ConsoleOutput {
	return NewConsoleOutput(ConsoleConfig{
		ShowTimestamp: true,
		ShowMetadata:  false,
		Writer:        os.Stdout,
	})
}

// Write writes a transcription result to the console
func (c *ConsoleOutput) Write(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.showTimestamp {
		timestamp := time.Now().Format("15:04:05")
		fmt.Fprintf(c.writer, "[%s] %s\n", timestamp, text)
	} else {
		fmt.Fprintf(c.writer, "%s\n", text)
	}

	return nil
}

// WriteWithMetadata writes a transcription with additional metadata
func (c *ConsoleOutput) WriteWithMetadata(text string, confidence float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	timestamp := ""
	if c.showTimestamp {
		timestamp = fmt.Sprintf("[%s] ", time.Now().Format("15:04:05"))
	}

	metadata := ""
	if c.showMetadata {
		metadata = fmt.Sprintf(" (confidence: %.2f)", confidence)
	}

	fmt.Fprintf(c.writer, "%s%s%s\n", timestamp, text, metadata)
	return nil
}

// WritePartial writes a partial transcription (work-in-progress)
// This typically overwrites the current line
func (c *ConsoleOutput) WritePartial(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use carriage return to overwrite the current line
	fmt.Fprintf(c.writer, "\r%s", text)
	return nil
}

// Finalize finalizes a partial transcription (adds newline)
func (c *ConsoleOutput) Finalize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Fprintln(c.writer)
	return nil
}

// WriteAudioLevel writes the current audio level (for visualization)
func (c *ConsoleOutput) WriteAudioLevel(level float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a simple bar visualization
	barLength := int(level * 50) // 50 chars max
	if barLength > 50 {
		barLength = 50
	}

	bar := ""
	for i := 0; i < barLength; i++ {
		bar += "="
	}

	fmt.Fprintf(c.writer, "\rLevel: [%-50s] %.1f%%", bar, level*100)
	return nil
}

// Clear clears the current line
func (c *ConsoleOutput) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Fprintf(c.writer, "\r%80s\r", " ") // Clear line
	return nil
}

// Info writes an informational message
func (c *ConsoleOutput) Info(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Fprintf(c.writer, "[INFO] %s\n", msg)
}

// Error writes an error message to stderr
func (c *ConsoleOutput) Error(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
}

// Status writes a status message (typically overwritten)
func (c *ConsoleOutput) Status(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Fprintf(c.writer, "\r[*] %s", msg)
}
