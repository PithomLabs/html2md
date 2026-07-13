package metadata

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	OgTitle     string   `yaml:"og_title,omitempty"`
	OgDesc      string   `yaml:"og_description,omitempty"`
	OgImage     string   `yaml:"og_image,omitempty"`
	OgURL       string   `yaml:"og_url,omitempty"`
	Canonical   string   `yaml:"canonical,omitempty"`
	Author      string   `yaml:"author,omitempty"`
	Date        string   `yaml:"date,omitempty"`
	ReadingTime string   `yaml:"reading_time,omitempty"`
	SourceFile  string   `yaml:"source_file,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	FAQ         []QA     `yaml:"faq,omitempty"`
}

type QA struct {
	Question string `yaml:"question"`
	Answer   string `yaml:"answer"`
}

func Extract(doc *goquery.Document, sourceFile string) Frontmatter {
	fm := Frontmatter{SourceFile: sourceFile}

	fm.Title = extractTitle(doc)
	fm.Description = extractMeta(doc, "description")
	fm.OgTitle = extractMeta(doc, "og:title")
	fm.OgDesc = extractMeta(doc, "og:description")
	fm.OgImage = extractMeta(doc, "og:image")
	fm.OgURL = extractMeta(doc, "og:url")
	fm.Canonical = extractCanonical(doc)
	fm.Author = extractAuthor(doc)
	fm.Date = extractDate(doc)
	fm.ReadingTime = extractReadingTime(doc)
	fm.FAQ = extractFAQ(doc)

	return fm
}

func extractTitle(doc *goquery.Document) string {
	if t := doc.Find("title").Text(); t != "" {
		return strings.TrimSpace(t)
	}
	if h1 := doc.Find("h1").First().Text(); h1 != "" {
		return strings.TrimSpace(h1)
	}
	return ""
}

func extractMeta(doc *goquery.Document, name string) string {
	// Try property first (og:*), then name
	sel := doc.Find("meta[property=\"" + name + "\"]")
	if v, exists := sel.Attr("content"); exists {
		return strings.TrimSpace(v)
	}
	sel = doc.Find("meta[name=\"" + name + "\"]")
	if v, exists := sel.Attr("content"); exists {
		return strings.TrimSpace(v)
	}
	return ""
}

func extractCanonical(doc *goquery.Document) string {
	if v, exists := doc.Find("link[rel=canonical]").Attr("href"); exists {
		return strings.TrimSpace(v)
	}
	return ""
}

func extractAuthor(doc *goquery.Document) string {
	// Try JSON-LD first
	doc.Find("script[type=\"application/ld+json\"]").Each(func(_ int, s *goquery.Selection) {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s.Text()), &data); err != nil {
			return
		}
		if author, ok := data["author"].(map[string]interface{}); ok {
			if name, ok := author["name"].(string); ok {
				if name != "" {
					return
				}
			}
		}
	})

	// Try common author selectors
	for _, sel := range []string{
		".post-heading__info a",
		".author-bio .name",
		"[rel=author]",
		".info__name",
	} {
		if t := doc.Find(sel).First().Text(); t != "" {
			return strings.TrimSpace(t)
		}
	}
	return ""
}

func extractDate(doc *goquery.Document) string {
	// Try JSON-LD
	doc.Find("script[type=\"application/ld+json\"]").Each(func(_ int, s *goquery.Selection) {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s.Text()), &data); err != nil {
			return
		}
		if d, ok := data["datePublished"].(string); ok && d != "" {
			return
		}
	})

	// Try meta tags
	if v := extractMeta(doc, "article:published_time"); v != "" {
		return v
	}
	if v := extractMeta(doc, "date"); v != "" {
		return v
	}

	// Try common date selectors
	for _, sel := range []string{
		".info__date",
		"time[datetime]",
		".post-date",
		".entry-date",
	} {
		s := doc.Find(sel).First()
		if t := s.Text(); t != "" {
			return strings.TrimSpace(t)
		}
		if dt, exists := s.Attr("datetime"); exists {
			return strings.TrimSpace(dt)
		}
	}
	return ""
}

func extractReadingTime(doc *goquery.Document) string {
	if t := doc.Find(".info__time").First().Text(); t != "" {
		return strings.TrimSpace(t)
	}
	return ""
}

func extractFAQ(doc *goquery.Document) []QA {
	var faqs []QA

	doc.Find("script[type=\"application/ld+json\"]").Each(func(_ int, s *goquery.Selection) {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s.Text()), &data); err != nil {
			return
		}
		if t, ok := data["@type"].(string); ok && t != "FAQPage" {
			return
		}
		mainEntity, ok := data["mainEntity"].([]interface{})
		if !ok {
			return
		}
		for _, item := range mainEntity {
			entity, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			q, _ := entity["name"].(string)
			answer, ok := entity["acceptedAnswer"].(map[string]interface{})
			if !ok {
				continue
			}
			a, _ := answer["text"].(string)
			if q != "" && a != "" {
				faqs = append(faqs, QA{Question: q, Answer: a})
			}
		}
	})

	// Fallback: try details/summary elements
	if len(faqs) == 0 {
		doc.Find("details.faq-item").Each(func(_ int, s *goquery.Selection) {
			q := strings.TrimSpace(s.Find("summary").Text())
			a := strings.TrimSpace(s.Find(".faq-content").Text())
			if q != "" && a != "" {
				faqs = append(faqs, QA{Question: q, Answer: a})
			}
		})
	}

	return faqs
}
