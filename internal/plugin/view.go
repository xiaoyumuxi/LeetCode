package plugin

import (
	"fmt"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
	"strings"
)

func snapshot(lang string, s *state) map[string]any {
	questions := make([]map[string]any, 0, len(s.questions))
	for _, q := range s.questions {
		questions = append(questions, map[string]any{"slug": q.Slug, "number": q.FrontendID, "title": q.Name(chinese(lang)), "difficulty": q.Difficulty, "paid": q.Paid, "status": q.Status})
	}
	title := s.selected
	for _, q := range s.questions {
		if q.Slug == s.selected {
			title = q.FrontendID + ". " + q.Name(chinese(lang))
			break
		}
	}
	if s.detail != nil {
		title = s.detail.FrontendID + ". " + s.detail.Name(chinese(lang))
	}
	text := map[string]string{}
	for key := range messages {
		text[key] = tr(lang, key)
	}
	languages := make([]map[string]string, 0, len(leetcode.Languages))
	for _, l := range leetcode.Languages {
		languages = append(languages, map[string]string{"id": l.ID, "name": l.Name})
	}
	var community any
	if page := s.community; page != nil {
		items := make([]map[string]string, 0, len(page.items))
		for _, item := range page.items {
			items = append(items, map[string]string{"slug": item.id, "title": item.title, "author": item.author})
		}
		community = map[string]any{"kind": page.kind, "items": items, "total": page.total, "hasMore": page.more, "cursor": fmt.Sprint(len(page.items))}
	}
	return map[string]any{"uiVersion": 1, "credentialStored": s.credentialStored, "site": s.site, "language": s.language, "query": s.search, "difficulty": s.difficulty, "status": s.status, "tag": s.tag, "questions": questions, "total": s.total, "loaded": s.loaded, "selected": s.selected, "title": singleLine(title, 200), "account": singleLine(s.account, 80), "pending": s.pending != nil, "notice": s.notice, "hasResult": s.lastReport != nil, "hasMore": s.more, "cursor": fmt.Sprint(len(s.questions)), "text": text, "languages": languages, "community": community}
}
func report(lang string, j leetcode.Job, v leetcode.Verdict) string {
	kind := "reportRun"
	if j.Submit {
		kind = "reportSubmit"
	}
	status := v.StatusMessage
	if status == "" {
		status = tr(lang, "result")
	}
	result := "failed"
	if verdictPassed(v) {
		result = "passed"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s #%s — %s\n%s\n", tr(lang, kind), j.ID, tr(lang, result), status)
	for _, entry := range []struct{ key, value string }{{"runtime", v.Runtime}, {"memory", v.Memory}, {"compiler", v.CompileError}, {"runtimeError", v.RuntimeError}, {"input", v.Input}, {"output", strings.Join(v.CodeAnswer, "\n")}, {"expected", strings.Join(v.ExpectedAnswer, "\n")}, {"stdout", strings.Join(v.Stdout, "\n")}} {
		if entry.value != "" {
			fmt.Fprintf(&b, "\n%s:\n%s\n", tr(lang, entry.key), entry.value)
		}
	}
	if v.Passed != nil && v.Total != nil {
		fmt.Fprintf(&b, "\n%s: %d / %d\n", tr(lang, "tests"), *v.Passed, *v.Total)
	}
	return b.String()
}
