package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sukalov/karaokebot/internal/lyrics/parsers/amdm"
)

func main() {
	var outputFile string

	flag.StringVar(&outputFile, "output", "extracted_lyrics.txt", "Output file name")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <URL>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Example: %s https://123.amdm.ru/akkordi/mihail_krug/102195/vladimirskiy_tsentral/\n", os.Args[0])
		os.Exit(1)
	}

	url := args[0]

	fmt.Println("=== AmDm.ru Lyrics Extractor CLI ===")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Println()

	parser := amdm.NewParser()
	result, err := parser.ExtractLyricsFromAmdm(url)
	if err != nil {
		log.Fatalf("Error extracting lyrics: %v", err)
	}

	if !result.Success {
		log.Fatalf("Failed to extract lyrics: %s", result.Error)
	}

	if err := os.WriteFile(outputFile, []byte(result.Text), 0644); err != nil {
		log.Fatalf("Error saving file: %v", err)
	}

	fmt.Printf("Lyrics saved to: %s\n", outputFile)
	fmt.Println("=== EXTRACTION COMPLETED SUCCESSFULLY ===")
}
