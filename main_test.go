package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test parseResponse function
func TestParseResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantExplain string
	}{
		{
			name:        "standard response",
			input:       "ls -la\nLists all files including hidden ones",
			wantCommand: "ls -la",
			wantExplain: "Lists all files including hidden ones",
		},
		{
			name:        "response with empty lines",
			input:       "grep -r 'pattern' .\n\nSearches recursively for pattern",
			wantCommand: "grep -r 'pattern' .",
			wantExplain: "Searches recursively for pattern",
		},
		{
			name:        "command only",
			input:       "cd /home/user",
			wantCommand: "cd /home/user",
			wantExplain: "",
		},
		{
			name:        "multiline explanation",
			input:       "find . -type f -name '*.go'\nFinds all Go files\nRecursively searches directories",
			wantCommand: "find . -type f -name '*.go'",
			wantExplain: "Finds all Go files\nRecursively searches directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseResponse(tt.input)
			if got.Command != tt.wantCommand {
				t.Errorf("parseResponse() command = %v, want %v", got.Command, tt.wantCommand)
			}
			if got.Explanation != tt.wantExplain {
				t.Errorf("parseResponse() explanation = %v, want %v", got.Explanation, tt.wantExplain)
			}
		})
	}
}

// Test parseInteractiveLine function
func TestParseInteractiveLine(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantQuery        string
		wantOptions      ResponseOptions
		wantShowExamples bool
	}{
		{
			name:      "query only",
			input:     "how to list files",
			wantQuery: "how to list files",
			wantOptions: ResponseOptions{
				CopyToClipboard: false,
				Execute:         false,
			},
			wantShowExamples: false,
		},
		{
			name:      "query with -c flag",
			input:     "list files -c",
			wantQuery: "list files",
			wantOptions: ResponseOptions{
				CopyToClipboard: true,
				Execute:         false,
			},
			wantShowExamples: false,
		},
		{
			name:      "query with multiple flags",
			input:     "-e -c find large files -x",
			wantQuery: "find large files",
			wantOptions: ResponseOptions{
				CopyToClipboard: true,
				Execute:         true,
			},
			wantShowExamples: true,
		},
		{
			name:      "flags mixed with query",
			input:     "how -c to -e list -x files",
			wantQuery: "how to list files",
			wantOptions: ResponseOptions{
				CopyToClipboard: true,
				Execute:         true,
			},
			wantShowExamples: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotOptions, gotShowExamples := parseInteractiveLine(tt.input)
			if gotQuery != tt.wantQuery {
				t.Errorf("parseInteractiveLine() query = %v, want %v", gotQuery, tt.wantQuery)
			}
			if gotOptions != tt.wantOptions {
				t.Errorf("parseInteractiveLine() options = %v, want %v", gotOptions, tt.wantOptions)
			}
			if gotShowExamples != tt.wantShowExamples {
				t.Errorf("parseInteractiveLine() showExamples = %v, want %v", gotShowExamples, tt.wantShowExamples)
			}
		})
	}
}

// Test dangerous command detection
func TestIsDangerous(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"rm root", "rm -rf /", true},
		{"rm with force wildcard", "rm -rf *", true},
		{"dd command", "dd if=/dev/zero of=/dev/sda", true},
		{"format disk", "mkfs.ext4 /dev/sda", true},
		{"safe ls", "ls -la", false},
		{"safe grep", "grep -r 'pattern' .", false},
		{"fork bomb", ":(){ :|:& };:", true},
		{"redirect to device", "> /dev/sda", true},
		{"mv to dev null", "mv important.file /dev/null", true},
		{"safe rm", "rm -f important.txt", false}, // not in patterns
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDangerous(tt.command); got != tt.want {
				t.Errorf("isDangerous(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// Test config file operations
func TestConfigFileOperations(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigDir)

	// Test saving config
	t.Run("save config", func(t *testing.T) {
		config := FileConfig{
			Provider:     "anthropic",
			AnthropicKey: "test-key",
			OpenAIKey:    "openai-test-key",
		}

		err := saveConfigFile(config)
		if err != nil {
			t.Fatalf("saveConfigFile() error = %v", err)
		}

		// Check if file exists
		configPath := filepath.Join(tempDir, "howtfdoi", configFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Config file was not created")
		}

		// Check permissions
		info, _ := os.Stat(configPath)
		if info.Mode().Perm() != 0600 {
			t.Errorf("Config file permissions = %v, want 0600", info.Mode().Perm())
		}

		// Check .gitignore was created
		gitignorePath := filepath.Join(tempDir, "howtfdoi", ".gitignore")
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			t.Fatal(".gitignore file was not created")
		}
	})

	// Test loading config
	t.Run("load config", func(t *testing.T) {
		config := loadConfigFile()
		if config.Provider != "anthropic" {
			t.Errorf("loaded Provider = %v, want anthropic", config.Provider)
		}
		if config.AnthropicKey != "test-key" {
			t.Errorf("loaded AnthropicKey = %v, want test-key", config.AnthropicKey)
		}
	})

	// Test loading non-existent config
	t.Run("load non-existent config", func(t *testing.T) {
		os.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "nonexistent"))
		config := loadConfigFile()
		if config.Provider != "" {
			t.Errorf("Expected empty config, got %+v", config)
		}
	})
}

