package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunConvert(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "heading produces valid bike",
			input:      "# Hello",
			wantOutput: `data-type="heading"`,
			wantErr:    false,
		},
		{
			name:       "xml declaration present",
			input:      "# Test",
			wantOutput: `<?xml version="1.0" encoding="UTF-8"?>`,
			wantErr:    false,
		},
		{
			name:       "paragraph produces body row",
			input:      "Just text",
			wantOutput: `<p>Just text</p>`,
			wantErr:    false,
		},
		{
			name:       "empty input produces valid document",
			input:      "",
			wantOutput: `<html>`,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			opts := convertOptions{}

			err := runConvert(input, stdout, stderr, opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("runConvert() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := stdout.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output does not contain %q\ngot:\n%s", tt.wantOutput, output)
			}
		})
	}
}
