package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// TranscriptionResult represents a single transcription result
type TranscriptionResult struct {
	Index      int       `json:"index"`
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Partial    bool      `json:"partial"`
	Type       string    `json:"type,omitempty"`
}

// Event represents a system event
type Event struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Formatter is the interface for output formatters
type Formatter interface {
	// WriteResult writes a transcription result
	WriteResult(result TranscriptionResult) error

	// WritePartial writes a partial (in-progress) result
	WritePartial(text string) error

	// WriteEvent writes a system event (e.g., VAD state changes)
	WriteEvent(eventType, message string) error

	// Flush ensures all buffered output is written
	Flush() error

	// Close closes the formatter and releases resources
	Close() error
}

// JSONFormatter outputs transcriptions in JSON format
type JSONFormatter struct {
	writer  io.Writer
	encoder *json.Encoder
	results []TranscriptionResult
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(writer io.Writer) *JSONFormatter {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	return &JSONFormatter{
		writer:  writer,
		encoder: encoder,
		results: make([]TranscriptionResult, 0),
	}
}

// WriteResult writes a transcription result in JSON format
func (j *JSONFormatter) WriteResult(result TranscriptionResult) error {
	if !result.Partial {
		// Only store final results
		j.results = append(j.results, result)
	}

	// Write individual result immediately
	return j.encoder.Encode(result)
}

// WritePartial writes a partial result
func (j *JSONFormatter) WritePartial(text string) error {
	result := TranscriptionResult{
		Text:      text,
		Timestamp: time.Now(),
		Partial:   true,
	}
	return j.encoder.Encode(result)
}

// WriteEvent writes a system event
func (j *JSONFormatter) WriteEvent(eventType, message string) error {
	event := Event{
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
	}
	return j.encoder.Encode(event)
}

// Flush ensures all buffered output is written
func (j *JSONFormatter) Flush() error {
	// JSON encoder writes immediately, nothing to flush
	return nil
}

// Close closes the formatter
func (j *JSONFormatter) Close() error {
	// Optionally write a summary
	return nil
}

// GetResults returns all final transcription results
func (j *JSONFormatter) GetResults() []TranscriptionResult {
	return j.results
}

// PlainTextFormatter outputs transcriptions in plain text format
type PlainTextFormatter struct {
	writer io.Writer
}

// NewPlainTextFormatter creates a new plain text formatter
func NewPlainTextFormatter(writer io.Writer) *PlainTextFormatter {
	return &PlainTextFormatter{
		writer: writer,
	}
}

// WriteResult writes a transcription result in plain text
func (p *PlainTextFormatter) WriteResult(result TranscriptionResult) error {
	if result.Partial {
		return nil // Don't write partial results in plain text mode
	}

	timestamp := result.Timestamp.Format("15:04:05")
	text := fmt.Sprintf("[%s] %s\n", timestamp, result.Text)

	_, err := p.writer.Write([]byte(text))
	return err
}

// WritePartial writes a partial result (no-op for plain text)
func (p *PlainTextFormatter) WritePartial(text string) error {
	return nil
}

// WriteEvent writes a system event
func (p *PlainTextFormatter) WriteEvent(eventType, message string) error {
	timestamp := time.Now().Format("15:04:05")
	text := fmt.Sprintf("[%s] [%s] %s\n", timestamp, eventType, message)
	_, err := p.writer.Write([]byte(text))
	return err
}

// Flush ensures all buffered output is written
func (p *PlainTextFormatter) Flush() error {
	return nil
}

// Close closes the formatter
func (p *PlainTextFormatter) Close() error {
	return nil
}
