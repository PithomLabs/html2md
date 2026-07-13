package converter

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ToMarkdown converts a cleaned goquery selection to markdown, writing to w.
// baseURL is used to resolve relative URLs (e.g. "/about" -> "https://example.com/about").
func ToMarkdown(w io.Writer, sel *goquery.Selection, baseURL string) {
	for i := 0; i < sel.Length(); i++ {
		node := sel.Eq(i)
		processNode(w, node.Children(), 0, baseURL)
	}
}

func processNode(w io.Writer, sel *goquery.Selection, depth int, baseURL string) {
	for i := 0; i < sel.Length(); i++ {
		node := sel.Eq(i)
		tag := goquery.NodeName(node)

		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			if goquery.NodeName(node.Parent()) == "li" {
				continue
			}
			writeHeading(w, node, tag)
		case "p":
			writeParagraph(w, node, baseURL)
		case "a":
			writeLink(w, node, baseURL)
		case "img":
			writeImage(w, node)
		case "ol":
			merged := collectConsecutiveOLs(sel, &i)
			if len(merged) > 1 {
				writeMergedList(w, merged, true, 0, baseURL)
			} else {
				writeList(w, node, true, 0, baseURL)
			}
		case "ul":
			writeList(w, node, false, 0, baseURL)
		case "table":
			writeTable(w, node)
		case "blockquote":
			writeBlockquote(w, node)
		case "pre":
			writeCodeBlock(w, node)
		case "code":
			if goquery.NodeName(node.Parent()) != "pre" {
				fmt.Fprintf(w, "`%s`", node.Text())
			}
		case "details":
			writeDetails(w, node, baseURL)
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
			processNode(w, node.Children(), depth, baseURL)
		default:
			text := strings.TrimSpace(node.Text())
			if text != "" && tag != "script" && tag != "style" {
				io.WriteString(w, text)
				io.WriteString(w, "\n")
			}
		}
	}
}

func collectConsecutiveOLs(sel *goquery.Selection, i *int) []*goquery.Selection {
	var ols []*goquery.Selection
	for j := *i; j < sel.Length(); j++ {
		node := sel.Eq(j)
		if goquery.NodeName(node) != "ol" {
			break
		}
		liCount := node.Find("> li").Length()
		if liCount == 0 {
			break
		}
		if len(ols) > 0 && liCount > 1 {
			break
		}
		ols = append(ols, node)
		*i = j
		if liCount > 1 {
			break
		}
	}
	return ols
}

func writeMergedList(w io.Writer, ols []*goquery.Selection, ordered bool, indent int, baseURL string) {
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
			text := extractInlineText(li, baseURL)
			io.WriteString(w, text)
			io.WriteString(w, "\n")
		})
	}
	io.WriteString(w, "\n")
}

func writeHeading(w io.Writer, sel *goquery.Selection, tag string) {
	level := int(tag[1] - '0')
	text := extractInlineText(sel, "")
	if strings.TrimSpace(text) == "" {
		return
	}
	io.WriteString(w, "\n")
	io.WriteString(w, strings.Repeat("#", level))
	io.WriteString(w, " ")
	io.WriteString(w, text)
	io.WriteString(w, "\n\n")
}

func writeParagraph(w io.Writer, sel *goquery.Selection, baseURL string) {
	text := extractInlineText(sel, baseURL)
	if strings.TrimSpace(text) == "" {
		return
	}
	io.WriteString(w, text)
	io.WriteString(w, "\n\n")
}

func writeLink(w io.Writer, sel *goquery.Selection, baseURL string) {
	text := extractInlineText(sel, baseURL)
	href, _ := sel.Attr("href")
	if href == "" {
		io.WriteString(w, text)
		return
	}
	href = resolveURL(href, baseURL)
	fmt.Fprintf(w, "[%s](%s)", text, href)
}

func writeImage(w io.Writer, sel *goquery.Selection) {
	src, _ := sel.Attr("src")
	alt, _ := sel.Attr("alt")
	if src == "" {
		return
	}
	if strings.Contains(src, "1x1") || strings.Contains(src, "pixel") || strings.Contains(src, "tracking") {
		return
	}
	fmt.Fprintf(w, "![%s](%s)\n\n", alt, src)
}

func writeList(w io.Writer, sel *goquery.Selection, ordered bool, indent int, baseURL string) {
	io.WriteString(w, "\n")
	counter := 0
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

		hasSubList := li.Find("ul, ol").Length() > 0

		if hasSubList {
			li.Children().Each(func(_ int, child *goquery.Selection) {
				childTag := goquery.NodeName(child)
				if childTag == "ul" || childTag == "ol" {
					writeList(w, child, childTag == "ol", indent+1, baseURL)
				} else {
					text := extractInlineText(child, baseURL)
					if strings.TrimSpace(text) != "" {
						io.WriteString(w, text)
					}
				}
			})
		} else {
			text := extractInlineText(li, baseURL)
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

	thead := sel.Find("thead tr")
	tbody := sel.Find("tbody tr")

	if thead.Length() > 0 {
		thead.Find("th").Each(func(_ int, th *goquery.Selection) {
			headers = append(headers, extractInlineText(th, ""))
		})
		tbody.Each(func(_ int, tr *goquery.Selection) {
			var row []string
			tr.Find("td").Each(func(_ int, td *goquery.Selection) {
				row = append(row, extractInlineText(td, ""))
			})
			dataRows = append(dataRows, row)
		})
	} else {
		firstRow := rows.First()
		firstRow.Find("th, td").Each(func(_ int, cell *goquery.Selection) {
			headers = append(headers, extractInlineText(cell, ""))
		})
		rows.Slice(1, rows.Length()).Each(func(_ int, tr *goquery.Selection) {
			var row []string
			tr.Find("td").Each(func(_ int, td *goquery.Selection) {
				row = append(row, extractInlineText(td, ""))
			})
			dataRows = append(dataRows, row)
		})
	}

	if len(headers) == 0 {
		return
	}

	io.WriteString(w, "| ")
	io.WriteString(w, strings.Join(headers, " | "))
	io.WriteString(w, " |\n")

	io.WriteString(w, "| ")
	for range headers {
		io.WriteString(w, "--- | ")
	}
	io.WriteString(w, "\n")

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
	text := extractInlineText(sel, "")
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

func writeDetails(w io.Writer, sel *goquery.Selection, baseURL string) {
	summary := strings.TrimSpace(sel.Find("summary").Text())
	content := extractInlineText(sel.Find(".faq-content, .details-content, div:not(summary)"), baseURL)

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

// resolveURL makes a relative URL absolute using baseURL.
// If href already has a scheme or baseURL is empty, href is returned as-is.
func resolveURL(href, baseURL string) string {
	if baseURL == "" || !strings.HasPrefix(href, "/") {
		return href
	}
	return baseURL + href
}

// extractInlineText extracts text from a selection, preserving inline formatting.
func extractInlineText(sel *goquery.Selection, baseURL string) string {
	var sb strings.Builder
	for _, node := range sel.Nodes {
		extractInlineNode(&sb, node, baseURL)
	}
	return strings.TrimSpace(sb.String())
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
func extractInlineNode(sb *strings.Builder, node *html.Node, baseURL string) {
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
			href = resolveURL(href, baseURL)
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
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractInlineNode(sb, c, baseURL)
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
