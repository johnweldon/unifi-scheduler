package output

import (
	"bytes"
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

