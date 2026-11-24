// Package leetcode adapts both sites without opening sockets; the host owns HTTP.
package leetcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Site string

const (
	CN  Site = "cn"
	COM Site = "com"
)

func (s Site) Valid() bool { return s == CN || s == COM }
func (s Site) Origin() string {
	if s == CN {
		return "https://leetcode.cn"
	}
	return "https://leetcode.com"
}

type Response struct {
	Status int
	Body   []byte
}

// Transport must never forward credentials across redirects.
type Transport interface {
	Do(context.Context, string, string, map[string]string, []byte) (Response, error)
}
type Credential struct {
	Session string `json:"session"`
	CSRF    string `json:"csrf"`
}

func (c Credential) Valid() bool {
	if len(c.Session) == 0 || len(c.Session) > 6000 || len(c.CSRF) == 0 || len(c.CSRF) > 256 {
		return false
	}
	for _, v := range []string{c.Session, c.CSRF} {
		for _, r := range v {
			if r < 33 || r > 126 || strings.ContainsRune(";,\"\\", r) {
				return false
			}
		}
	}
	return true
}

// Error contains a controlled category, never a response, URL or secret.
type Error struct {
	Kind string
	HTTP int
}

func (e *Error) Error() string  { return "leetcode: " + e.Kind }
func problem(kind string) error { return &Error{Kind: kind} }
func Category(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return "requestFailed"
}

type Client struct {
	Site       Site
	Transport  Transport
	Credential Credential
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var idPattern = regexp.MustCompile(`^[0-9]{1,12}$`)
var jobPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_-]{0,159}$`)

func ValidSlug(s string) bool  { return len(s) <= 180 && slugPattern.MatchString(s) }
func ValidJobID(s string) bool { return jobPattern.MatchString(s) }
func (c Client) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	if !c.Site.Valid() || c.Transport == nil || !strings.HasPrefix(path, "/") {
		return nil, problem("invalidRequest")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, problem("invalidRequest")
	}
	headers := map[string]string{"Accept": "application/json", "Content-Type": "application/json", "Referer": c.Site.Origin() + "/", "Origin": c.Site.Origin(), "User-Agent": "Lithe-LeetCode/0.0.1"}
	if c.Credential.Session != "" {
		if !c.Credential.Valid() {
			return nil, problem("invalidCredential")
		}
		headers["Cookie"] = "LEETCODE_SESSION=" + c.Credential.Session + "; csrftoken=" + c.Credential.CSRF
		headers["X-CSRFToken"] = c.Credential.CSRF
	}
	response, err := c.Transport.Do(ctx, method, c.Site.Origin()+path, headers, data)
	if err != nil {
		return nil, err
	}
	kind := ""
	switch {
	case response.Status == 401:
		kind = "signIn"
	case response.Status == 403:
		kind = "forbidden"
	case response.Status == 429:
		kind = "rateLimited"
	case response.Status < 200 || response.Status >= 300:
		kind = "http"
	}
	if kind != "" {
		return nil, &Error{Kind: kind, HTTP: response.Status}
	}
	return response.Body, nil
}
func (c Client) graph(ctx context.Context, query string, variables, out any) error {
	data, err := c.request(ctx, "POST", "/graphql/", map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	var envelope struct {
		Data   json.RawMessage   `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Errors) > 0 || len(envelope.Data) == 0 || string(envelope.Data) == "null" || json.Unmarshal(envelope.Data, out) != nil {
		return problem("schemaChanged")
	}
	return nil
}

type Question struct {
	ID              string `json:"questionId"`
	FrontendID      string `json:"questionFrontendId"`
	Title           string `json:"title"`
	TitleCN         string `json:"translatedTitle"`
	Slug            string `json:"titleSlug"`
	Difficulty      string `json:"difficulty"`
	Paid            bool   `json:"isPaidOnly"`
	Status          string `json:"status"`
	ChinaFrontendID string `json:"frontendQuestionId"`
	ChinaTitle      string `json:"titleCn"`
	ChinaPaid       bool   `json:"paidOnly"`
}

func (q Question) Name(zh bool) string {
	if zh && q.TitleCN != "" {
		return q.TitleCN
	}
	return q.Title
}

type Search struct {
	Query, Difficulty, Status, Tag string
	Offset, Limit                  int
}
type Page struct {
	Questions []Question
	Total     int
}

