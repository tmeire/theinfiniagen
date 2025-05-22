package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
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

	resp, err := client.Models.GenerateContent(ctx, "models/gemini-2.5-flash-preview-05-20", genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	var result string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			result += part.Text
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

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
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
