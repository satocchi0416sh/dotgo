package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeEngine(t *testing.T) {
	tests := []struct {
		name    string
		dryRun  bool
		verbose bool
		setup   func() error
		cleanup func()
		wantErr bool
	}{
		{
			name:    "successful initialization",
			dryRun:  false,
			verbose: false,
			setup:   nil,
			cleanup: nil,
			wantErr: false,
		},
		{
			name:    "successful initialization with dry-run",
			dryRun:  true,
			verbose: false,
			setup:   nil,
			cleanup: nil,
			wantErr: false,
		},
		{
			name:    "successful initialization with verbose",
			dryRun:  false,
			verbose: true,
			setup:   nil,
			cleanup: nil,
			wantErr: false,
		},
		// Note: It's difficult to make os.Getwd() fail in a controlled way
		// without causing issues for the test runner itself.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current directory
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)

			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			eng, err := InitializeEngine(tt.dryRun, tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitializeEngine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && eng == nil {
				t.Error("InitializeEngine() returned nil engine without error")
			}
		})
	}
}

func TestProcessTags(t *testing.T) {
	tests := []struct {
		name     string
		rawTags  []string
		expected []string
	}{
		{
			name:     "empty slice returns nil",
			rawTags:  []string{},
			expected: nil,
		},
		{
			name:     "nil slice returns nil",
			rawTags:  nil,
			expected: nil,
		},
		{
			name:     "single tag without whitespace",
			rawTags:  []string{"work"},
			expected: []string{"work"},
		},
		{
			name:     "single tag with leading whitespace",
			rawTags:  []string{" work"},
			expected: []string{"work"},
		},
		{
			name:     "single tag with trailing whitespace",
			rawTags:  []string{"work "},
			expected: []string{"work"},
		},
		{
			name:     "single tag with surrounding whitespace",
			rawTags:  []string{"  work  "},
			expected: []string{"work"},
		},
		{
			name:     "multiple tags with mixed whitespace",
			rawTags:  []string{" work ", "personal", "  linux  ", "darwin "},
			expected: []string{"work", "personal", "linux", "darwin"},
		},
		{
			name:     "tags with only whitespace become empty",
			rawTags:  []string{"  ", "work", "   "},
			expected: []string{"", "work", ""},
		},
		{
			name:     "tags with tabs and newlines",
			rawTags:  []string{"\twork\n", " personal\t"},
			expected: []string{"work", "personal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessTags(tt.rawTags)

			// Check nil cases
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ProcessTags() = %v, expected nil", result)
				}
				return
			}

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("ProcessTags() returned %d tags, expected %d", len(result), len(tt.expected))
				return
			}

			// Check each tag
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ProcessTags()[%d] = %q, expected %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkProcessTags(b *testing.B) {
	tags := []string{" work ", " personal ", " linux ", " darwin ", " common "}
	for i := 0; i < b.N; i++ {
		ProcessTags(tags)
	}
}

func BenchmarkInitializeEngine(b *testing.B) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitializeEngine(false, false)
	}
}

// Test helper to ensure working directory is valid after tests
func TestMain(m *testing.M) {
	// Save the original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	// Ensure we're back in the original directory
	if currentDir, _ := os.Getwd(); currentDir != originalDir {
		os.Chdir(originalDir)
	}

	os.Exit(code)
}

// Table-driven test for combined scenarios
func TestInitializeEngine_Combinations(t *testing.T) {
	scenarios := []struct {
		dryRun  bool
		verbose bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}

	for _, s := range scenarios {
		name := filepath.Join("dryRun", boolStr(s.dryRun), "verbose", boolStr(s.verbose))
		t.Run(name, func(t *testing.T) {
			eng, err := InitializeEngine(s.dryRun, s.verbose)
			if err != nil {
				t.Errorf("InitializeEngine(%v, %v) failed: %v", s.dryRun, s.verbose, err)
			}
			if eng == nil {
				t.Error("InitializeEngine() returned nil engine")
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