func (c Client) List(ctx context.Context, s Search) (Page, error) {
	if s.Limit < 1 || s.Limit > 50 || s.Offset < 0 || s.Offset > 10000 || len(s.Query) > 256 || len(s.Tag) > 100 {
		return Page{}, problem("invalidRequest")
	}
	filters := map[string]any{}
	if s.Query != "" {
		filters["searchKeywords"] = s.Query
	}
	if s.Difficulty != "" {
		filters["difficulty"] = s.Difficulty
	}
	if s.Status != "" {
		filters["status"] = s.Status
	}
	if s.Tag != "" {
		if !ValidSlug(s.Tag) {
			return Page{}, problem("invalidRequest")
		}
		filters["tags"] = []string{s.Tag}
	}
	root := "questionList"
	selection := "totalNum data { questionFrontendId title titleSlug difficulty isPaidOnly status }"
	if c.Site == CN {
		root = "problemsetQuestionList"
		selection = "total questions { frontendQuestionId title titleCn titleSlug difficulty paidOnly status }"
	}
	query := "query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) { " + root + "(categorySlug: $categorySlug, limit: $limit, skip: $skip, filters: $filters) { " + selection + " } }"
	type listResult struct {
		Questions []Question `json:"questions"`
		Total     int        `json:"total"`
		Data      []Question `json:"data"`
		TotalNum  int        `json:"totalNum"`
	}
	var result struct {
		China  *listResult `json:"problemsetQuestionList"`
		Global *listResult `json:"questionList"`
	}
	if err := c.graph(ctx, query, map[string]any{"categorySlug": "", "limit": s.Limit, "skip": s.Offset, "filters": filters}, &result); err != nil {
		return Page{}, err
	}
	page := result.China
	if page == nil {
		page = result.Global
	}
	if page != nil && page.Questions == nil {
		page.Questions, page.Total = page.Data, page.TotalNum
	}
	if page == nil || page.Questions == nil || page.Total < 0 || len(page.Questions) > s.Limit {
		return Page{}, problem("schemaChanged")
	}
	for i := range page.Questions {
		q := &page.Questions[i]
		// China list responses use their canonical names even for aliases.
		if q.FrontendID == "" {
			q.FrontendID = q.ChinaFrontendID
		}
		if q.TitleCN == "" {
			q.TitleCN = q.ChinaTitle
		}
		q.Paid = q.Paid || q.ChinaPaid
		q.Difficulty = strings.ToUpper(q.Difficulty)
		if !ValidSlug(q.Slug) || q.Title == "" || q.FrontendID == "" {
			return Page{}, problem("schemaChanged")
		}
	}
	return Page{page.Questions, page.Total}, nil
}

type Snippet struct {
	Language string `json:"langSlug"`
	Code     string `json:"code"`
}
type Detail struct {
	Question
	Content          string    `json:"content"`
	ContentCN        string    `json:"translatedContent"`
	Snippets         []Snippet `json:"codeSnippets"`
	SampleTestCase   string    `json:"sampleTestCase"`
	ExampleTestcases string    `json:"exampleTestcases"`
}

func (c Client) Detail(ctx context.Context, slug string) (Detail, error) {
	if !ValidSlug(slug) {
		return Detail{}, problem("invalidRequest")
	}
	var result struct {
		Question *Detail `json:"question"`
	}
	err := c.graph(ctx, `query questionData($titleSlug: String!) { question(titleSlug: $titleSlug) { questionId questionFrontendId title translatedTitle titleSlug difficulty isPaidOnly content translatedContent codeSnippets { langSlug code } sampleTestCase exampleTestcases } }`, map[string]any{"titleSlug": slug}, &result)
	if err != nil {
		return Detail{}, err
	}
	if result.Question == nil {
		return Detail{}, problem("notFound")
	}
	q := result.Question
	if q.Paid {
		return Detail{}, problem("paid")
	}
	if q.Slug != slug || !idPattern.MatchString(q.ID) || q.Content == "" {
		return Detail{}, problem("schemaChanged")
	}
	q.Difficulty = strings.ToUpper(q.Difficulty)
	return *q, nil
}
func (c Client) Daily(ctx context.Context) (string, error) {
	var result struct {
		Daily *struct {
			Question Question `json:"question"`
		} `json:"activeDailyCodingChallengeQuestion"`
		Today []struct {
			Question Question `json:"question"`
		} `json:"todayRecord"`
	}
	query := `query { activeDailyCodingChallengeQuestion { question { titleSlug } } }`
	if c.Site == CN {
		query = `query { todayRecord { question { titleSlug } } }`
	}
	if err := c.graph(ctx, query, map[string]any{}, &result); err != nil {
		return "", err
	}
	slug := ""
	if result.Daily != nil {
		slug = result.Daily.Question.Slug
	}
	if c.Site == CN && len(result.Today) > 0 {
		slug = result.Today[0].Question.Slug
	}
	if !ValidSlug(slug) {
		return "", problem("schemaChanged")
	}
	return slug, nil
}
func (c Client) Account(ctx context.Context) (string, error) {
	var result struct {
		Status *struct {
			SignedIn bool   `json:"isSignedIn"`
			Username string `json:"username"`
		} `json:"userStatus"`
	}
	if err := c.graph(ctx, `query { userStatus { isSignedIn username } }`, map[string]any{}, &result); err != nil {
		return "", err
	}
	if result.Status == nil {
		return "", problem("schemaChanged")
	}
	if !result.Status.SignedIn || result.Status.Username == "" {
		return "", problem("signIn")
	}
	return result.Status.Username, nil
}

type Job struct {
	ID     string
	Submit bool
}

