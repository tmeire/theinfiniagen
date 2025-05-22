package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// readTextFile reads the content of a text file and returns it as a string
func readTextFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	return string(content), nil
}

// extractBVO uses Google Gemini API to extract beliefs, values, and opinions from text
func extractBVO(ctx context.Context, client *genai.Client, text string) (string, error) {
	model := client.GenerativeModel("models/gemini-2.5-flash-preview-05-20")

	prompt := fmt.Sprintf(`
	Please analyze the following interview transcript and extract the beliefs, values, and opinions expressed by the speakers.
	Organize your response into three sections:
	1. Beliefs: What factual claims or worldviews are expressed?
	2. Values: What principles, ideals, or priorities are emphasized?
	3. Opinions: What subjective judgments or preferences are shared?

	For each item, include a brief quote or reference to the specific part of the transcript.

	Transcript:
	%s
	`, text)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	var result string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if str, ok := part.(genai.Text); ok {
				result += string(str)
			}
		}
	}

	return result, nil
}

func main() {
	// Check if API key is set
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is not set")
	}

	// Initialize Gemini API client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	models := client.ListModels(ctx)
	m, err := models.Next()
	for err == nil {
		fmt.Println(m)
		m, err = models.Next()
	}

	// Read the transcript file
	// Use relative path to the file in the same directory
	filePath := "./lex-fridman.txt"
	text, err := readTextFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read transcript: %v", err)
	}

	fmt.Println("Analyzing transcript to extract beliefs, values, and opinions...")

	// Extract beliefs, values, and opinions
	result, err := extractBVO(ctx, client, text)
	if err != nil {
		log.Fatalf("Failed to extract BVO: %v", err)
	}

	// Print the result
	fmt.Println("\n--- BELIEFS, VALUES, AND OPINIONS ANALYSIS ---")
	fmt.Println(result)
}
