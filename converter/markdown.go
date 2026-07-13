package converter

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ToMarkdown converts a cleaned goquery selection to markdown, writing to w.
func ToMarkdown(w io.Writer, sel *goquery.Selection) {
	// Process children of the selection (not the container itself)
	for i := 0; i < sel.Length(); i++ {
		node := sel.Eq(i)
		processNode(w, node.Children(), 0)
	}
}

func processNode(w io.Writer, sel *goquery.Selection, depth int) {
	for i := 0; i < sel.Length(); i++ {
		node := sel.Eq(i)
		tag := goquery.NodeName(node)

		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			// Skip headings inside list items - they're handled by writeList
			if goquery.NodeName(node.Parent()) == "li" {
				continue
			}
			writeHeading(w, node, tag)
		case "p":
			writeParagraph(w, node)
		case "a":
			writeLink(w, node)
		case "img":
			writeImage(w, node)
		case "ol":
			// Collect consecutive single-item <ol> lists and merge them
			merged := collectConsecutiveOLs(sel, &i)
			if len(merged) > 1 {
				writeMergedList(w, merged, true, 0)
			} else {
				writeList(w, node, true, 0)
			}
		case "ul":
			writeList(w, node, false, 0)
		case "table":
			writeTable(w, node)
		case "blockquote":
			writeBlockquote(w, node)
		case "pre":
			writeCodeBlock(w, node)
		case "code":
			// Inline code inside a paragraph - handled by writeInline
			if goquery.NodeName(node.Parent()) != "pre" {
				fmt.Fprintf(w, "`%s`", node.Text())
			}
		case "details":
			writeDetails(w, node)
		case "hr":
			io.WriteString(w, "\n---\n\n")
		case "br":
			io.WriteString(w, "\n")
		case "strong", "b":
			text := extractText(node)
			if strings.TrimSpace(text) != "" {
				fmt.Fprintf(w, "**%s**", text)
			}
		case "em", "i":
			text := extractText(node)
			if strings.TrimSpace(text) != "" {
				fmt.Fprintf(w, "*%s*", text)
			}
		case "div", "section", "article", "main", "span", "header", "figure", "figcaption":
			// Wrapper elements - recurse into children
			processNode(w, node.Children(), depth)
		default:
			// For any other element, try to extract its text content
			text := strings.TrimSpace(node.Text())
			if text != "" && tag != "script" && tag != "style" {
				io.WriteString(w, text)
				io.WriteString(w, "\n")
			}
		}
	}
}

// collectConsecutiveOLs collects consecutive single-item <ol> elements starting at index i.
// It advances i past the collected elements.
func collectConsecutiveOLs(sel *goquery.Selection, i *int) []*goquery.Selection {
	var ols []*goquery.Selection
	for j := *i; j < sel.Length(); j++ {
		node := sel.Eq(j)
		if goquery.NodeName(node) != "ol" {
			break
		}
		// Check if it's a single-item list (just one <li>)
		liCount := node.Find("> li").Length()
		if liCount == 0 {
			break
		}
		if len(ols) > 0 && liCount > 1 {
			// Multi-item list after single-item lists - stop collecting
			break
		}
		ols = append(ols, node)
		*i = j
		if liCount > 1 {
			break // This is a multi-item list, include it and stop
		}
	}
	return ols
}

// writeMergedList writes multiple <ol> elements as a single numbered list.
func writeMergedList(w io.Writer, ols []*goquery.Selection, ordered bool, indent int) {
	io.WriteString(w, "\n")
	counter := 0
	for _, ol := range ols {
		ol.Find("> li").Each(func(_ int, li *goquery.Selection) {
			counter++
			prefix := "- "
			if ordered {
				prefix = fmt.Sprintf("%d. ", counter)
			}
			indentStr := strings.Repeat("  ", indent)
			io.WriteString(w, indentStr+prefix)
			text := extractInlineText(li)
			io.WriteString(w, text)
			io.WriteString(w, "\n")
		})
	}
	io.WriteString(w, "\n")
}

