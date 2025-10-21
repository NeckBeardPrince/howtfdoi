package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/atotto/clipboard"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	openai "github.com/sashabaranov/go-openai"
)

const (
	// Model configuration
	claudeModel = anthropic.ModelClaudeHaiku4_5
	gptModel    = "gpt-4o-mini"
	maxTokens   = 1024

	// UI thresholds
	aliasLengthThreshold = 40
	aliasPipeThreshold   = 1

	// History file name
	historyFileName = ".howtfdoi_history"

	// Provider types
	providerAnthropic = "anthropic"
	providerOpenAI    = "openai"
	providerChatGPT   = "chatgpt" // alias for openai
)

var (
	// version is set at build time via -ldflags
	version = "dev"
	// repository URL for documentation and downloads
	repository = "https://github.com/NeckBeardPrince/howtfdoi"
)

var (
	// Dangerous command patterns (compiled once at startup)
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`rm\s+-rf\s+/`),
		regexp.MustCompile(`rm\s+-rf\s+\*`),
		regexp.MustCompile(`dd\s+.*of=/dev/`),
		regexp.MustCompile(`mkfs\.`),
		regexp.MustCompile(`:(){ :|:& };:`),
		regexp.MustCompile(`>\s*/dev/sd`),
		regexp.MustCompile(`mv\s+.*\s+/dev/null`),
	}

	// Regex for sanitizing alias names (compiled once at startup)
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// Config holds runtime configuration
type Config struct {
	APIKey      string
	HistoryFile string
	Platform    string
	Verbose     bool
	Provider    string // "anthropic" or "openai"
}

// Response holds the parsed response
type Response struct {
	Command     string
	Explanation string
	FullText    string
}

// ResponseOptions holds options for processing responses
type ResponseOptions struct {
	CopyToClipboard bool
	Execute         bool
}

// Provider defines the interface for AI providers (Anthropic, OpenAI, etc.)
type Provider interface {
	// Query sends a query to the AI provider and returns the response text
	Query(ctx context.Context, systemPrompt, userQuery string) (string, error)
}

// AnthropicProvider implements Provider for Anthropic's Claude API
type AnthropicProvider struct {
	client anthropic.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

// Query sends a query to Anthropic's API
func (p *AnthropicProvider) Query(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     claudeModel,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: systemPrompt,
				// Enable prompt caching for the system prompt
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: "user",
				Content: []anthropic.ContentBlockParamUnion{
					anthropic.NewTextBlock(userQuery),
				},
			},
		},
	})

	var fullResponse strings.Builder
	for stream.Next() {
		event := stream.Current()
		if event.Type == "content_block_delta" {
			contentDelta := event.AsContentBlockDelta()
			textDelta := contentDelta.Delta.AsTextDelta()
			fullResponse.WriteString(textDelta.Text)
		}
	}

	if err := stream.Err(); err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}

// OpenAIProvider implements Provider for OpenAI's API
type OpenAIProvider struct {
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
	}
}

// Query sends a query to OpenAI's API
func (p *OpenAIProvider) Query(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     gptModel,
		MaxTokens: maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userQuery,
			},
		},
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullResponse strings.Builder
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}

		if len(response.Choices) > 0 {
			fullResponse.WriteString(response.Choices[0].Delta.Content)
		}
	}

	return fullResponse.String(), nil
}

