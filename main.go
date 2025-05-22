package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	//"google.golang.org/api/option"
	"google.golang.org/genai"
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
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Printf("error creating Gemini client: %v\n", err)
		return
	}

	articleContent, ok := cached("plain", filename)
	if !ok {
		articleContent, err = eyes(ctx, client, articleLink)
		if err != nil {
			fmt.Println(err)
			return
		}
		cache("plain", filename, articleContent)
	}

	basedArticle, ok := cached("based", filename)
	if !ok {
		basedArticle, err := brain(ctx, client, articleContent)
		if err != nil {
			fmt.Println(err)
			return
		}
		cache("based", filename, basedArticle)
	}

	if _, err := os.Stat(filename + ".wav"); os.IsNotExist(err) {
		audio, err := mouth(ctx, client, basedArticle)
		if err != nil {
			fmt.Println(err)
			return
		}
		os.WriteFile(filename+".wav", toWav(audio), 0644)
	}
}

func cached(cacheType, filename string) (string, bool) {
	content, err := os.ReadFile(filename + "." + cacheType + ".txt")
	if err != nil {
		return "", false
	}
	return string(content), true
}

func cache(cacheType, filename, content string) {
	err := os.WriteFile(filename+"."+cacheType+".txt", []byte(content), 0644)
	if err != nil {
		panic(err)
	}
}

var imageRegex = regexp.MustCompile(`\!\[[^\]]*\]\([^\)]*\)(\n_([^_]*)_)?`)
var linkRegex = regexp.MustCompile(`\[([^\]]*)\]\([^\)]*\)`)

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

	// Create prompt for extracting text-only content
	prompt := fmt.Sprintf(`Extract the main text content from this HTML. 
Return ONLY the article text without any HTML tags, navigation elements, ads or other non-content elements.
Format the output as markdown text.

HTML content:
%s`, string(body))

	// Generate content
	genResp, err := client.Models.GenerateContent(ctx, "models/gemini-2.5-flash-preview-05-20", genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("error generating content: %v", err)
	}

	// Extract the response text
	var result strings.Builder
	for _, candidate := range genResp.Candidates {
		for _, part := range candidate.Content.Parts {
			result.WriteString(part.Text)
		}
	}

	// remove all images
	text := imageRegex.ReplaceAllString(result.String(), "")

	// replace all links with just the link text
	text = linkRegex.ReplaceAllString(text, "$1")

	return text, nil
}

// brain downloads an article from the provided URL, sends it to the Gemini API
// with beliefs, values, and opinions from bvo/bvo.txt, and returns the response.
func brain(ctx context.Context, client *genai.Client, articleContent string) (string, error) {
	// Read beliefs, values, and opinions
	bvoContent, err := os.ReadFile("bvo/bvo.txt")
	if err != nil {
		return "", fmt.Errorf("error reading BVO file: %v", err)
	}

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
	systemInstruction := &genai.Content{
		Parts: []*genai.Part{
			{Text: systemPrompt},
		},
	}

	// Create the prompt with the article content
	prompt := fmt.Sprintf("Here is the article to process:\n\n%s", articleContent)

	fmt.Println("Prereading article...")

	// Generate content
	resp, err := client.Models.GenerateContent(ctx, "models/gemini-2.5-flash-preview-05-20", genai.Text(prompt), &genai.GenerateContentConfig{SystemInstruction: systemInstruction})
	if err != nil {
		return "", fmt.Errorf("error generating content: %v", err)
	}

	// Extract the response text
	var result strings.Builder
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			result.WriteString(part.Text)
		}
	}

	return result.String(), nil
}

func mouth(ctx context.Context, client *genai.Client, article string) ([]byte, error) {
	config := &genai.GenerateContentConfig{}
	config.ResponseModalities = []string{"AUDIO"}
	config.SpeechConfig = &genai.SpeechConfig{
		LanguageCode: "en",
		VoiceConfig: &genai.VoiceConfig{
			PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
				VoiceName: "Iapetus", // Fenrir, Orus, Enceladus, Iapetus
			},
		},
	}

	// Create prompt for text-to-speech conversion with male voice and MP3 format
	prompt := fmt.Sprintf(`Say the following text in an upbeat tone. 
	%s`, article)

	// Generate content
	resp, err := client.Models.GenerateContent(ctx, "models/gemini-2.5-flash-preview-tts", genai.Text(prompt), config)
	if err != nil {
		return nil, fmt.Errorf("error generating audio content: %v", err)
	}

	// Extract the audio data
	var result []byte
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			blob := part.InlineData
			if blob != nil {
				result = append(result, blob.Data...)
			}
		}
	}
	if len(result) > 0 {
		return result, nil
	}

	return nil, fmt.Errorf("no audio data found in the response")
}

const sampleRate = 24000
const numChannels = 1
const bitDepth = 16

func toWav(audioData []byte) []byte {
	var b bytes.Buffer

	blockAlign := numChannels * bitDepth / 8
	byteRate := sampleRate * blockAlign

	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(len(audioData)))
	b.WriteString("WAVE")
	b.WriteString("fmt ")
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(1))
	binary.Write(&b, binary.LittleEndian, uint16(numChannels))
	binary.Write(&b, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&b, binary.LittleEndian, uint32(byteRate))
	binary.Write(&b, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&b, binary.LittleEndian, uint16(bitDepth))
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(len(audioData)))
	b.Write(audioData)

	return b.Bytes()
}