func writeHeading(w io.Writer, sel *goquery.Selection, tag string) {
	level := int(tag[1] - '0')
	text := extractInlineText(sel)
	if strings.TrimSpace(text) == "" {
		return
	}
	io.WriteString(w, "\n")
	io.WriteString(w, strings.Repeat("#", level))
	io.WriteString(w, " ")
	io.WriteString(w, text)
	io.WriteString(w, "\n\n")
}

func writeParagraph(w io.Writer, sel *goquery.Selection) {
	text := extractInlineText(sel)
	if strings.TrimSpace(text) == "" {
		return
	}
	io.WriteString(w, text)
	io.WriteString(w, "\n\n")
}

func writeLink(w io.Writer, sel *goquery.Selection) {
	text := extractInlineText(sel)
	href, _ := sel.Attr("href")
	if href == "" {
		io.WriteString(w, text)
		return
	}
	// Make relative URLs absolute if they start with /
	if strings.HasPrefix(href, "/") {
		href = "https://www.expressvpn.com" + href
	}
	fmt.Fprintf(w, "[%s](%s)", text, href)
}

func writeImage(w io.Writer, sel *goquery.Selection) {
	src, _ := sel.Attr("src")
	alt, _ := sel.Attr("alt")
	if src == "" {
		return
	}
	// Skip tiny tracking pixels and icons
	if strings.Contains(src, "1x1") || strings.Contains(src, "pixel") || strings.Contains(src, "tracking") {
		return
	}
	fmt.Fprintf(w, "![%s](%s)\n\n", alt, src)
}

func writeList(w io.Writer, sel *goquery.Selection, ordered bool, indent int) {
	io.WriteString(w, "\n")
	counter := 0
	// Check for start attribute on ordered lists
	if ordered {
		if startStr, exists := sel.Attr("start"); exists {
			var start int
			fmt.Sscanf(startStr, "%d", &start)
			counter = start - 1
		}
	}
	sel.Children().Each(func(_ int, li *goquery.Selection) {
		if goquery.NodeName(li) != "li" {
			return
		}
		counter++
		prefix := "- "
		if ordered {
			prefix = fmt.Sprintf("%d. ", counter)
		}
		indentStr := strings.Repeat("  ", indent)
		io.WriteString(w, indentStr+prefix)

		// Check if the li contains sub-lists
		hasSubList := li.Find("ul, ol").Length() > 0

		if hasSubList {
			// Process direct children, skip nested lists
			li.Children().Each(func(_ int, child *goquery.Selection) {
				childTag := goquery.NodeName(child)
				if childTag == "ul" || childTag == "ol" {
					writeList(w, child, childTag == "ol", indent+1)
				} else {
					text := extractInlineText(child)
					if strings.TrimSpace(text) != "" {
						io.WriteString(w, text)
					}
				}
			})
		} else {
			text := extractInlineText(li)
			io.WriteString(w, text)
		}
		io.WriteString(w, "\n")
	})
	io.WriteString(w, "\n")
}

func writeTable(w io.Writer, sel *goquery.Selection) {
	rows := sel.Find("tr")
	if rows.Length() == 0 {
		return
	}

	io.WriteString(w, "\n")

	var headers []string
	var dataRows [][]string

	// Check for thead
	thead := sel.Find("thead tr")
	tbody := sel.Find("tbody tr")

	if thead.Length() > 0 {
		thead.Find("th").Each(func(_ int, th *goquery.Selection) {
			headers = append(headers, extractInlineText(th))
		})
		tbody.Each(func(_ int, tr *goquery.Selection) {
			var row []string
			tr.Find("td").Each(func(_ int, td *goquery.Selection) {
				row = append(row, extractInlineText(td))
			})
			dataRows = append(dataRows, row)
		})
	} else {
		// First row is header
		firstRow := rows.First()
		firstRow.Find("th, td").Each(func(_ int, cell *goquery.Selection) {
			headers = append(headers, extractInlineText(cell))
		})
		rows.Slice(1, rows.Length()).Each(func(_ int, tr *goquery.Selection) {
			var row []string
			tr.Find("td").Each(func(_ int, td *goquery.Selection) {
				row = append(row, extractInlineText(td))
			})
			dataRows = append(dataRows, row)
		})
	}

	if len(headers) == 0 {
		return
	}

	// Write header
	io.WriteString(w, "| ")
	io.WriteString(w, strings.Join(headers, " | "))
	io.WriteString(w, " |\n")

	// Write separator
	io.WriteString(w, "| ")
	for range headers {
		io.WriteString(w, "--- | ")
	}
	io.WriteString(w, "\n")

	// Write data rows
	for _, row := range dataRows {
		io.WriteString(w, "| ")
		for j, cell := range row {
			if j < len(headers) {
				io.WriteString(w, cell)
				io.WriteString(w, " | ")
			}
		}
		io.WriteString(w, "\n")
	}
	io.WriteString(w, "\n")
}

