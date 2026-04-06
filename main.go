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
	"runtime/debug"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	openai "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

const (
	// Model configuration
	claudeModel = anthropic.ModelClaudeHaiku4_5
	gptModel    = "gpt-4o-mini"
	maxTokens   = 1024

	// History file name
	historyFileName = "history.log"

	// Config file name
	configFileName = "howtfdoi.yaml"

	// Provider types
	providerAnthropic = "anthropic"
	providerOpenAI    = "openai"
	providerChatGPT   = "chatgpt" // alias for openai
	providerLMStudio  = "lmstudio"

	// LM Studio defaults
	defaultLMStudioBaseURL = "http://localhost:1234/v1"
	defaultLMStudioModel   = "local-model"
)

var (
	// version is set at build time via -ldflags, or read from Go module info
	version = ""
	// repository URL for documentation and downloads
	repository = "https://github.com/NeckBeardPrince/howtfdoi"
)

func init() {
	if version != "" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	} else {
		version = "unknown"
	}
}

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
)

// FileConfig holds configuration loaded from the YAML config file
type FileConfig struct {
	Provider        string `yaml:"provider,omitempty"`
	AnthropicKey    string `yaml:"anthropic_api_key,omitempty"`
	OpenAIKey       string `yaml:"openai_api_key,omitempty"`
	LMStudioBaseURL string `yaml:"lmstudio_base_url,omitempty"`
	LMStudioModel   string `yaml:"lmstudio_model,omitempty"`
}

// Config holds runtime configuration
type Config struct {
	APIKey          string
	HistoryFile     string
	Platform        string
	Verbose         bool
	Provider        string // "anthropic", "openai", or "lmstudio"
	LMStudioBaseURL string
	LMStudioModel   string
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
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  gptModel,
	}
}

