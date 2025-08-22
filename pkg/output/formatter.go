package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents an output format
type Format string

const (
	// FormatTable represents table output format
	FormatTable Format = "table"
	// FormatJSON represents JSON output format
	FormatJSON Format = "json"
	// FormatYAML represents YAML output format
	FormatYAML Format = "yaml"
)

// ParseFormat parses a string into a Format
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "tab", "t":
		return FormatTable, nil
	case "json", "js", "j":
		return FormatJSON, nil
	case "yaml", "yml", "y":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unsupported output format: %s (supported: table, json, yaml)", s)
	}
}

// Formatter handles different output formats
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new formatter
func NewFormatter(format Format, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// Write outputs data in the configured format
func (f *Formatter) Write(data interface{}) error {
	switch f.format {
	case FormatJSON:
		return f.writeJSON(data)
	case FormatYAML:
		return f.writeYAML(data)
	case FormatTable:
		return f.writeTable(data)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

func (f *Formatter) writeJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (f *Formatter) writeYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

func (f *Formatter) writeTable(data interface{}) error {
	// For table format, we expect the data to implement a specific interface
	// or we fall back to a simple string representation
	if tableWriter, ok := data.(TableWriter); ok {
		tableWriter.WriteTable(f.writer)
		return nil
	}

	// Fallback to string representation
	_, err := fmt.Fprintf(f.writer, "%v\n", data)
	return err
}

// TableWriter interface for types that can render themselves as tables
type TableWriter interface {
	WriteTable(io.Writer)
}

// OutputOptions contains configuration for output formatting
type OutputOptions struct {
	Format Format
	Writer io.Writer
}

// NewOutputOptions creates OutputOptions from a format string
func NewOutputOptions(formatStr string, writer io.Writer) (*OutputOptions, error) {
	format, err := ParseFormat(formatStr)
	if err != nil {
		return nil, err
	}

	return &OutputOptions{
		Format: format,
		Writer: writer,
	}, nil
}

// CreateFormatter creates a formatter from the options
func (o *OutputOptions) CreateFormatter() *Formatter {
	return NewFormatter(o.Format, o.Writer)
}
