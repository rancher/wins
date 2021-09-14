package npipes

import (
	"strings"
	"testing"
)

func TestCreatePipe(t *testing.T) {
	type args struct {
		name       string
		sddl       string
		bufferSize int32
	}
	tests := []struct {
		name  string
		args  args
		want  string
		error bool
	}{
		{
			name:  "create pipe",
			args:  args{name: "//./pipe/test", sddl: "", bufferSize: 0},
			want:  "could not recognize path:",
			error: true,
		},
		{
			name:  "get path to pipe",
			args:  args{name: GetFullPath("test"), sddl: "", bufferSize: 0},
			error: false,
		},
		{
			name:  "duplicate listener",
			args:  args{name: "npipe:////./pipe/test", sddl: "", bufferSize: 0},
			want:  "Access is denied.",
			error: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.name, tt.args.sddl, tt.args.bufferSize)
			if err != nil {
				if !tt.error {
					t.Errorf("error occurred, " + err.Error())
				}
				if !strings.Contains(err.Error(), tt.want) {
					t.Errorf("error, should be " + tt.want + ", but got " + err.Error())
				}
			}
		})
	}
}
