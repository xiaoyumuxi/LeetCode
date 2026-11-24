package leetcode

import (
	"context"
	"regexp"
	"strconv"
)

const CommunityPageSize = 10

var articleSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,239}$`)

type Article struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  struct {
		Username string `json:"username"`
	} `json:"author"`
}

type Articles struct {
	Total int
	Items []Article
}

type Comment struct {
	ID   string `json:"id"`
	Post struct {
		Content string `json:"content"`
		Author  struct {
			Username string `json:"username"`
		} `json:"author"`
	} `json:"post"`
}

type Comments struct {
	Total int
	Items []Comment
}

// Community APIs currently expose different models on the two sites. The
// China adapter uses its public article and bound-topic APIs; the global
// adapter is not claimed until its newer topic API can be verified.
func (c Client) Articles(ctx context.Context, slug string, offset int) (Articles, error) {
	if c.Site != CN {
		return Articles{}, problem("communitySite")
	}
	if !ValidSlug(slug) || offset < 0 || offset > 10000 {
		return Articles{}, problem("invalidRequest")
	}
	var result struct {
		Page *struct {
			Total int `json:"totalNum"`
			Edges []struct {
				Node Article `json:"node"`
			} `json:"edges"`
		} `json:"questionSolutionArticles"`
	}
	err := c.graph(ctx, `query questionSolutions($slug: String!, $skip: Int!, $first: Int!) { questionSolutionArticles(questionSlug: $slug, skip: $skip, first: $first) { totalNum edges { node { slug title author { username } } } } }`, map[string]any{"slug": slug, "skip": offset, "first": CommunityPageSize}, &result)
	if err != nil {
		return Articles{}, err
	}
	if result.Page == nil || result.Page.Total < 0 || len(result.Page.Edges) > CommunityPageSize {
		return Articles{}, problem("schemaChanged")
	}
	page := Articles{Total: result.Page.Total}
	for _, edge := range result.Page.Edges {
		if !articleSlugPattern.MatchString(edge.Node.Slug) || edge.Node.Title == "" {
			return Articles{}, problem("schemaChanged")
		}
		page.Items = append(page.Items, edge.Node)
	}
	return page, nil
}

func (c Client) Article(ctx context.Context, slug string) (Article, error) {
	if c.Site != CN {
		return Article{}, problem("communitySite")
	}
	if !articleSlugPattern.MatchString(slug) {
		return Article{}, problem("invalidRequest")
	}
	var result struct {
		Article *Article `json:"solutionArticle"`
	}
	err := c.graph(ctx, `query solutionArticle($slug: String!) { solutionArticle(slug: $slug) { slug title content author { username } } }`, map[string]any{"slug": slug}, &result)
	if err != nil {
		return Article{}, err
	}
	if result.Article == nil {
		return Article{}, problem("notFound")
	}
	if result.Article.Slug != slug || result.Article.Content == "" {
		return Article{}, problem("schemaChanged")
	}
	return *result.Article, nil
}

func (c Client) Comments(ctx context.Context, slug string, offset int) (Comments, error) {
	if c.Site != CN {
		return Comments{}, problem("communitySite")
	}
	if !ValidSlug(slug) || offset < 0 || offset > 10000 {
		return Comments{}, problem("invalidRequest")
	}
	var question struct {
		Question *struct {
			Topic int `json:"boundTopicId"`
		} `json:"question"`
	}
	if err := c.graph(ctx, `query discussionTopic($slug: String!) { question(titleSlug:$slug) { boundTopicId } }`, map[string]any{"slug": slug}, &question); err != nil {
		return Comments{}, err
	}
	if question.Question == nil || question.Question.Topic <= 0 {
		return Comments{}, problem("notFound")
	}
	var result struct {
		Page *struct {
			Total int `json:"totalNum"`
			Edges []struct {
				Node Comment `json:"node"`
			} `json:"edges"`
		} `json:"commonTopicComments"`
	}
	err := c.graph(ctx, `query problemComments($topic: Int!, $skip: Int!, $first: Int!) { commonTopicComments(topicId:$topic, skip:$skip, first:$first, orderBy:HOT) { totalNum edges { node { id post { content author { username } } } } } }`, map[string]any{"topic": question.Question.Topic, "skip": offset, "first": CommunityPageSize}, &result)
	if err != nil {
		return Comments{}, err
	}
	if result.Page == nil || result.Page.Total < 0 || len(result.Page.Edges) > CommunityPageSize {
		return Comments{}, problem("schemaChanged")
	}
	page := Comments{Total: result.Page.Total}
	for _, edge := range result.Page.Edges {
		if _, err := strconv.ParseUint(edge.Node.ID, 10, 64); err != nil {
			return Comments{}, problem("schemaChanged")
		}
		page.Items = append(page.Items, edge.Node)
	}
	return page, nil
}