func writeBlockquote(w io.Writer, sel *goquery.Selection) {
	text := extractInlineText(sel)
	lines := strings.Split(text, "\n")
	io.WriteString(w, "\n")
	for _, line := range lines {
		io.WriteString(w, "> ")
		io.WriteString(w, strings.TrimSpace(line))
		io.WriteString(w, "\n")
	}
	io.WriteString(w, "\n")
}

func writeCodeBlock(w io.Writer, sel *goquery.Selection) {
	// Try to detect language from class
	lang := ""
	for _, class := range strings.Split(sel.AttrOr("class", ""), " ") {
		if strings.HasPrefix(class, "language-") || strings.HasPrefix(class, "lang-") {
			lang = strings.TrimPrefix(class, "language-")
			lang = strings.TrimPrefix(lang, "lang-")
			break
		}
	}
	code := sel.Text()
	io.WriteString(w, "\n```")
	io.WriteString(w, lang)
	io.WriteString(w, "\n")
	io.WriteString(w, code)
	io.WriteString(w, "\n```\n\n")
}

func writeDetails(w io.Writer, sel *goquery.Selection) {
	summary := strings.TrimSpace(sel.Find("summary").Text())
	content := extractInlineText(sel.Find(".faq-content, .details-content, div:not(summary)"))

	if summary == "" {
		summary = "Details"
	}

	io.WriteString(w, "\n### ")
	io.WriteString(w, summary)
	io.WriteString(w, "\n\n")
	if content != "" {
		io.WriteString(w, content)
		io.WriteString(w, "\n\n")
	}
}

// extractInlineText extracts text from a selection, preserving inline formatting.
func extractInlineText(sel *goquery.Selection) string {
	var sb strings.Builder
	extractInlineNodes(&sb, sel.Nodes)
	return strings.TrimSpace(sb.String())
}

// extractInlineNodes processes a list of HTML nodes recursively.
func extractInlineNodes(sb *strings.Builder, nodes []*html.Node) {
	for _, node := range nodes {
		extractInlineNode(sb, node)
	}
}

// getInnerText recursively gets all text content from a node and its children.
func getInnerText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getInnerText(c))
	}
	return sb.String()
}

// extractInlineNode processes a single HTML node and its children.
func extractInlineNode(sb *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		sb.WriteString(node.Data)
		return
	}
	if node.Type != html.ElementNode {
		return
	}

	tag := node.Data

	switch tag {
	case "strong", "b":
		text := getInnerText(node)
		if strings.TrimSpace(text) != "" {
			fmt.Fprintf(sb, "**%s**", text)
		}
	case "em", "i":
		text := getInnerText(node)
		if strings.TrimSpace(text) != "" {
			fmt.Fprintf(sb, "*%s*", text)
		}
	case "a":
		text := getInnerText(node)
		href := getAttr(node, "href")
		if href != "" {
			if strings.HasPrefix(href, "/") {
				href = "https://www.expressvpn.com" + href
			}
			fmt.Fprintf(sb, "[%s](%s)", text, href)
		} else {
			sb.WriteString(text)
		}
	case "code":
		fmt.Fprintf(sb, "`%s`", getInnerText(node))
	case "img":
		src := getAttr(node, "src")
		alt := getAttr(node, "alt")
		if src != "" {
			fmt.Fprintf(sb, "![%s](%s)", alt, src)
		}
	case "br":
		sb.WriteString("\n")
	default:
		// Recurse into children
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractInlineNode(sb, c)
		}
	}
}

func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func extractText(sel *goquery.Selection) string {
	return strings.TrimSpace(sel.Text())
}