func main() {
	// Customize help output to include version information
	flag.Usage = func() {
		fmt.Printf("howtfdoi version %s\n", version)
		fmt.Printf("Download and documentation: %s\n\n", repository)

		fmt.Fprintf(os.Stderr, "Ask CLI questions in plain English and get instant answers powered by AI.\n\n")

		fmt.Fprintf(os.Stderr, "GETTING STARTED:\n")
		fmt.Fprintf(os.Stderr, "  1. Set up your API key (choose one):\n")
		fmt.Fprintf(os.Stderr, "     • For Claude:   export ANTHROPIC_API_KEY='your-key-here'\n")
		fmt.Fprintf(os.Stderr, "     • For ChatGPT:  export OPENAI_API_KEY='your-key-here'\n\n")
		fmt.Fprintf(os.Stderr, "  2. Ask a question:\n")
		fmt.Fprintf(os.Stderr, "     howtfdoi compress a directory\n\n")

		fmt.Fprintf(os.Stderr, "USAGE:\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi [flags] <query>\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi              (interactive mode)\n\n")

		fmt.Fprintf(os.Stderr, "FLAGS:\n")
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, "\nENVIRONMENT VARIABLES:\n")
		fmt.Fprintf(os.Stderr, "  ANTHROPIC_API_KEY     Your Anthropic API key (get it at console.anthropic.com)\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY        Your OpenAI API key (get it at platform.openai.com)\n")
		fmt.Fprintf(os.Stderr, "  HOWTFDOI_AI_PROVIDER  Override provider choice: anthropic, openai, or chatgpt\n")
		fmt.Fprintf(os.Stderr, "                        (defaults to anthropic, or auto-detects from available keys)\n")

		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi list files\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi find large files over 100MB\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi -c compress a directory    # copy to clipboard\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi -e tar                     # show examples\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi -x git commit              # execute with confirmation\n")
		fmt.Fprintf(os.Stderr, "  HOWTFDOI_AI_PROVIDER=openai howtfdoi list files\n\n")
	}

	// Parse flags
	versionFlag := flag.Bool("version", false, "Show version information")
	verboseFlag := flag.Bool("v", false, "Enable verbose logging")
	copyFlag := flag.Bool("c", false, "Copy command to clipboard")
	executeFlag := flag.Bool("x", false, "Execute the command directly")
	examplesFlag := flag.Bool("e", false, "Show multiple examples")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("howtfdoi version %s\n", version)
		fmt.Printf("Download and documentation: %s\n", repository)
		os.Exit(0)
	}

	// Setup config
	config := setupConfig(*verboseFlag)

	// Check API key
	if config.APIKey == "" {
		if config.Provider == providerAnthropic {
			color.Red("Error: ANTHROPIC_API_KEY environment variable not set")
			fmt.Fprintln(os.Stderr, "Set it with: export ANTHROPIC_API_KEY='your-api-key'")
		} else {
			color.Red("Error: OPENAI_API_KEY environment variable not set")
			fmt.Fprintln(os.Stderr, "Set it with: export OPENAI_API_KEY='your-api-key'")
		}
		os.Exit(1)
	}

	// If no arguments, enter interactive mode
	args := flag.Args()
	if len(args) == 0 {
		runInteractiveMode(config)
		return
	}

	// Join all arguments into a single query
	query := strings.Join(args, " ")

	// Run the query
	response, err := runQuery(config, query, *examplesFlag)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}

	// Handle the response
	opts := ResponseOptions{
		CopyToClipboard: *copyFlag,
		Execute:         *executeFlag,
	}
	handleResponse(config, query, response, opts)
}

func setupConfig(verbose bool) Config {
	dataDir := getDataDirectory()

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// If we can't create the directory, fail explicitly
		color.Red("Error: Could not create data directory at %s: %v", dataDir, err)
		os.Exit(1)
	}

	if verbose {
		color.Cyan("Using data directory: %s", dataDir)
	}

	// Determine which provider to use
	// Check HOWTFDOI_AI_PROVIDER env var first, then fall back to checking which API key is set
	provider := strings.ToLower(os.Getenv("HOWTFDOI_AI_PROVIDER"))
	var apiKey string

	switch provider {
	case providerOpenAI, providerChatGPT:
		provider = providerOpenAI
		apiKey = os.Getenv("OPENAI_API_KEY")
	case providerAnthropic, "claude", "":
		// Default to Anthropic if not specified or if "claude" is specified
		provider = providerAnthropic
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		// If Anthropic key not set but OpenAI key is, switch to OpenAI
		if apiKey == "" && os.Getenv("OPENAI_API_KEY") != "" {
			provider = providerOpenAI
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
	default:
		color.Yellow("Warning: Unknown HOWTFDOI_AI_PROVIDER '%s', defaulting to Anthropic", provider)
		provider = providerAnthropic
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if verbose {
		color.Cyan("Using AI provider: %s", provider)
	}

	return Config{
		APIKey:      apiKey,
		HistoryFile: filepath.Join(dataDir, historyFileName),
		Platform:    runtime.GOOS,
		Verbose:     verbose,
		Provider:    provider,
	}
}

// getDataDirectory returns the appropriate data directory following XDG Base Directory spec
func getDataDirectory() string {
	// Check for XDG_STATE_HOME first (for logs and history)
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "howtfdoi")
	}

	// Fall back to ~/.local/state/howtfdoi
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home directory, fail explicitly
		color.Red("Error: Could not determine home directory: %v", err)
		os.Exit(1)
	}

	return filepath.Join(homeDir, ".local", "state", "howtfdoi")
}

