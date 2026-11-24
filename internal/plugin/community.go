package plugin

import (
	"fmt"
	"strings"

	"github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go/lithe"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
)

const communityItemLimit = 100
const communityContentLimit = 256 * 1024

type communityItem struct {
	id, title, author, content string
}
type communityState struct {
	kind, slug string
	items      []communityItem
	total      int
	offset     int
	bytes      int
	more       bool
}

func loadCommunity(call *lithe.Context, s *state, lang, kind string, more bool) error {
	if !leetcode.ValidSlug(s.selected) {
		return uiError(lang, "selectFirst")
	}
	if err := require(call.Host(), &s.workspace, lang, "network/request", "credentials/get", "preview/openMarkdown"); err != nil {
		return err
	}
	c, err := clientFor(call.Host(), s.workspace, s.site)
	if err != nil {
		return err
	}
	page := &communityState{kind: kind, slug: s.selected}
	if more {
		if s.community == nil || !s.community.more || s.community.slug != s.selected {
			return uiError(lang, "invalidRequest")
		}
		previous := *s.community
		page = &previous
	}
	var items []communityItem
	if kind == "solutions" {
		result, err := c.Articles(call.Cancellation().Context(), s.selected, page.offset)
		if err != nil {
			return err
		}
		page.total = result.Total
		for _, article := range result.Items {
			items = append(items, communityItem{id: article.Slug, title: singleLine(article.Title, 200), author: singleLine(article.Author.Username, 80)})
		}
	} else {
		result, err := c.Comments(call.Cancellation().Context(), s.selected, page.offset)
		if err != nil {
			return err
		}
		page.total = result.Total
		for _, comment := range result.Items {
			items = append(items, communityItem{id: comment.ID, title: singleLine(comment.Post.Content, 160), author: singleLine(comment.Post.Author.Username, 80), content: comment.Post.Content})
		}
	}
	page.offset += len(items)
	seen := make(map[string]bool, len(page.items))
	for _, item := range page.items {
		seen[item.id] = true
	}
	limited := false
	for _, item := range items {
		if seen[item.id] {
			continue
		}
		size := len(item.id) + len(item.title) + len(item.author) + len(item.content)
		if len(page.items) >= communityItemLimit || page.bytes+size > communityContentLimit {
			limited = true
			break
		}
		page.items = append(page.items, item)
		page.bytes += size
		seen[item.id] = true
	}
	page.more = !limited && len(items) > 0 && page.offset < page.total && len(page.items) < communityItemLimit
	s.community = page
	s.notice = ""
	if limited || (len(page.items) == communityItemLimit && page.offset < page.total) {
		s.notice = tr(lang, "communityLimit")
	}
	return nil
}

func openCommunity(call *lithe.Context, s *state, lang, id string) error {
	page := s.community
	if page == nil || page.slug != s.selected {
		return uiError(lang, "selectFirst")
	}
	var chosen *communityItem
	for i := range page.items {
		if page.items[i].id == id {
			chosen = &page.items[i]
			break
		}
	}
	if chosen == nil {
		return uiError(lang, "invalidRequest")
	}
	if err := require(call.Host(), &s.workspace, lang, "preview/openMarkdown"); err != nil {
		return err
	}
	content, title := chosen.content, chosen.title
	if page.kind == "solutions" {
		if err := require(call.Host(), &s.workspace, lang, "network/request", "credentials/get"); err != nil {
			return err
		}
		client, err := clientFor(call.Host(), s.workspace, s.site)
		if err != nil {
			return err
		}
		article, err := client.Article(call.Cancellation().Context(), id)
		if err != nil {
			return err
		}
		content, title = article.Content, singleLine(article.Title, 100)
	}
	// The Markdown preview's renderer controls markup and remote resources.
	// Site text is user content; it is never injected into the plugin page.
	text := fmt.Sprintf("# %s\n\n%s\n\n%s\n\n---\n\n%s", markdownText(title), markdownText(chosen.author), s.site.Origin()+"/problems/"+s.selected+"/", content)
	if len(text) > 128*1024 {
		return uiError(lang, "tooLarge")
	}
	_, err := call.Host().Preview().OpenMarkdown(s.workspace, map[string]any{"id": "leetcode." + string(s.site) + "." + page.kind + "." + id, "title": singleLine(tr(lang, page.kind)+" · "+title, 70), "text": text, "presentation": "previewOnly"})
	return err
}

func markdownText(s string) string {
	return strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "#", "\\#", "`", "\\`").Replace(s)
}
