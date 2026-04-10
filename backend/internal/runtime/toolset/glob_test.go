package toolset

import "testing"

func TestMatchDoublestar(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Simple patterns (no **)
		{"*.go", "main.go", true},
		{"*.go", "main.ts", false},
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "src/sub/main.go", false},

		// ** at the beginning
		{"**/*.go", "main.go", true},
		{"**/*.go", "src/main.go", true},
		{"**/*.go", "src/sub/main.go", true},
		{"**/*.go", "main.ts", false},

		// ** in the middle
		{"src/**/*.go", "src/main.go", true},
		{"src/**/*.go", "src/sub/main.go", true},
		{"src/**/*.go", "src/a/b/c/main.go", true},
		{"src/**/*.go", "lib/main.go", false},

		// ** at the end
		{"src/**", "src/main.go", true},
		{"src/**", "src/a/b/c.txt", true},

		// Multiple **
		{"src/**/pkg/**/*.go", "src/pkg/main.go", true},
		{"src/**/pkg/**/*.go", "src/a/pkg/main.go", true},
		{"src/**/pkg/**/*.go", "src/a/b/pkg/c/d.go", true},
		{"src/**/pkg/**/*.go", "lib/pkg/main.go", false},

		// Exact match (no wildcards)
		{"README.md", "README.md", true},
		{"README.md", "src/README.md", false},

		// Question mark
		{"?.go", "a.go", true},
		{"?.go", "ab.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchDoublestar(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchDoublestar(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