// Query sends a query to OpenAI's API
func (p *OpenAIProvider) Query(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     p.model,
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

// LMStudioProvider implements Provider for LM Studio's local OpenAI-compatible API.
// Embeds OpenAIProvider since LM Studio speaks the same protocol.
type LMStudioProvider struct {
	*OpenAIProvider
}

// NewLMStudioProvider creates a new LM Studio provider with a custom base URL.
func NewLMStudioProvider(baseURL, model string) *LMStudioProvider {
	config := openai.DefaultConfig("")
	config.BaseURL = baseURL
	return &LMStudioProvider{
		OpenAIProvider: &OpenAIProvider{
			client: openai.NewClientWithConfig(config),
			model:  model,
		},
	}
}

func main() {
	// Customize help output to include version information
	flag.Usage = func() {
		fmt.Printf("howtfdoi version %s\n", version)
		fmt.Printf("Download and documentation: %s\n\n", repository)

		fmt.Fprintf(os.Stderr, "Ask CLI questions in plain English and get instant answers powered by AI.\n\n")

		fmt.Fprintf(os.Stderr, "GETTING STARTED:\n")
		fmt.Fprintf(os.Stderr, "  1. Run howtfdoi — first-time setup will prompt you for your API key.\n")
		fmt.Fprintf(os.Stderr, "     Or set up manually via environment variable:\n")
		fmt.Fprintf(os.Stderr, "     • For Claude:     export ANTHROPIC_API_KEY='your-key-here'\n")
		fmt.Fprintf(os.Stderr, "     • For ChatGPT:    export OPENAI_API_KEY='your-key-here'\n")
		fmt.Fprintf(os.Stderr, "     • For LM Studio:  export HOWTFDOI_AI_PROVIDER='lmstudio'\n\n")
		fmt.Fprintf(os.Stderr, "  2. Ask a question:\n")
		fmt.Fprintf(os.Stderr, "     howtfdoi compress a directory\n\n")

		fmt.Fprintf(os.Stderr, "USAGE:\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi [flags] <query>\n")
		fmt.Fprintf(os.Stderr, "  howtfdoi              (interactive mode)\n\n")

		fmt.Fprintf(os.Stderr, "FLAGS:\n")
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, "\nCONFIG FILE:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", filepath.Join(getConfigDirectory(), configFileName))
		fmt.Fprintf(os.Stderr, "  API keys and provider preference can be stored in this YAML file.\n")
		fmt.Fprintf(os.Stderr, "  Environment variables take precedence over config file values.\n")

		fmt.Fprintf(os.Stderr, "\nENVIRONMENT VARIABLES:\n")
		fmt.Fprintf(os.Stderr, "  ANTHROPIC_API_KEY     Your Anthropic API key (get it at console.anthropic.com)\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY        Your OpenAI API key (get it at platform.openai.com)\n")
		fmt.Fprintf(os.Stderr, "  HOWTFDOI_AI_PROVIDER  Override provider choice: anthropic, openai, chatgpt, or lmstudio\n")
		fmt.Fprintf(os.Stderr, "                        (defaults to anthropic, or auto-detects from available keys)\n")
		fmt.Fprintf(os.Stderr, "  LMSTUDIO_BASE_URL     LM Studio server URL (default: %s)\n", defaultLMStudioBaseURL)
		fmt.Fprintf(os.Stderr, "  LMSTUDIO_MODEL        LM Studio model name (default: %s)\n", defaultLMStudioModel)
		fmt.Fprintf(os.Stderr, "  XDG_CONFIG_HOME       Override config directory (default: ~/.config)\n")
		fmt.Fprintf(os.Stderr, "  XDG_STATE_HOME        Override state directory (default: ~/.local/state)\n")

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

	// Check API key (LM Studio doesn't need one)
	if config.APIKey == "" && config.Provider != providerLMStudio {
		configPath := filepath.Join(getConfigDirectory(), configFileName)
		if config.Provider == providerAnthropic {
			color.Red("Error: No Anthropic API key found")
			fmt.Fprintf(os.Stderr, "Set it via environment variable: export ANTHROPIC_API_KEY='your-api-key'\n")
			fmt.Fprintf(os.Stderr, "Or add it to your config file: %s\n", configPath)
		} else {
			color.Red("Error: No OpenAI API key found")
			fmt.Fprintf(os.Stderr, "Set it via environment variable: export OPENAI_API_KEY='your-api-key'\n")
			fmt.Fprintf(os.Stderr, "Or add it to your config file: %s\n", configPath)
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

// resolveLMStudioConfig resolves LM Studio base URL and model from env vars, config file, then defaults.
func resolveLMStudioConfig(fileConfig FileConfig) (baseURL, model string) {
	baseURL = os.Getenv("LMSTUDIO_BASE_URL")
	if baseURL == "" {
		baseURL = fileConfig.LMStudioBaseURL
	}
	if baseURL == "" {
		baseURL = defaultLMStudioBaseURL
	}

	model = os.Getenv("LMSTUDIO_MODEL")
	if model == "" {
		model = fileConfig.LMStudioModel
	}
	if model == "" {
		model = defaultLMStudioModel
	}
	return
}

func setupConfig(verbose bool) Config {
	dataDir := getDataDirectory()
	configDir := getConfigDirectory()

	// Ensure both directories exist on first run
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		color.Red("Error: Could not create data directory at %s: %v", dataDir, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0700); err != nil {
		color.Red("Error: Could not create config directory at %s: %v", configDir, err)
		os.Exit(1)
	}

	if verbose {
		color.Cyan("Using data directory: %s", dataDir)
		color.Cyan("Using config file: %s", filepath.Join(configDir, configFileName))
	}

	// Load config file
	fileConfig := loadConfigFile()

	// Determine which provider to use
	// Priority: env var > config file > default (anthropic)
	provider := strings.ToLower(os.Getenv("HOWTFDOI_AI_PROVIDER"))
	var apiKey string
	var lmStudioBaseURL, lmStudioModel string

	switch provider {
	case providerOpenAI, providerChatGPT:
		provider = providerOpenAI
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = fileConfig.OpenAIKey
		}
	case providerAnthropic, "claude":
		provider = providerAnthropic
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = fileConfig.AnthropicKey
		}
	case providerLMStudio:
		lmStudioBaseURL, lmStudioModel = resolveLMStudioConfig(fileConfig)
	case "":
		// No env var set — check config file provider, then auto-detect
		if fileConfig.Provider != "" {
			provider = strings.ToLower(fileConfig.Provider)
			if provider == providerChatGPT {
				provider = providerOpenAI
			}
		}

		switch provider {
		case providerOpenAI:
			apiKey = os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				apiKey = fileConfig.OpenAIKey
			}
		case providerLMStudio:
			lmStudioBaseURL, lmStudioModel = resolveLMStudioConfig(fileConfig)
		default:
			provider = providerAnthropic
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				apiKey = fileConfig.AnthropicKey
			}
			// If still no Anthropic key, try OpenAI from env or config
			if apiKey == "" {
				if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
					provider = providerOpenAI
					apiKey = envKey
				} else if fileConfig.OpenAIKey != "" {
					provider = providerOpenAI
					apiKey = fileConfig.OpenAIKey
				}
			}
		}
	default:
		color.Yellow("Warning: Unknown HOWTFDOI_AI_PROVIDER '%s', defaulting to Anthropic", provider)
		provider = providerAnthropic
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = fileConfig.AnthropicKey
		}
	}

	// If still no API key (and not LM Studio) and stdin is a terminal, run first-time setup
	if apiKey == "" && provider != providerLMStudio && isatty.IsTerminal(os.Stdin.Fd()) {
		fc, err := runFirstTimeSetup()
		if err != nil {
			color.Red("Error during setup: %v", err)
			os.Exit(1)
		}
		fileConfig = fc
		provider = fc.Provider
		switch provider {
		case providerOpenAI:
			apiKey = fc.OpenAIKey
		case providerLMStudio:
			lmStudioBaseURL, lmStudioModel = resolveLMStudioConfig(fc)
		default:
			provider = providerAnthropic
			apiKey = fc.AnthropicKey
		}
	}

	if verbose {
		color.Cyan("Using AI provider: %s", provider)
		if provider == providerLMStudio {
			color.Cyan("LM Studio base URL: %s", lmStudioBaseURL)
			color.Cyan("LM Studio model: %s", lmStudioModel)
		}
	}

	return Config{
		APIKey:          apiKey,
		HistoryFile:     filepath.Join(dataDir, historyFileName),
		Platform:        runtime.GOOS,
		Verbose:         verbose,
		Provider:        provider,
		LMStudioBaseURL: lmStudioBaseURL,
		LMStudioModel:   lmStudioModel,
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

// getConfigDirectory returns the appropriate config directory following XDG Base Directory spec
func getConfigDirectory() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "howtfdoi")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		color.Red("Error: Could not determine home directory: %v", err)
		os.Exit(1)
	}

	return filepath.Join(homeDir, ".config", "howtfdoi")
}

