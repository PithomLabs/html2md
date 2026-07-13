package main

import (
	"flag"
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
	baseURL := flag.String("base", "", "Base URL for resolving relative links (e.g. https://example.com)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input.html>\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := flag.Arg(0)

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

	// Determine base URL: flag > <base> tag > <link rel=canonical>
	resolvedBase := resolveBaseURL(doc, *baseURL)

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
	converter.ToMarkdown(outFile, contentRoot, resolvedBase)

	fmt.Printf("Converted: %s -> %s\n", inputPath, outPath)
}

// resolveBaseURL determines the base URL for resolving relative links.
// Priority: explicit flag > <base href> > <link rel="canonical"> > empty (leave relative).
func resolveBaseURL(doc *goquery.Document, flagBase string) string {
	if flagBase != "" {
		return strings.TrimRight(flagBase, "/")
	}
	if base, exists := doc.Find("base[href]").Attr("href"); exists && base != "" {
		return strings.TrimRight(base, "/")
	}
	if canonical, exists := doc.Find("link[rel=canonical]").Attr("href"); exists && canonical != "" {
		// Strip path to get origin only
		if idx := strings.Index(canonical, "://"); idx != -1 {
			end := idx + 3
			for i := end; i < len(canonical); i++ {
				if canonical[i] == '/' {
					return canonical[:i]
				}
			}
			return canonical
		}
	}
	return ""
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
	escaped := strings.ReplaceAll(value, "\"", "\\\"")
	fmt.Fprintf(w, "%s: \"%s\"\n", key, escaped)
}
