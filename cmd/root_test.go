package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "help flag shows usage",
			args:       []string{"--help"},
			wantOutput: "bikemark converts between GitHub-flavored Markdown",
			wantErr:    false,
		},
		{
			name:       "version command",
			args:       []string{"version"},
			wantOutput: "dev",
			wantErr:    false,
		},
		{
			name:       "version --full",
			args:       []string{"version", "--full"},
			wantOutput: "version: dev",
			wantErr:    false,
		},
		{
			name:    "unknown flag errors",
			args:    []string{"--badFlag"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Execute() output = %q, want to contain %q", output, tt.wantOutput)
			}
		})
	}
}

func TestHelpShowsFlags(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	_ = cmd.Execute()
	output := buf.String()

	flags := []string{"-m, --markdown", "-b, --bike"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("help output missing %q:\n%s", flag, output)
		}
	}
}