// loadConfigFile reads and parses the YAML config file.
// Returns a zero-value FileConfig if the file doesn't exist.
func loadConfigFile() FileConfig {
	configPath := filepath.Join(getConfigDirectory(), configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return FileConfig{}
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}
	}
	return fc
}

// saveConfigFile writes the FileConfig to the YAML config file.
func saveConfigFile(fc FileConfig) error {
	configDir := getConfigDirectory()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := yaml.Marshal(&fc)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	header := "# WARNING: This file contains API keys. Do NOT commit this file to git.\n" +
		"# Add this file to your .gitignore if it is inside a repository.\n\n"

	configPath := filepath.Join(configDir, configFileName)
	if err := os.WriteFile(configPath, []byte(header+string(data)), 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	// Write a .gitignore to protect against accidental commits
	gitignorePath := filepath.Join(configDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := "# Ignore config file containing API keys\n" + configFileName + "\n"
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			// Non-fatal — warn but don't fail
			fmt.Fprintf(os.Stderr, "Warning: Could not create .gitignore in config directory: %v\n", err)
		}
	}

	return nil
}

// runFirstTimeSetup interactively prompts the user to configure their API key and provider.
func runFirstTimeSetup() (FileConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	color.Cyan("Welcome to howtfdoi! Let's set up your configuration.")
	fmt.Println()

	// Prompt for provider
	fmt.Println("Which AI provider would you like to use?")
	fmt.Println("  1. Anthropic (Claude) — default")
	fmt.Println("  2. OpenAI (ChatGPT)")
	fmt.Println("  3. LM Studio (Local)")
	fmt.Print("Enter 1, 2, or 3 [1]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var fc FileConfig
	switch choice {
	case "2":
		fc.Provider = providerOpenAI
	case "3":
		fc.Provider = providerLMStudio
	default:
		fc.Provider = providerAnthropic
	}

	// Prompt for provider-specific configuration
	switch fc.Provider {
	case providerOpenAI:
		fmt.Println("\nGet your API key at: https://platform.openai.com/api-keys")
		fmt.Print("Enter your OpenAI API key: ")
		key, _ := reader.ReadString('\n')
		fc.OpenAIKey = strings.TrimSpace(key)
		if fc.OpenAIKey == "" {
			return fc, fmt.Errorf("no API key provided")
		}
	case providerLMStudio:
		fmt.Println("\nLM Studio runs locally — no API key needed.")
		fmt.Println("Make sure LM Studio is running with a model loaded.")
		fmt.Println("Download LM Studio at: https://lmstudio.ai/")
		fmt.Printf("\nEnter LM Studio server URL [%s]: ", defaultLMStudioBaseURL)
		baseURL, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			baseURL = defaultLMStudioBaseURL
		}
		fc.LMStudioBaseURL = baseURL

		fmt.Printf("Enter model name [%s]: ", defaultLMStudioModel)
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = defaultLMStudioModel
		}
		fc.LMStudioModel = model
	default:
		fmt.Println("\nGet your API key at: https://console.anthropic.com/settings/keys")
		fmt.Print("Enter your Anthropic API key: ")
		key, _ := reader.ReadString('\n')
		fc.AnthropicKey = strings.TrimSpace(key)
		if fc.AnthropicKey == "" {
			return fc, fmt.Errorf("no API key provided")
		}
	}

	// Save config
	if err := saveConfigFile(fc); err != nil {
		return fc, fmt.Errorf("could not save config: %w", err)
	}

	configPath := filepath.Join(getConfigDirectory(), configFileName)
	fmt.Println()
	color.Green("Configuration saved to %s", configPath)
	color.Yellow("⚠️  This file contains your API key. Do NOT commit it to git.")
	fmt.Println()

	return fc, nil
}

