package toolset

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePath(t *testing.T) {
	workDir := filepath.Clean("/workspace")

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		want      string
		linuxOnly bool // some absolute-path tests only make sense on Linux
	}{
		{
			name:  "relative path inside workspace",
			input: "src/main.go",
			want:  filepath.Join(workDir, "src/main.go"),
		},
		{
			name:  "current directory",
			input: ".",
			want:  workDir,
		},
		{
			name:    "relative path traversal escapes workspace",
			input:   "../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "dotdot in middle escapes workspace",
			input:   "src/../../etc/passwd",
			wantErr: true,
		},
		{
			name:  "dotdot that stays inside workspace",
			input: "src/../main.go",
			want:  filepath.Join(workDir, "main.go"),
		},
		{
			name:  "nested directory",
			input: "a/b/c/d.txt",
			want:  filepath.Join(workDir, "a/b/c/d.txt"),
		},
		// Absolute path tests only valid on Linux (where /etc is truly absolute)
		{
			name:      "absolute path outside workspace (Linux)",
			input:     "/etc/passwd",
			wantErr:   true,
			linuxOnly: true,
		},
		{
			name:      "absolute path inside workspace (Linux)",
			input:     "/workspace/src/main.go",
			want:      filepath.Join(workDir, "src/main.go"),
			linuxOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.linuxOnly && runtime.GOOS != "linux" {
				t.Skipf("skipping Linux-only test on %s", runtime.GOOS)
			}

			got, err := resolvePath(workDir, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for path %q, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("resolvePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
