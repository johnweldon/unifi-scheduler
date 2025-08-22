package output

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		wantErr  bool
	}{
		{"table", FormatTable, false},
		{"tab", FormatTable, false},
		{"t", FormatTable, false},
		{"json", FormatJSON, false},
		{"js", FormatJSON, false},
		{"j", FormatJSON, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"y", FormatYAML, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatter_WriteJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewFormatter(FormatJSON, buf)

	data := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	err := formatter.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name": "test"`) {
		t.Errorf("Expected JSON output to contain name field, got: %s", output)
	}
	if !strings.Contains(output, `"id": 123`) {
		t.Errorf("Expected JSON output to contain id field, got: %s", output)
	}
}

func TestFormatter_WriteYAML(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewFormatter(FormatYAML, buf)

	data := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	err := formatter.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name: test") {
		t.Errorf("Expected YAML output to contain name field, got: %s", output)
	}
	if !strings.Contains(output, "id: 123") {
		t.Errorf("Expected YAML output to contain id field, got: %s", output)
	}
}

func TestNewOutputOptions(t *testing.T) {
	buf := &bytes.Buffer{}

	opts, err := NewOutputOptions("json", buf)
	if err != nil {
		t.Fatalf("NewOutputOptions() error = %v", err)
	}

	if opts.Format != FormatJSON {
		t.Errorf("Expected format JSON, got %v", opts.Format)
	}

	if opts.Writer != buf {
		t.Errorf("Expected writer to be set correctly")
	}

	formatter := opts.CreateFormatter()
	if formatter.format != FormatJSON {
		t.Errorf("Expected formatter format JSON, got %v", formatter.format)
	}
}

func TestNewOutputOptions_InvalidFormat(t *testing.T) {
	buf := &bytes.Buffer{}

	_, err := NewOutputOptions("invalid", buf)
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestFormatter_WriteTable(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewFormatter(FormatTable, buf)

	// Test with data that implements TableWriter interface
	tableData := &mockTableWriter{content: "test table data"}
	err := formatter.Write(tableData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if output != "test table data\n" {
		t.Errorf("Expected table output to be 'test table data\\n', got: %q", output)
	}
}

func TestFormatter_WriteTable_Fallback(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewFormatter(FormatTable, buf)

	// Test with data that doesn't implement TableWriter interface
	data := "simple string data"
	err := formatter.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if output != "simple string data\n" {
		t.Errorf("Expected fallback output to be 'simple string data\\n', got: %q", output)
	}
}

func TestFormatter_WriteUnsupportedFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &Formatter{format: "unsupported", writer: buf}

	err := formatter.Write("test data")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected error message to contain 'unsupported format', got: %v", err)
	}
}

// mockTableWriter implements the TableWriter interface for testing
type mockTableWriter struct {
	content string
}

func (m *mockTableWriter) WriteTable(w io.Writer) {
	w.Write([]byte(m.content + "\n"))
}
