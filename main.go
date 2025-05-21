package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <article link>")
		return
	}
	articleLink := os.Args[1]

	articleURL, err := url.Parse(articleLink)
	if err != nil {
		fmt.Println(err)
		return
	}

	basedArticle, err := opiniate(articleLink)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(basedArticle)

	audio, err := tts(basedArticle)
	if err != nil {
		fmt.Println(err)
		return
	}

	filepath.Base(articleURL.Path)

	os.WriteFile("audio.mp3", audio, 0644)
}

func opiniate(articleLink string) (string, error) {
	return "", nil
}

func tts(article string) ([]byte, error) {
	return nil, nil
}