func runQuery(config Config, query string, showExamples bool) (*Response, error) {
	// Create the appropriate provider
	var provider Provider
	switch config.Provider {
	case providerOpenAI:
		provider = NewOpenAIProvider(config.APIKey)
	case providerAnthropic:
		provider = NewAnthropicProvider(config.APIKey)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// Build system prompt with platform info
	systemPrompt := buildSystemPrompt(config.Platform, showExamples)

	// Build user query
	userQuery := query
	if !showExamples {
		userQuery = fmt.Sprintf("Platform: %s\nQuery: %s", config.Platform, query)
	}

	// Query the provider
	fullResponse, err := provider.Query(context.Background(), systemPrompt, userQuery)
	if err != nil {
		return nil, err
	}

	// Parse the response
	return parseResponse(fullResponse), nil
}

func buildSystemPrompt(platform string, showExamples bool) string {
	if showExamples {
		return `You are a command-line expert assistant. Provide multiple practical examples for the requested command or tool.

Rules:
- Output in plain text only, as this output may be copied directly to a terminal
- Show 3-5 different use cases
- Each example should have the command and a brief explanation
- Focus on common, practical scenarios
- Format: command followed by explanation in parentheses

Example format:
tar -czf archive.tar.gz directory/
(Creates a compressed tarball)

tar -xzf archive.tar.gz
(Extracts a compressed tarball)

tar -tzf archive.tar.gz
(Lists contents without extracting)`
	}

	return fmt.Sprintf(`You are a command-line expert assistant for %s systems. Provide concise, accurate answers about CLI tools and commands.

Rules:
- Output in plain text only, as this output may be copied directly to a terminal
- Give the command/answer directly and immediately
- Be extremely concise - no unnecessary explanation unless the command is complex
- Show the actual command first, then a brief one-line explanation if needed
- Provide platform-specific commands when relevant (%s vs Linux vs Windows)
- Focus on common Unix/Linux CLI tools

Example format:
tar -czf archive.tar.gz directory/
(Creates a compressed tarball of the directory)`, platform, platform)
}

// parseResponse extracts the command and explanation from Claude's response.
// The expected format is:
//   - First non-empty line: the actual command
//   - Remaining lines: explanation/context
func parseResponse(text string) *Response {
	lines := strings.Split(text, "\n")
	response := &Response{
		FullText: text,
	}

	// Filter out empty lines first to simplify parsing
	var nonEmptyLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}

	// First non-empty line is the command
	if len(nonEmptyLines) > 0 {
		response.Command = nonEmptyLines[0]
	}

	// Remaining lines are the explanation
	if len(nonEmptyLines) > 1 {
		response.Explanation = strings.Join(nonEmptyLines[1:], "\n")
	}

	return response
}

func displayResponse(response *Response) {
	// Color setup
	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgHiWhite)

	if response.Command != "" {
		// Display command in green
		green.Println(response.Command)

		// Display explanation in white if present
		if response.Explanation != "" {
			white.Println(response.Explanation)
		}
	} else {
		// If we couldn't parse it, just show the full response
		fmt.Println(response.FullText)
	}
}

