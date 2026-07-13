# AGENTS.md — AI Coding Agent Reference

## Build & Run

```bash
go build -o html2md .    # Build binary
go vet ./...             # Lint
./html2md input.html     # Convert file (writes input.md)
./html2md -base https://example.com input.html  # With base URL
```

No test suite exists yet. Verify changes by converting a sample HTML file and inspecting the output.

## Architecture

```
HTML file → metadata.Extract() → extractor.Remove*() → converter.ToMarkdown() → .md file
```

Three packages, one CLI entrypoint:

| Package | File | Purpose |
|---------|------|---------|
| `metadata` | `frontmatter.go` | Extract page metadata into YAML frontmatter struct |
| `extractor` | `content.go` | Strip non-content DOM elements via CSS selectors |
| `converter` | `markdown.go` | Walk cleaned DOM, emit markdown to `io.Writer` |
| `main` | `main.go` | CLI glue: open file, run pipeline, write output |

## Key Files

- `main.go:20-85` — CLI entrypoint with `-base` flag, `resolveBaseURL()`, pipeline orchestration.
- `main.go:87-92` — `computeOutputPath()`: same dir, `.html` → `.md`.
- `metadata/frontmatter.go:31-47` — `Extract()`: orchestrates metadata extraction.
- `metadata/frontmatter.go:154-197` — `extractFAQ()`: parses JSON-LD `FAQPage` or `<details>` fallback.
- `extractor/content.go:8-12` — `RemoveScripts()`: strips `<script>`, `<style>`, `<meta>`.
- `extractor/content.go:16-69` — `RemoveBoilerplate()`: strips nav, footer, sidebar, promos, etc.
- `extractor/content.go:72-89` — `GetContentRoot()`: finds `main` > `article` > `.content` > `body`.
- `converter/markdown.go:15-19` — `ToMarkdown(w, sel, baseURL)`: entry point for conversion.
- `converter/markdown.go:22-92` — `processNode()`: recursive DOM walker with tag switch.
- `converter/markdown.go:220-255` — `writeList()`: handles ordered/unordered lists with `start` attr.
- `converter/markdown.go:300-320` — `resolveURL()`: resolves relative URLs against baseURL.
- `converter/markdown.go:330-380` — Inline text extraction via `golang.org/x/net/html` nodes.

## Conventions

- **No comments** in code unless explaining a non-obvious decision.
- **goquery** for CSS-selector-based DOM queries.
- **`golang.org/x/net/html`** for low-level node traversal in `extractInlineNode()`.
- Output file always written to **same directory** as input.
- Relative URLs resolved against `-base` flag, `<base>` tag, or `<link rel=canonical>` (see `converter/markdown.go:300`).

## Adding a New HTML Element

1. Add a `case` in `processNode()` (`converter/markdown.go:26`).
2. Write a `write*()` function that emits markdown.
3. For inline elements, handle in `extractInlineNode()` (`converter/markdown.go:345`).

## Adding a New Boilerplate Selector

Add a CSS selector string to the appropriate `doc.Find(...)` call in `extractor/content.go:16-68`. Group by category (nav, footer, sidebar, promo, etc.).

## Adding a New Frontmatter Field

1. Add a field to `Frontmatter` struct (`metadata/frontmatter.go:10`).
2. Add extraction logic in `Extract()` (`metadata/frontmatter.go:31`).
3. Add `writeYAMLField()` call in `writeFrontmatter()` (`main.go:107`).

## Known Issues

- **No streaming**: full DOM parse, memory-heavy for files >1MB.
- **No batch mode**: single file only.
