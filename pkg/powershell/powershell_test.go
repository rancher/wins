package powershell

import (
	"strings"
	"testing"
)

func TestExecuteCommand(t *testing.T) {
	type args struct {
		command string
	}

	tests := []struct {
		name  string
		args  args
		want  string
		error bool
	}{
		{
			name:  "write to standard out",
			args:  args{command: "Write-Output 'test-value'"},
			want:  "test-value",
			error: false,
		},
		{
			name:  "write to host",
			args:  args{command: "Write-Host 'test-value'"},
			want:  "test-value",
			error: false,
		},
		{
			name:  "write to standard error",
			args:  args{command: "Write-Error 'test-value'"},
			want:  "test-value",
			error: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := RunCommand(tt.args.command)
			if err != nil && !tt.error {
				t.Errorf("error occurred, %s", err.Error())
			}

			if !tt.error {
				got := strings.TrimSpace(string(o))
				if got != tt.want {
					t.Errorf("error, should be %s, but got %s ", tt.want, got)
				}
			}
		})
	}
}
