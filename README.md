# html2md

Convert any HTML file to clean, AI agent-friendly markdown with YAML frontmatter. Strips all non-essential content — scripts, styles, navigation, sidebars, footers, promos — and outputs only the semantic content.

## Quick Start

```bash
# Build
go build -o html2md .

# Convert a file (writes .md alongside the .html)
./html2md page.html
# -> page.md

# With a base URL to resolve relative links
./html2md -base https://example.com page.html
# -> page.md (relative "/about" becomes "https://example.com/about")
```

**Requires:** Go 1.21+

## What It Does

| Input | Output |
|-------|--------|
| Bloated HTML (300KB–1MB) | Clean markdown (10–30KB) |
| Navigation, sidebars, footers, promos | Article content only |
| No metadata | YAML frontmatter with title, description, author, date, FAQ |

### Stripped

`<script>`, `<style>`, `<noscript>`, `<meta>`, `<svg>`, navigation bars, sidebars, footers, deal timers, promo banners, cookie popups, author bios, related posts, newsletter forms, partner logos.

### Preserved

Headings (h1–h6), paragraphs, bold, italic, inline code, links (relative → absolute), images with alt text, ordered lists (respects `start` attribute), unordered lists, tables, blockquotes, code blocks with language detection, `<details>`/`<summary>` (FAQs).

## Options

```
-base string   Base URL for resolving relative links (e.g. https://example.com)
```

If `-base` is not provided, the tool tries to auto-detect from:
1. `<base href="...">` tag
2. `<link rel="canonical" href="...">` tag (origin only)

If neither is found, relative URLs are left as-is.

## Output Format

```markdown
---
title: "Understanding Web Security | Example Blog"
description: "A deep dive into modern web security practices..."
og_image: "https://example.com/images/security.jpg"
canonical: "https://example.com/blog/web-security"
author: "Jane Smith"
date: "2024-03-15"
reading_time: "8 mins"
source_file: "blog/web-security.html"
---

# Understanding Web Security

A deep dive into modern web security practices...

## Why It Matters

Every [website](https://example.com) needs proper security...
```

## Project Structure

```
html2md/
├── main.go              # CLI: reads HTML, writes .md with frontmatter
├── metadata/
│   └── frontmatter.go   # Extracts title, OG tags, author, date, FAQ from JSON-LD
├── extractor/
│   └── content.go       # Strips JS, CSS, nav, footer, promos, SVGs
└── converter/
    └── markdown.go      # DOM → Markdown (headings, links, lists, tables, code)
```

## How It Works

Three-step pipeline:

1. **Metadata** (`metadata/`) — Parses `<title>`, `<meta>`, Open Graph tags, JSON-LD structured data, and common CSS selectors to extract page metadata into YAML frontmatter.

2. **Extraction** (`extractor/`) — Removes non-content elements using goquery CSS selectors. Finds the best content root (`<main>`, `<article>`, `.content`, etc.) and falls back to `<body>`.

3. **Conversion** (`converter/`) — Walks the cleaned DOM tree recursively, mapping HTML elements to markdown syntax. Handles edge cases like split `<ol start="N">` lists and headings inside list items.

## Limitations

- **Single file mode** — processes one HTML file per invocation.
- **Full DOM parse** — loads the entire HTML into memory. Works for files up to ~1MB; larger files may need more memory.
