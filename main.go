package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"html2md/converter"
	"html2md/extractor"
	"html2md/metadata"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.html>\n", os.Args[0])
		os.Exit(1)
	}

	inputPath := os.Args[1]

	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing HTML: %v\n", err)
		os.Exit(1)
	}

	// Determine source file path relative to working directory
	sourceFile := inputPath

	// Extract metadata
	fm := metadata.Extract(doc, sourceFile)

	// Strip non-content elements
	extractor.RemoveScripts(doc)
	extractor.RemoveBoilerplate(doc)

	// Get the content root
	contentRoot := extractor.GetContentRoot(doc)

	// Build output path: same directory, .md extension
	outPath := computeOutputPath(inputPath)

	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Write YAML frontmatter
	writeFrontmatter(outFile, fm)

	// Write markdown content
	converter.ToMarkdown(outFile, contentRoot)

	fmt.Printf("Converted: %s -> %s\n", inputPath, outPath)
}

func computeOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	return filepath.Join(dir, base+".md")
}

func writeFrontmatter(w io.Writer, fm metadata.Frontmatter) {
	io.WriteString(w, "---\n")
	writeYAMLField(w, "title", fm.Title)
	writeYAMLField(w, "description", fm.Description)
	writeYAMLField(w, "og_title", fm.OgTitle)
	writeYAMLField(w, "og_description", fm.OgDesc)
	writeYAMLField(w, "og_image", fm.OgImage)
	writeYAMLField(w, "og_url", fm.OgURL)
	writeYAMLField(w, "canonical", fm.Canonical)
	writeYAMLField(w, "author", fm.Author)
	writeYAMLField(w, "date", fm.Date)
	writeYAMLField(w, "reading_time", fm.ReadingTime)
	writeYAMLField(w, "source_file", fm.SourceFile)
	io.WriteString(w, "---\n\n")
}

func writeYAMLField(w io.Writer, key, value string) {
	if value == "" {
		return
	}
	// Escape quotes in YAML strings
	escaped := strings.ReplaceAll(value, "\"", "\\\"")
	fmt.Fprintf(w, "%s: \"%s\"\n", key, escaped)
}