func runQuery(config Config, query string, showExamples bool) (*Response, error) {
	// Create the appropriate provider
	var provider Provider
	switch config.Provider {
	case providerOpenAI:
		provider = NewOpenAIProvider(config.APIKey)
	case providerAnthropic:
		provider = NewAnthropicProvider(config.APIKey)
	case providerLMStudio:
		provider = NewLMStudioProvider(config.LMStudioBaseURL, config.LMStudioModel)
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
	noMarkdownRule := "- Output in PLAIN TEXT ONLY — no markdown, no backticks, no code fences. Never wrap commands in backtick or triple-backtick blocks."

	if showExamples {
		return "You are a command-line expert assistant. Provide multiple practical examples for the requested command or tool.\n\n" +
			"Rules:\n" +
			noMarkdownRule + "\n" +
			"- Show 3-5 different use cases\n" +
			"- Each example: command on its own line, then explanation in parentheses on the next line\n" +
			"- Focus on common, practical scenarios\n\n" +
			"Example format:\n" +
			"tar -czf archive.tar.gz directory/\n" +
			"(Creates a compressed tarball)\n\n" +
			"tar -xzf archive.tar.gz\n" +
			"(Extracts a compressed tarball)\n\n" +
			"tar -tzf archive.tar.gz\n" +
			"(Lists contents without extracting)"
	}

	return fmt.Sprintf(
		"You are a command-line expert assistant for %s systems. Provide concise, accurate answers about CLI tools and commands.\n\n"+
			"Rules:\n"+
			noMarkdownRule+"\n"+
			"- Give the command/answer directly and immediately\n"+
			"- Be extremely concise - no unnecessary explanation unless the command is complex\n"+
			"- Show the actual command first, then a brief one-line explanation if needed\n"+
			"- Provide platform-specific commands when relevant (%s vs Linux vs Windows)\n"+
			"- Focus on common Unix/Linux CLI tools\n\n"+
			"Example format:\n"+
			"tar -czf archive.tar.gz directory/\n"+
			"(Creates a compressed tarball of the directory)",
		platform, platform,
	)
}

// stripMarkdown removes markdown code fences and inline backticks from text.
// The AI occasionally returns backtick-fenced blocks despite being told not to.
func stripMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		// Drop lines that are only a code fence (``` or ```bash etc.)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			continue
		}
		// Strip inline backtick wrapping from a whole line (e.g. `command`)
		if strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`") && len(trimmed) > 2 {
			line = trimmed[1 : len(trimmed)-1]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// parseResponse extracts the command and explanation from Claude's response.
// The expected format is:
//   - First non-empty line: the actual command
//   - Remaining lines: explanation/context
func parseResponse(text string) *Response {
	text = stripMarkdown(text)
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

// --- Bubbletea TUI for interactive mode ---

// tuiState represents what the TUI is currently doing
type tuiState int

const (
	tuiStateInput    tuiState = iota // waiting for user input
	tuiStateLoading                  // querying the AI
	tuiStateResponse                 // displaying a response
)

// queryResultMsg carries the result of an async AI query back to the TUI
type queryResultMsg struct {
	response *Response
	query    string
	opts     ResponseOptions
	err      error
}

// tuiModel is the Bubbletea application model
type tuiModel struct {
	config    Config
	state     tuiState
	textarea  textarea.Model
	viewport  viewport.Model
	spinner   spinner.Model
	history   []string // rendered response history
	width     int
	height    int
	lastQuery string
	lastOpts  ResponseOptions
	err       error

	// styles
	stylePrompt   lipgloss.Style
	styleResponse lipgloss.Style
	styleCommand  lipgloss.Style
	styleHint     lipgloss.Style
	styleError    lipgloss.Style
	styleBorder   lipgloss.Style
}

func newTUIModel(config Config) tuiModel {
	ta := textarea.New()
	ta.Placeholder = "Ask a CLI question... (Enter to send, Ctrl+D or 'exit' to quit)"
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter submits, not newlines

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	vp := viewport.New(80, 20)
	vp.SetContent("")

	return tuiModel{
		config:   config,
		state:    tuiStateInput,
		textarea: ta,
		viewport: vp,
		spinner:  sp,

		stylePrompt:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		styleResponse: lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		styleCommand:  lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		styleHint:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		styleError:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		styleBorder:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("6")).Padding(0, 1),
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textarea.Blink
}

// asyncQuery runs the AI query in a goroutine and returns a tea.Cmd
func asyncQuery(config Config, query string, opts ResponseOptions, showExamples bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := runQuery(config, query, showExamples)
		return queryResultMsg{response: resp, query: query, opts: opts, err: err}
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.state != tuiStateInput {
				break
			}
			line := strings.TrimSpace(m.textarea.Value())
			if line == "" {
				break
			}
			if line == "exit" || line == "quit" {
				return m, tea.Quit
			}

			query, opts, showExamples := parseInteractiveLine(line)
			if query == "" {
				m.textarea.Reset()
				break
			}

			m.lastQuery = query
			m.lastOpts = opts
			m.state = tuiStateLoading
			m.textarea.Reset()
			cmds = append(cmds, asyncQuery(m.config, query, opts, showExamples), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.viewport.SetContent(strings.Join(m.history, "\n\n"))

	case queryResultMsg:
		m.state = tuiStateResponse
		if msg.err != nil {
			m.err = msg.err
			entry := m.styleError.Render("Error: " + msg.err.Error())
			m.history = append(m.history, m.stylePrompt.Render("howtfdoi> ")+m.styleHint.Render(msg.query), entry)
		} else {
			// Save to history file
			saveToHistory(m.config, msg.query, msg.response.FullText)

			// Copy to clipboard if requested
			if msg.opts.CopyToClipboard && msg.response.Command != "" {
				_ = clipboard.WriteAll(msg.response.Command)
			}

			// Build rendered entry
			var parts []string
			parts = append(parts, m.stylePrompt.Render("howtfdoi> ")+m.styleHint.Render(msg.query))
			if msg.response.Command != "" {
				parts = append(parts, m.styleCommand.Render(msg.response.Command))
				if msg.response.Explanation != "" {
					parts = append(parts, m.styleResponse.Render(msg.response.Explanation))
				}
				if isDangerous(msg.response.Command) {
					parts = append(parts, m.styleError.Render("WARNING: This command may be dangerous!"))
				}
				if msg.opts.CopyToClipboard {
					parts = append(parts, m.styleHint.Render("Copied to clipboard."))
				}
			} else {
				parts = append(parts, m.styleResponse.Render(msg.response.FullText))
			}
			m.history = append(m.history, strings.Join(parts, "\n"))

			// If execute was requested, we'll need to quit TUI and run it
			if msg.opts.Execute && msg.response.Command != "" {
				m.state = tuiStateInput
				m.viewport.SetContent(strings.Join(m.history, "\n\n"))
				m.viewport.GotoBottom()
				// Queue execution after render
				return m, tea.Sequence(tea.Println(""), tea.Quit)
			}
		}

		m.viewport.SetContent(strings.Join(m.history, "\n\n"))
		m.viewport.GotoBottom()
		m.state = tuiStateInput
		cmds = append(cmds, textarea.Blink)

	case spinner.TickMsg:
		if m.state == tuiStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update child components
	var taCmd, vpCmd tea.Cmd
	if m.state == tuiStateInput {
		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	hint := m.styleHint.Render("Flags: -c copy  -x execute  -e examples  |  Ctrl+D or 'exit' to quit")

	var statusLine string
	if m.state == tuiStateLoading {
		statusLine = m.spinner.View() + " " + m.styleHint.Render("Asking AI...")
	} else {
		statusLine = m.stylePrompt.Render("howtfdoi")
	}

	vpView := m.styleBorder.Width(m.width - 4).Render(m.viewport.View())
	taView := m.styleBorder.Width(m.width - 4).Render(m.textarea.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		vpView,
		"",
		statusLine,
		taView,
		hint,
	)
}

func runInteractiveMode(config Config) {
	m := newTUIModel(config)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		color.Red("Error running interactive mode: %v", err)
		os.Exit(1)
	}

	// Handle execute after TUI exits (if -x was used on last query)
	if fm, ok := finalModel.(tuiModel); ok {
		if fm.lastOpts.Execute && fm.state == tuiStateInput {
			// Re-run the last query to get response and execute
			resp, err := runQuery(config, fm.lastQuery, false)
			if err == nil && resp.Command != "" {
				executeCommand(resp.Command)
			}
		}
	}

	fmt.Println("Goodbye!")
}