func (c Client) Start(ctx context.Context, d Detail, language, code string, submit bool) (Job, error) {
	if !c.Credential.Valid() {
		return Job{}, problem("signIn")
	}
	if !ValidSlug(d.Slug) || !idPattern.MatchString(d.ID) || len(code) == 0 || len(code) > 96*1024 || !SupportedLanguage(language) {
		return Job{}, problem("invalidRequest")
	}
	payload := map[string]any{"lang": language, "question_id": d.ID, "typed_code": code}
	path := "/problems/" + d.Slug + "/interpret_solution/"
	if submit {
		path = "/problems/" + d.Slug + "/submit/"
	} else {
		tests := d.ExampleTestcases
		if tests == "" {
			tests = d.SampleTestCase
		}
		if tests == "" {
			return Job{}, problem("noSamples")
		}
		payload["data_input"] = tests
	}
	data, err := c.request(ctx, "POST", path, payload)
	if err != nil {
		return Job{}, err
	}
	var result struct {
		Interpret json.RawMessage `json:"interpret_id"`
		Submit    json.RawMessage `json:"submission_id"`
	}
	if json.Unmarshal(data, &result) != nil {
		return Job{}, problem("schemaChanged")
	}
	id := result.Interpret
	if submit {
		id = result.Submit
	}
	var text string
	if json.Unmarshal(id, &text) != nil {
		text = string(id)
	}
	if !ValidJobID(text) || (submit && !idPattern.MatchString(text)) {
		return Job{}, problem("schemaChanged")
	}
	return Job{text, submit}, nil
}

type Verdict struct {
	State          string   `json:"state"`
	StatusCode     int      `json:"status_code"`
	StatusMessage  string   `json:"status_msg"`
	Runtime        string   `json:"status_runtime"`
	Memory         string   `json:"status_memory"`
	CompileError   string   `json:"full_compile_error"`
	RuntimeError   string   `json:"full_runtime_error"`
	CodeAnswer     []string `json:"code_answer"`
	ExpectedAnswer []string `json:"expected_code_answer"`
	Stdout         []string `json:"std_output_list"`
	Input          string   `json:"last_testcase"`
	Passed         *int     `json:"total_correct"`
	Total          *int     `json:"total_testcases"`
	Correct        *bool    `json:"correct_answer"`
}

func (c Client) Check(ctx context.Context, j Job) (Verdict, error) {
	if !c.Credential.Valid() {
		return Verdict{}, problem("signIn")
	}
	if !ValidJobID(j.ID) {
		return Verdict{}, problem("invalidRequest")
	}
	data, err := c.request(ctx, "GET", "/submissions/detail/"+url.PathEscape(j.ID)+"/check/", nil)
	if err != nil {
		return Verdict{}, err
	}
	var v Verdict
	if json.Unmarshal(data, &v) != nil || (v.State != "PENDING" && v.State != "STARTED" && v.State != "SUCCESS") {
		return Verdict{}, problem("schemaChanged")
	}
	return v, nil
}

type Language struct{ ID, Name, File, Comment string }

var Languages = []Language{{"python3", "Python 3", "solution.py", "#"}, {"java", "Java", "Solution.java", "//"}, {"cpp", "C++", "solution.cpp", "//"}, {"golang", "Go", "solution.go", "//"}, {"rust", "Rust", "solution.rs", "//"}, {"javascript", "JavaScript", "solution.js", "//"}, {"typescript", "TypeScript", "solution.ts", "//"}, {"csharp", "C#", "Solution.cs", "//"}}

func LanguageFor(id string) (Language, bool) {
	for _, l := range Languages {
		if l.ID == id {
			return l, true
		}
	}
	return Language{}, false
}
func SupportedLanguage(id string) bool { _, ok := LanguageFor(id); return ok }

// Marker travels with the source and is checked against the current site before upload.
func Marker(site Site, slug, language string) string {
	return fmt.Sprintf("@lithe-leetcode site=%s slug=%s lang=%s", site, slug, language)
}

var markerPattern = regexp.MustCompile(`^\s*(?://|#) @lithe-leetcode site=(cn|com) slug=([a-z0-9-]+) lang=([a-z0-9]+)\r?$`)

func ParseSolution(text string) (Site, string, string, error) {
	first, _, _ := strings.Cut(text, "\n")
	m := markerPattern.FindStringSubmatch(first)
	if m == nil || !ValidSlug(m[2]) || !SupportedLanguage(m[3]) {
		return "", "", "", problem("notSolution")
	}
	return Site(m[1]), m[2], m[3], nil
}
func Solution(d Detail, site Site, language string) (string, error) {
	l, ok := LanguageFor(language)
	if !ok {
		return "", problem("noSnippet")
	}
	for _, s := range d.Snippets {
		if s.Language == language && s.Code != "" {
			return l.Comment + " " + Marker(site, d.Slug, language) + "\n" + l.Comment + " " + site.Origin() + "/problems/" + d.Slug + "/\n\n" + s.Code + "\n", nil
		}
	}
	return "", problem("noSnippet")
}
