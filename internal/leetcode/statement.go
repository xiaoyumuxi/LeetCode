package leetcode

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"strings"
)

// PlainStatement keeps text/code and block boundaries, dropping active HTML,
// remote images and links. It never inserts site markup into a WebView.
func PlainStatement(content string) string {
	z := html.NewTokenizer(strings.NewReader(content))
	var out strings.Builder
	skip := 0
	for {
		token := z.Next()
		if token == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			break
		}
		switch token {
		case html.StartTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if tag == "script" || tag == "style" || tag == "iframe" {
				skip++
			}
			if skip == 0 && block(tag) {
				out.WriteString("\n")
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if tag == "script" || tag == "style" || tag == "iframe" {
				if skip > 0 {
					skip--
				}
				continue
			}
			if skip == 0 && block(tag) {
				out.WriteString("\n")
			}
		case html.SelfClosingTagToken:
			name, _ := z.TagName()
			if skip == 0 && string(name) == "br" {
				out.WriteString("\n")
			}
		case html.TextToken:
			if skip == 0 {
				out.Write(z.Text())
			}
		}
	}
	lines := strings.Split(strings.ReplaceAll(out.String(), "\u00a0", " "), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(line) == "" {
			if len(cleaned) == 0 || cleaned[len(cleaned)-1] == "" {
				continue
			}
			line = ""
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
func block(tag string) bool {
	switch tag {
	case "p", "div", "pre", "li", "ul", "ol", "h1", "h2", "h3", "br", "table", "tr":
		return true
	}
	return false
}

// Text is passed as HTML-escaped text to the host's read-only HTML preview.
func Statement(d Detail, site Site, zh bool) string {
	content := d.Content
	if zh && d.ContentCN != "" {
		content = d.ContentCN
	}
	return fmt.Sprintf("%s. %s\n%s\n\n%s", d.FrontendID, d.Name(zh), site.Origin()+"/problems/"+d.Slug+"/", PlainStatement(content))
}