// handleResponse processes a response with all requested options.
// This consolidates post-processing logic: display, safety checks, history logging,
// clipboard copying, execution, and alias suggestions.
func handleResponse(config Config, query string, response *Response, opts ResponseOptions) {
	// Display the response
	displayResponse(response)

	// Check for dangerous commands
	if isDangerous(response.Command) {
		color.Yellow("\n⚠️  WARNING: This command may be dangerous!")
		color.Yellow("Please review carefully before executing.")
	}

	// Save to history
	saveToHistory(config, query, response.FullText)

	// Copy to clipboard if requested
	if opts.CopyToClipboard && response.Command != "" {
		if err := clipboard.WriteAll(response.Command); err == nil {
			color.Cyan("\n📋 Command copied to clipboard!")
		}
	}

	// Execute if requested
	if opts.Execute && response.Command != "" {
		executeCommand(response.Command)
	}

	// Suggest alias for complex commands
	if shouldSuggestAlias(response.Command) {
		suggestAlias(query, response.Command)
	}
}

// isDangerous checks if a command matches any dangerous patterns.
// Uses pre-compiled regex patterns for efficiency.
func isDangerous(command string) bool {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(command) {
			return true
		}
	}
	return false
}

// saveToHistory appends a query and response to the history file.
// Logs warnings in verbose mode if saving fails.
func saveToHistory(config Config, query, response string) {
	f, err := os.OpenFile(config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if config.Verbose {
			color.Yellow("Warning: Could not open history file: %v", err)
		}
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n%s\n---\n", timestamp, query, response)
	if _, err := f.WriteString(entry); err != nil {
		if config.Verbose {
			color.Yellow("Warning: Could not write to history file: %v", err)
		}
		return
	}

	if config.Verbose {
		color.Cyan("Saved to history: %s", config.HistoryFile)
	}
}

func executeCommand(command string) {
	color.Cyan("\n⚡ Executing: %s\n", command)

	// Ask for confirmation for safety
	fmt.Print("Continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		color.Yellow("Cancelled.")
		return
	}

	// Execute the command
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		color.Red("Error executing command: %v", err)
	}
}

func shouldSuggestAlias(command string) bool {
	// Suggest alias for commands longer than threshold or with complex pipes
	return len(command) > aliasLengthThreshold || strings.Count(command, "|") > aliasPipeThreshold
}

func suggestAlias(query, command string) {
	color.Cyan("\n💡 This command is complex. Want to create a shell alias?")

	// Generate a simple alias name from the query
	aliasName := generateAliasName(query)

	color.HiBlack("Suggested alias:")
	fmt.Printf("  alias %s='%s'\n", aliasName, command)
	color.HiBlack("\nAdd this to your ~/.bashrc or ~/.zshrc")
}

func generateAliasName(query string) string {
	// Create a simple alias name from the query (max 3 words)
	words := strings.Fields(query)
	if len(words) > 3 {
		words = words[:3]
	}

	// Join and sanitize to alphanumeric only
	name := strings.Join(words, "")
	name = nonAlphanumericRegex.ReplaceAllString(name, "")
	return strings.ToLower(name)
}

// parseInteractiveLine extracts query and flags from an interactive line.
// Supports inline flags: -c (copy), -x (execute), -e (examples)
func parseInteractiveLine(line string) (query string, opts ResponseOptions, showExamples bool) {
	parts := strings.Fields(line)
	var queryParts []string

	for _, part := range parts {
		switch part {
		case "-c":
			opts.CopyToClipboard = true
		case "-x":
			opts.Execute = true
		case "-e":
			showExamples = true
		default:
			queryParts = append(queryParts, part)
		}
	}

	query = strings.Join(queryParts, " ")
	return
}

func runInteractiveMode(config Config) {
	// Setup readline for interactive mode
	rl, err := readline.New("howtfdoi> ")
	if err != nil {
		color.Red("Error starting interactive mode: %v", err)
		os.Exit(1)
	}
	defer rl.Close()

	color.Cyan("🚀 Interactive mode - Type your questions or 'exit' to quit")
	color.HiBlack("Tip: Use -c to copy, -x to execute, -e for examples\n")

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			color.Cyan("Goodbye! 👋")
			break
		}

		// Parse query and flags
		query, opts, showExamples := parseInteractiveLine(line)
		if query == "" {
			continue
		}

		// Run the query
		response, err := runQuery(config, query, showExamples)
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}

		// Handle the response
		fmt.Println()
		handleResponse(config, query, response, opts)
		fmt.Println()
	}
}
