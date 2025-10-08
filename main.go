package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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
)

// Dangerous command patterns
var dangerousPatterns = []string{
	`rm\s+-rf\s+/`,
	`rm\s+-rf\s+\*`,
	`dd\s+.*of=/dev/`,
	`mkfs\.`,
	`:(){ :|:& };:`,
	`>\s*/dev/sd`,
	`mv\s+.*\s+/dev/null`,
}

// Config holds runtime configuration
type Config struct {
	APIKey      string
	HistoryFile string
	Platform    string
}

// Response holds the parsed response
type Response struct {
	Command     string
	Explanation string
	FullText    string
}

func main() {
	// Parse flags
	copyFlag := flag.Bool("c", false, "Copy command to clipboard")
	executeFlag := flag.Bool("x", false, "Execute the command directly")
	examplesFlag := flag.Bool("e", false, "Show multiple examples")
	flag.Parse()

	// Setup config
	config := setupConfig()

	// Check API key
	if config.APIKey == "" {
		color.Red("Error: ANTHROPIC_API_KEY environment variable not set")
		fmt.Fprintln(os.Stderr, "Set it with: export ANTHROPIC_API_KEY='your-api-key'")
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

	// Display the response with colors
	displayResponse(response)

	// Check for dangerous commands
	if isDangerous(response.Command) {
		color.Yellow("\n⚠️  WARNING: This command may be dangerous!")
		color.Yellow("Please review carefully before executing.")
	}

	// Save to history
	saveToHistory(config, query, response.FullText)

	// Copy to clipboard if requested
	if *copyFlag && response.Command != "" {
		if err := clipboard.WriteAll(response.Command); err == nil {
			color.Cyan("\n📋 Command copied to clipboard!")
		}
	}

	// Execute if requested
	if *executeFlag && response.Command != "" {
		executeCommand(response.Command)
	}

	// Suggest alias for complex commands
	if shouldSuggestAlias(response.Command) {
		suggestAlias(query, response.Command)
	}
}

func setupConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		HistoryFile: filepath.Join(homeDir, ".howtfdoi_history"),
		Platform:    runtime.GOOS,
	}
}

func runQuery(config Config, query string, showExamples bool) (*Response, error) {
	// Create Claude client
	client := anthropic.NewClient(
		option.WithAPIKey(config.APIKey),
	)

	// Build system prompt with platform info and prompt caching
	systemPrompt := buildSystemPrompt(config.Platform, showExamples)

	// Build user query
	userQuery := query
	if !showExamples {
		userQuery = fmt.Sprintf("Platform: %s\nQuery: %s", config.Platform, query)
	}

	// Stream the response for speed
	stream := client.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5HaikuLatest,
		MaxTokens: 1024,
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

	// Collect the full response
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
		return nil, err
	}

	// Parse the response
	return parseResponse(fullResponse.String()), nil
}

func buildSystemPrompt(platform string, showExamples bool) string {
	if showExamples {
		return `You are a command-line expert assistant. Provide multiple practical examples for the requested command or tool.

Rules:
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
- Give the command/answer directly and immediately
- Be extremely concise - no unnecessary explanation unless the command is complex
- Show the actual command first, then a brief one-line explanation if needed
- Provide platform-specific commands when relevant (%s vs Linux vs Windows)
- Focus on common Unix/Linux CLI tools

Example format:
tar -czf archive.tar.gz directory/
(Creates a compressed tarball of the directory)`, platform, platform)
}

func parseResponse(text string) *Response {
	lines := strings.Split(text, "\n")
	response := &Response{
		FullText: text,
	}

	// Try to find the first line that looks like a command
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First non-empty line is likely the command
		if response.Command == "" {
			response.Command = line
			continue
		}

		// Rest is explanation
		if i < len(lines) {
			response.Explanation = strings.TrimSpace(strings.Join(lines[i:], "\n"))
			break
		}
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

func isDangerous(command string) bool {
	for _, pattern := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, command)
		if matched {
			return true
		}
	}
	return false
}

func saveToHistory(config Config, query, response string) {
	f, err := os.OpenFile(config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n%s\n---\n", timestamp, query, response)
	f.WriteString(entry)
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
	// Suggest alias for commands longer than 40 chars or with complex pipes
	return len(command) > 40 || strings.Count(command, "|") > 1
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
	// Create a simple alias name from the query
	words := strings.Fields(query)
	if len(words) > 3 {
		words = words[:3]
	}

	name := strings.Join(words, "")
	name = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(name, "")
	name = strings.ToLower(name)

	return name
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

		// Parse flags from the line
		parts := strings.Fields(line)
		copyFlag := false
		executeFlag := false
		examplesFlag := false
		var queryParts []string

		for _, part := range parts {
			switch part {
			case "-c":
				copyFlag = true
			case "-x":
				executeFlag = true
			case "-e":
				examplesFlag = true
			default:
				queryParts = append(queryParts, part)
			}
		}

		if len(queryParts) == 0 {
			continue
		}

		query := strings.Join(queryParts, " ")

		// Run the query
		response, err := runQuery(config, query, examplesFlag)
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}

		// Display the response
		fmt.Println()
		displayResponse(response)

		// Check for dangerous commands
		if isDangerous(response.Command) {
			color.Yellow("\n⚠️  WARNING: This command may be dangerous!")
		}

		// Save to history
		saveToHistory(config, query, response.FullText)

		// Copy to clipboard if requested
		if copyFlag && response.Command != "" {
			if err := clipboard.WriteAll(response.Command); err == nil {
				color.Cyan("📋 Copied to clipboard!")
			}
		}

		// Execute if requested
		if executeFlag && response.Command != "" {
			executeCommand(response.Command)
		}

		fmt.Println()
	}
}
