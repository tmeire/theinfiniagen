package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <article link>")
		return
	}
	articleLink := os.Args[1]

	// Check if API key is set
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("GOOGLE_API_KEY environment variable is not set")
		return
	}

	articleURL, err := url.Parse(articleLink)
	if err != nil {
		fmt.Println(err)
		return
	}
	filename := filepath.Base(articleURL.Path)
	if ext := filepath.Ext(filename); ext != "" {
		filename = filename[:len(filename)-len(ext)]
	}

	// Initialize Gemini API client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Printf("error creating Gemini client: %v\n", err)
		return
	}
	defer client.Close()

	articleContent, err := eyes(ctx, client, articleLink)
	if err != nil {
		fmt.Println(err)
		return
	}
	cache("plain", filename, articleContent)

	basedArticle, err := brain(ctx, client, articleContent)
	if err != nil {
		fmt.Println(err)
		return
	}
	cache("based", filename, basedArticle)
	fmt.Println(basedArticle)

	//audio, err := mouth(basedArticle)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//
	//os.WriteFile(filename+".mp3", audio, 0644)
}

func cache(cacheType, filename, content string) {
	err := os.WriteFile(filename+"."+cacheType+".txt", []byte(content), 0644)
	if err != nil {
		panic(err)
	}
}

// eyes downloads the content from the provided URL and uses Gemini API to extract a text-only version
func eyes(ctx context.Context, client *genai.Client, articleURL string) (string, error) {
	fmt.Println("Navigating the web...")
	// Make HTTP request
	resp, err := http.Get(articleURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Trying to read the page, i'm very good at reading...")

	// Create a model instance
	model := client.GenerativeModel("models/gemini-2.5-flash-preview-05-20")

	// Create prompt for extracting text-only content
	prompt := fmt.Sprintf(`Extract the main text content from this HTML. 
Return ONLY the article text without any HTML tags, navigation elements, ads or other non-content elements.
Format the output as markdown text.

HTML content:
%s`, string(body))

	// Generate content
	genResp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content: %v", err)
	}

	// Extract the response text
	var result strings.Builder
	for _, candidate := range genResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				result.WriteString(string(text))
			}
		}
	}

	return result.String(), nil
}

// brain downloads an article from the provided URL, sends it to the Gemini API
// with beliefs, values, and opinions from bvo/bvo.txt, and returns the response.
func brain(ctx context.Context, client *genai.Client, articleContent string) (string, error) {
	// Read beliefs, values, and opinions
	bvoContent, err := os.ReadFile("bvo/bvo.txt")
	if err != nil {
		return "", fmt.Errorf("error reading BVO file: %v", err)
	}

	// Create a model instance
	model := client.GenerativeModel("models/gemini-2.5-flash-preview-05-20")

	// Create system prompt with BVO content
	systemPrompt := fmt.Sprintf(`You are an AI that adds beliefs, values, and opinions to articles.
Your task is to take the provided article and return it verbatim, but add additional beliefs, values, and opinions
that align with the following core beliefs, values, and opinions:

%s

When responding, first return a piece of the original article text verbatim, then add your beliefs, values, and opinions
that are relevant to that piece of the article's content. The additions should be as close to spoken language as possible.
Avoid repeating the content of the original article when adding the believes, values, and opinions. 
Make it clear where the original article ends and your additions begin.`, string(bvoContent))

	// Set the system instruction
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(systemPrompt),
		},
	}

	// Create the prompt with the article content
	prompt := fmt.Sprintf("Here is the article to process:\n\n%s", articleContent)

	fmt.Println("Prereading article...")

	// Generate content
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content: %v", err)
	}

	// Extract the response text
	var result strings.Builder
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				result.WriteString(string(text))
			}
		}
	}

	return result.String(), nil
}

func mouth(article string) ([]byte, error) {
	return nil, nil
}
