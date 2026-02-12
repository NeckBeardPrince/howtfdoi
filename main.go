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
	"github.com/chzyer/readline"
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
	LMStudioBaseURL string // Base URL for LM Studio (e.g., "http://localhost:1234/v1")
	LMStudioModel   string // Model name for LM Studio
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

// LMStudioProvider implements Provider for LM Studio's local API (OpenAI-compatible)
type LMStudioProvider struct {
	client  *openai.Client
	model   string
	baseURL string
}

// NewLMStudioProvider creates a new LM Studio provider
func NewLMStudioProvider(baseURL, model string) *LMStudioProvider {
	// Create client config with custom base URL
	config := openai.DefaultConfig("")
	config.BaseURL = baseURL

	return &LMStudioProvider{
		client:  openai.NewClientWithConfig(config),
		model:   model,
		baseURL: baseURL,
	}
}

// Query sends a query to LM Studio's API
func (p *LMStudioProvider) Query(ctx context.Context, systemPrompt, userQuery string) (string, error) {
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
		fmt.Fprintf(os.Stderr, "  LMSTUDIO_BASE_URL     LM Studio base URL (default: http://localhost:1234/v1)\n")
		fmt.Fprintf(os.Stderr, "  LMSTUDIO_MODEL        LM Studio model name (default: local-model)\n")
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

	// Check API key (LM Studio doesn't need an API key, it runs locally)
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
	var lmStudioBaseURL string
	var lmStudioModel string

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
		provider = providerLMStudio
		// LM Studio doesn't need an API key
		apiKey = "not-needed"
		// Get base URL from env or config, default to localhost:1234
		lmStudioBaseURL = os.Getenv("LMSTUDIO_BASE_URL")
		if lmStudioBaseURL == "" {
			lmStudioBaseURL = fileConfig.LMStudioBaseURL
		}
		if lmStudioBaseURL == "" {
			lmStudioBaseURL = "http://localhost:1234/v1"
		}
		// Get model from env or config, default to "local-model"
		lmStudioModel = os.Getenv("LMSTUDIO_MODEL")
		if lmStudioModel == "" {
			lmStudioModel = fileConfig.LMStudioModel
		}
		if lmStudioModel == "" {
			lmStudioModel = "local-model"
		}
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
			apiKey = "not-needed"
			lmStudioBaseURL = os.Getenv("LMSTUDIO_BASE_URL")
			if lmStudioBaseURL == "" {
				lmStudioBaseURL = fileConfig.LMStudioBaseURL
			}
			if lmStudioBaseURL == "" {
				lmStudioBaseURL = "http://localhost:1234/v1"
			}
			lmStudioModel = os.Getenv("LMSTUDIO_MODEL")
			if lmStudioModel == "" {
				lmStudioModel = fileConfig.LMStudioModel
			}
			if lmStudioModel == "" {
				lmStudioModel = "local-model"
			}
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

	// If still no API key and stdin is a terminal, run first-time setup
	if apiKey == "" && isatty.IsTerminal(os.Stdin.Fd()) {
		fc, err := runFirstTimeSetup()
		if err != nil {
			color.Red("Error during setup: %v", err)
			os.Exit(1)
		}
		fileConfig = fc
		provider = fc.Provider
		if provider == providerOpenAI {
			apiKey = fc.OpenAIKey
		} else if provider == providerLMStudio {
			apiKey = "not-needed"
			lmStudioBaseURL = fc.LMStudioBaseURL
			if lmStudioBaseURL == "" {
				lmStudioBaseURL = "http://localhost:1234/v1"
			}
			lmStudioModel = fc.LMStudioModel
			if lmStudioModel == "" {
				lmStudioModel = "local-model"
			}
		} else {
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

	// Prompt for API key or LM Studio config
	if fc.Provider == providerOpenAI {
		fmt.Println("\nGet your API key at: https://platform.openai.com/api-keys")
		fmt.Print("Enter your OpenAI API key: ")
		key, _ := reader.ReadString('\n')
		fc.OpenAIKey = strings.TrimSpace(key)
		if fc.OpenAIKey == "" {
			return fc, fmt.Errorf("no API key provided")
		}
	} else if fc.Provider == providerLMStudio {
		fmt.Println("\nLM Studio runs locally and doesn't require an API key.")
		fmt.Println("Make sure LM Studio is running and has a model loaded.")
		fmt.Print("Enter LM Studio base URL [http://localhost:1234/v1]: ")
		baseURL, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			fc.LMStudioBaseURL = "http://localhost:1234/v1"
		} else {
			fc.LMStudioBaseURL = baseURL
		}
		fmt.Print("Enter model name [local-model]: ")
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			fc.LMStudioModel = "local-model"
		} else {
			fc.LMStudioModel = model
		}
	} else {
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