// Test getDataDirectory
func TestGetDataDirectory(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantDir string
		goos    string
	}{
		{
			name:    "XDG_STATE_HOME set",
			envVar:  "/custom/state",
			wantDir: "/custom/state/howtfdoi",
			goos:    "linux",
		},
		{
			name:    "default Linux",
			envVar:  "",
			wantDir: filepath.Join(os.Getenv("HOME"), ".local", "state", "howtfdoi"),
			goos:    "linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env var
			oldXDG := os.Getenv("XDG_STATE_HOME")
			os.Setenv("XDG_STATE_HOME", tt.envVar)
			defer os.Setenv("XDG_STATE_HOME", oldXDG)

			got := getDataDirectory()
			if got != tt.wantDir && tt.envVar == "" {
				// For empty env var, just check it contains expected suffix
				if !strings.HasSuffix(got, filepath.Join(".local", "state", "howtfdoi")) {
					t.Errorf("getDataDirectory() = %v, want suffix %v", got, filepath.Join(".local", "state", "howtfdoi"))
				}
			} else if tt.envVar != "" && got != tt.wantDir {
				t.Errorf("getDataDirectory() = %v, want %v", got, tt.wantDir)
			}
		})
	}
}

// Test provider creation with mock
type mockProvider struct {
	responses []string
	index     int
}

func (m *mockProvider) Query(ctx context.Context, query, platform string, examples bool) (string, error) {
	if m.index >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func TestProviderQuery(t *testing.T) {
	mock := &mockProvider{
		responses: []string{"ls -la\nList all files including hidden"},
	}

	ctx := context.Background()
	resp, err := mock.Query(ctx, "list files", "darwin", false)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if resp != "ls -la\nList all files including hidden" {
		t.Errorf("Query() = %v, want ls -la\\nList all files including hidden", resp)
	}
}

// Test history file operations
func TestHistoryLogging(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, "history.log")

	config := Config{
		HistoryFile: historyFile,
	}

	// Create a response to test with
	response := &Response{
		Command:     "test command",
		Explanation: "test explanation",
		FullText:    "test command\ntest explanation",
	}

	// Test handleResponse writes to history
	opts := ResponseOptions{
		CopyToClipboard: false,
		Execute:         false,
	}

	// Capture stdout to avoid test output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleResponse(config, "test query", response, opts)

	w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	// Read back and verify
	content, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "test query") {
		t.Errorf("History file doesn't contain query")
	}
	if !strings.Contains(string(content), "test command") {
		t.Errorf("History file doesn't contain command")
	}
	if !strings.Contains(string(content), "---") {
		t.Errorf("History file doesn't contain separator")
	}
}

// Test handleResponse function
func TestHandleResponse(t *testing.T) {
	// Create temp directory for history
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, "history.log")

	config := Config{
		HistoryFile: historyFile,
		Verbose:     false,
	}

	response := &Response{
		Command:     "echo 'test'",
		Explanation: "Prints test",
		FullText:    "echo 'test'\nPrints test",
	}

	options := ResponseOptions{
		CopyToClipboard: false,
		Execute:         false,
	}

	// Just test that it doesn't panic and writes to history
	handleResponse(config, "test query", response, options)

	// Check that history was written
	content, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatalf("Failed to read history file: %v", err)
	}

	if !strings.Contains(string(content), "test query") {
		t.Errorf("History doesn't contain query")
	}
	if !strings.Contains(string(content), "echo 'test'") {
		t.Errorf("History doesn't contain command")
	}
}

// Benchmark response parsing
func BenchmarkParseResponse(b *testing.B) {
	response := "find . -type f -name '*.go' -exec grep -l 'pattern' {} \\;\nFinds all Go files containing 'pattern'\nThis searches recursively through directories"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseResponse(response)
	}
}

// Benchmark dangerous command checking
func BenchmarkIsDangerous(b *testing.B) {
	commands := []string{
		"ls -la",
		"rm -rf /",
		"grep -r 'pattern' .",
		"dd if=/dev/zero of=/dev/sda",
		"find . -type f",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cmd := range commands {
			isDangerous(cmd)
		}
	}
}

// Test timeout handling for API calls
func TestQueryTimeout(t *testing.T) {
	// This would test context timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Mock provider that takes too long
	mock := &mockProvider{
		responses: []string{"timeout test"},
	}

	// In a real test, you'd want the provider to respect context
	_, err := mock.Query(ctx, "test", "darwin", false)
	if err == nil {
		// Note: This is a simplified test - real implementation would check context
		t.Skip("Timeout test requires context-aware mock")
	}
}

// Benchmark stripMarkdown function
func BenchmarkStripMarkdown(b *testing.B) {
	input := "```bash\nls -la\n```\nThis is a command with **bold** and `code`"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripMarkdown(input)
	}
}

// Benchmark provider comparison
func BenchmarkProviderComparison(b *testing.B) {
	// Note: This is a mock benchmark since we can't actually call APIs in tests
	// In a real benchmark, you'd use a test server or record/replay

	providers := map[string]*mockProvider{
		"anthropic": &mockProvider{
			responses: []string{"ls -la\nList all files including hidden"},
		},
		"openai": &mockProvider{
			responses: []string{"ls -la\nList all files including hidden"},
		},
		"lmstudio": &mockProvider{
			responses: []string{"ls -la\nList all files including hidden"},
		},
	}

	ctx := context.Background()

	for name, provider := range providers {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				provider.index = 0 // Reset for each iteration
				_, _ = provider.Query(ctx, "list files", "darwin", false)
			}
		})
	}
}
