package leetcode

import (
	"context"
	"strings"
	"testing"
)

func TestCommunityArticleSlugMayEndInHyphen(t *testing.T) {
	client := Client{Site: CN, Transport: transportFunc(func(_ context.Context, _, _ string, _ map[string]string, body []byte) (Response, error) {
		if strings.Contains(string(body), "questionSolutionArticles") {
			return Response{200, []byte(`{"data":{"questionSolutionArticles":{"totalNum":1,"edges":[{"node":{"slug":"protocol-article-","title":"Protocol boundary"}}]}}}`)}, nil
		}
		return Response{200, []byte(`{"data":{"solutionArticle":{"slug":"protocol-article-","title":"Protocol boundary","content":"# Content"}}}`)}, nil
	})}
	page, err := client.Articles(t.Context(), "protocol-fixture", 0)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("valid article slug rejected: %+v, %v", page, err)
	}
	article, err := client.Article(t.Context(), page.Items[0].Slug)
	if err != nil || article.Content != "# Content" {
		t.Fatalf("article could not be opened: %+v, %v", article, err)
	}
}
