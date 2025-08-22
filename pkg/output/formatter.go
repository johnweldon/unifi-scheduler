// Package output provides flexible output formatting capabilities for the unifi-scheduler CLI.
//
// This package supports multiple output formats including human-readable tables,
// machine-readable JSON, and configuration-friendly YAML. It's designed to make
// CLI output suitable for both interactive use and automation scripts.
//
// Supported formats:
//   - table: Human-readable tabular output (default)
//   - json: Machine-readable JSON format
//   - yaml: Configuration-friendly YAML format
//
// Example usage:
//
//	opts, err := output.NewOutputOptions("json", os.Stdout)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	formatter := opts.CreateFormatter()
//	err = formatter.Write(data)
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents a supported output format for CLI commands.
//
// The format determines how data is serialized and presented to the user.
// Each format serves different use cases:
//   - FormatTable: Interactive use with readable columns
//   - FormatJSON: Automation and programmatic consumption
//   - FormatYAML: Configuration files and human-readable data
type Format string

const (
	// FormatTable represents human-readable tabular output format.
	// This is the default format and provides nicely formatted tables
	// with columns, headers, and visual separators.
	FormatTable Format = "table"

	// FormatJSON represents machine-readable JSON output format.
	// This format is ideal for automation, scripting, and integration
	// with other tools that can parse JSON.
	FormatJSON Format = "json"

	// FormatYAML represents human-readable YAML output format.
	// This format is suitable for configuration files and provides
	// a clean, indented structure that's easy to read and edit.
	FormatYAML Format = "yaml"
)

// ParseFormat parses a string representation into a Format type.
//
// This function accepts the format name and various aliases for convenience:
//   - "table", "tab", "t" -> FormatTable
//   - "json", "js", "j" -> FormatJSON
//   - "yaml", "yml", "y" -> FormatYAML
//
// The parsing is case-insensitive. If the input string doesn't match
// any known format, an error is returned with a list of supported formats.
//
// Example:
//
//	format, err := ParseFormat("json")
//	if err != nil {
//	    log.Fatalf("Invalid format: %v", err)
//	}
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

// Formatter handles the serialization and output of data in different formats.
//
// A Formatter encapsulates the logic for converting Go data structures into
// formatted output. It supports multiple output formats and writes the result
// to the configured io.Writer.
//
// The Formatter is designed to be reusable - you can call Write multiple times
// with different data structures using the same format and writer.
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new Formatter with the specified format and writer.
//
// The format parameter determines how data will be serialized (table, JSON, or YAML).
// The writer parameter specifies where the formatted output will be written.
//
// Example:
//
//	formatter := output.NewFormatter(output.FormatJSON, os.Stdout)
//	err := formatter.Write(myData)
func NewFormatter(format Format, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// Write serializes and outputs data in the formatter's configured format.
//
// The data parameter can be any Go value that can be serialized by the chosen format:
//   - For JSON: any value supported by encoding/json
//   - For YAML: any value supported by gopkg.in/yaml.v3
//   - For table: types implementing TableWriter interface, or fallback to string representation
//
// Returns an error if the data cannot be serialized in the specified format
// or if writing to the underlying io.Writer fails.
//
// Example:
//
//	data := map[string]interface{}{
//	    "name": "example",
//	    "count": 42,
//	}
//	err := formatter.Write(data)
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

// writeJSON serializes data as indented JSON and writes it to the formatter's writer.
// The output is formatted with 2-space indentation for readability.
func (f *Formatter) writeJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// writeYAML serializes data as YAML and writes it to the formatter's writer.
// The output is formatted with 2-space indentation for readability.
func (f *Formatter) writeYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// writeTable attempts to write data in tabular format.
// If the data implements the TableWriter interface, it uses that method.
// Otherwise, it falls back to a simple string representation.
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

// TableWriter is an interface for types that can render themselves as formatted tables.
//
// Types implementing this interface can provide custom table formatting logic
// instead of relying on the default string representation. This is particularly
// useful for complex data structures that benefit from columnar presentation.
//
// Example implementation:
//
//	func (c ClientList) WriteTable(w io.Writer) {
//	    table := tablewriter.NewWriter(w)
//	    table.SetHeader([]string{"Name", "IP", "Status"})
//	    for _, client := range c {
//	        table.Append([]string{client.Name, client.IP, client.Status})
//	    }
//	    table.Render()
//	}
type TableWriter interface {
	// WriteTable writes a formatted table representation to the provided writer.
	WriteTable(io.Writer)
}

// OutputOptions contains configuration parameters for output formatting.
//
// This structure combines a format specification with an output destination,
// providing a convenient way to configure and create formatters. It serves
// as a bridge between command-line arguments and formatter creation.
type OutputOptions struct {
	// Format specifies how data should be serialized (table, JSON, or YAML)
	Format Format
	// Writer specifies where the formatted output should be written
	Writer io.Writer
}

// NewOutputOptions creates OutputOptions from a format string and writer.
//
// This is a convenience constructor that parses the format string and
// validates it before creating the OutputOptions. It's the recommended
// way to create OutputOptions from user input.
//
// Parameters:
//   - formatStr: A string representation of the desired format ("table", "json", "yaml", etc.)
//   - writer: The io.Writer where formatted output will be written
//
// Returns an error if the format string is not recognized.
//
// Example:
//
//	opts, err := output.NewOutputOptions("json", os.Stdout)
//	if err != nil {
//	    log.Fatalf("Invalid format: %v", err)
//	}
//	formatter := opts.CreateFormatter()
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

// CreateFormatter creates a new Formatter using the options' configuration.
//
// This method provides a convenient way to create a formatter that's
// properly configured with the format and writer from the OutputOptions.
// The returned formatter is ready to use for data serialization.
//
// Example:
//
//	opts := &OutputOptions{Format: FormatJSON, Writer: os.Stdout}
//	formatter := opts.CreateFormatter()
//	err := formatter.Write(myData)
func (o *OutputOptions) CreateFormatter() *Formatter {
	return NewFormatter(o.Format, o.Writer)
}
