package extractor

import (
	"github.com/PuerkitoBio/goquery"
)

// RemoveScripts removes all script, style, noscript, and link[rel=stylesheet] elements.
func RemoveScripts(doc *goquery.Document) {
	doc.Find("script, style, noscript").Remove()
	doc.Find("link[rel=stylesheet]").Remove()
	doc.Find("meta").Remove()
}

// RemoveBoilerplate removes navigation, headers, footers, sidebars, promos, and other
// non-content elements that bloat HTML pages.
func RemoveBoilerplate(doc *goquery.Document) {
	// Navigation & header
	doc.Find("nav, .navbar, .header, .menu-toggle, .side-list, .navbar__dropdown").Remove()

	// Footer
	doc.Find("footer, .footer, .site-footer").Remove()

	// Sidebar
	doc.Find("#sidebar, aside, .aside-promo, .aside-list, .blog-tab").Remove()

	// Breadcrumbs
	doc.Find(".breadcrumbs").Remove()

	// Deal timers / promo banners
	doc.Find(".deal-timer, .promo-content, .cta-banner, .timer").Remove()

	// Author bio & interaction buttons
	doc.Find(".author-bio, .author-bio__control").Remove()

	// Tab/TOC navigation components
	doc.Find(".tabs-component").Remove()

	// Related posts / recommendations
	doc.Find(".related-posts, .recommended, .suggested").Remove()

	// Cookie banners / popups
	doc.Find(".cookie-banner, .cookie-notice, .popup, .modal, .overlay").Remove()

	// Post metadata (author, date, reading time, category)
	doc.Find(".post-heading__info, .post-heading .info, .info__date, .info__time, .info__link, .info__name").Remove()

	// Newsletter / subscription forms
	doc.Find("form, [type=submit], [name=email], .newsletter, .subscribe").Remove()

	// Footer sections that may appear in content
	doc.Find(".footer, footer, .site-footer, .copyright").Remove()

	// Partner/supporter logos and badges - remove the section containing them
	doc.Find(".logos").Remove()
	doc.Find("h2:contains('proudly supporting')").Parent().Remove()

	// SVG logos and decorative images (keep content images)
	doc.Find("svg").Remove()

	// Hide-only elements
	doc.Find("[aria-hidden=true]").Remove()

	// Empty divs that are just wrappers
	doc.Find("div").Each(func(_ int, s *goquery.Selection) {
		if s.Children().Length() == 0 && s.Text() == "" {
			s.Remove()
		}
	})
}

// GetContentRoot returns the best content container, falling back to <body>.
func GetContentRoot(doc *goquery.Document) *goquery.Selection {
	// Try common content selectors
	for _, sel := range []string{
		"main",
		"article",
		".content",
		"#content",
		".post-content",
		".entry-content",
		".single-content",
		"section.container",
		".container--md",
	} {
		if s := doc.Find(sel); s.Length() > 0 {
			return s
		}
	}
	return doc.Find("body")
}
