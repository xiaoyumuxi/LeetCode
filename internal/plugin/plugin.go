// Package plugin composes the plugin page with public host-owned services.
package plugin

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"sync"
	"time"

	"github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go/lithe"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
)

const PluginID = "io.github.lithe-idea.leetcode"
const Version = "0.0.1"

type state struct {
	mu                                        sync.Mutex
	workspace                                 lithe.WorkspaceContext
	site                                      leetcode.Site
	language, search, difficulty, status, tag string
	questions                                 []leetcode.Question
	total                                     int
	selected                                  string
	detail                                    *leetcode.Detail
	account                                   string
	credentialStored                          bool
	loaded                                    bool
	pending                                   *leetcode.Job
	notice                                    string
	lastReport                                *judgeReport
	directory                                 string
	initialized                               bool
	saved                                     string
	nextOffset                                int
	more                                      bool
	community                                 *communityState
}
type Plugin struct {
	mu        sync.Mutex
	instances map[string]*state
}
type request struct {
	Instance string `json:"instance"`
	Action   string `json:"action"`
	Language string `json:"language"`
	Value    string `json:"value"`
	Item     string `json:"item"`
	Cursor   string `json:"cursor"`
}

func NewBackend() *lithe.Backend {
	b := lithe.NewBackend("lithe-leetcode").Version(Version)
	p := &Plugin{instances: make(map[string]*state)}
	must(b.Function("leetcode.dispatch", p.dispatch))
	b.OnContextsChanged(func(_ json.RawMessage) error {
		raw, err := b.Handle().Host().Workspace().ListContexts()
		if err != nil {
			return err
		}
		var list struct {
			Contexts []lithe.WorkspaceContext `json:"contexts"`
		}
		if json.Unmarshal(raw, &list) != nil {
			return uiError("en", "requestFailed")
		}
		live := map[string]bool{}
		for _, w := range list.Contexts {
			live[w.ID()+":"+strconv.FormatUint(w.Generation(), 10)] = true
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		for key := range p.instances {
			if !live[key] {
				delete(p.instances, key)
			}
		}
		return nil
	})
	return b
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func (p *Plugin) dispatch(raw json.RawMessage, call *lithe.Context) (any, error) {
	var r request
	if json.Unmarshal(raw, &r) != nil || !lithe.IsCredentialKey(r.Instance) || len(r.Language) > 32 {
		return nil, uiError("en", "invalidRequest")
	}
	switch r.Action {
	case "login", "connect", "logout", "create", "run", "submit":
	default:
		call.SuppressSuccessDiagnostics()
	}
	workspace := call.WorkspaceContext()
	if workspace == nil {
		return nil, uiError(r.Language, "noWorkspace")
	}
	key := workspace.ID() + ":" + strconv.FormatUint(workspace.Generation(), 10)
	p.mu.Lock()
	// A context belongs to one workspace window. Recreating its web page must
	// retain a pending job and must not create a second submission lane.
	s := p.instances[key]
	if s == nil && r.Action == "open" && len(p.instances) < 32 {
		site := leetcode.COM
		if chinese(r.Language) {
			site = leetcode.CN
		}
		s = &state{site: site, language: "python3", workspace: *workspace}
		p.instances[key] = s
	}
	if s == nil {
		p.mu.Unlock()
		return nil, uiError(r.Language, "requestFailed")
	}
	if !s.mu.TryLock() {
		p.mu.Unlock()
		return nil, uiError(r.Language, "busy")
	}
	p.mu.Unlock()
	defer s.mu.Unlock()
	if s.workspace.ID() != workspace.ID() || s.workspace.Generation() != workspace.Generation() {
		return nil, uiError(r.Language, "noWorkspace")
	}
	if !s.initialized {
		if err := restore(call, s, r.Language); err != nil {
			return nil, safeError(r.Language, err)
		}
	}
	if err := call.CheckCancelled(); err != nil {
		return nil, err
	}
	if r.Action != "open" {
		e := &lithe.ViewEvent{Kind: lithe.ViewClick, Language: r.Language, Node: r.Action, Value: r.Value, Item: r.Item, Cursor: r.Cursor}
		switch r.Action {
		case "site", "language", "search", "difficulty", "status", "tag":
			e.Kind = lithe.ViewChange
		case "select":
			e.Kind = lithe.ViewSelect
			e.Node = "questions"
		case "more":
			e.Kind = lithe.ViewLoadMore
			e.Node = "questions"
		case "activate":
			e.Kind = lithe.ViewActivate
			e.Node = "questions"
		case "refresh", "daily", "preview", "website", "create", "login", "connect", "logout", "account", "run", "submit", "check", "result", "solutions", "discussion", "communityMore", "communityOpen", "problems":
		default:
			return nil, uiError(r.Language, "invalidRequest")
		}
		if err := p.event(e, call, s); err != nil {
			return nil, safeError(r.Language, err)
		}
	}
	if err := persist(s); err != nil {
		return nil, uiError(r.Language, "stateFailed")
	}
	return snapshot(r.Language, s), nil
}
func (p *Plugin) event(e *lithe.ViewEvent, call *lithe.Context, s *state) error {
	host := call.Host()
	w := s.workspace
	lang := e.Language
	ctx := call.Cancellation().Context()
	if e.Kind == lithe.ViewSelect && e.Node == "questions" {
		for _, q := range s.questions {
			if q.Slug == e.Item {
				s.selected = q.Slug
				s.detail = nil
				s.community = nil
				break
			}
		}
		return nil
	}
	if e.Kind == lithe.ViewChange {
		value, _ := e.Value.(string)
		switch e.Node {
		case "site":
			site := leetcode.Site(value)
			if !site.Valid() {
				return uiError(lang, "invalidRequest")
			}
			if s.pending != nil {
				return uiError(lang, "pendingExists")
			}
			s.site = site
			s.questions = nil
			s.selected = ""
			s.detail = nil
			s.community = nil
			s.loaded = false
			s.account = ""
			client, err := clientFor(host, w, site)
			if err != nil {
				return err
			}
			s.credentialStored = client.Credential.Valid()
			s.notice = ""
			s.lastReport = nil
			s.more = false
			return nil
		case "language":
			if !leetcode.SupportedLanguage(value) {
				return uiError(lang, "invalidRequest")
			}
			s.language = value
			return nil
		case "search":
			if len(value) > 256 {
				return uiError(lang, "invalidRequest")
			}
			s.search = value
		case "difficulty":
			if value != "" && value != "EASY" && value != "MEDIUM" && value != "HARD" {
				return uiError(lang, "invalidRequest")
			}
			s.difficulty = value
		case "status":
			if value != "" && value != "AC" && value != "TRIED" && value != "NOT_STARTED" {
				return uiError(lang, "invalidRequest")
			}
			s.status = value
		case "tag":
			s.tag = value
		default:
			return nil
		}
		return load(call, s, e.Language, false)
	}
	if e.Kind == lithe.ViewLoadMore {
		if e.Node != "questions" || e.Cursor != strconv.Itoa(len(s.questions)) {
			return nil
		}
		return load(call, s, e.Language, true)
	}
	if e.Kind == lithe.ViewRefresh {
		if s.loaded {
			return load(call, s, e.Language, false)
		}
		return nil
	}
	if e.Kind == lithe.ViewActivate && e.Node == "questions" {
		s.selected = e.Item
		s.detail = nil
		s.community = nil
		return preview(call, s, lang)
	}
	if e.Kind != lithe.ViewClick {
		return nil
	}
	switch e.Node {
	case "result":
		if s.lastReport == nil {
			return uiError(lang, "noResult")
		}
		return showRunReport(call.Host(), s, lang)
	case "solutions", "discussion":
		return loadCommunity(call, s, lang, e.Node, false)
	case "communityMore":
		if s.community == nil || e.Cursor != strconv.Itoa(len(s.community.items)) {
			return uiError(lang, "invalidRequest")
		}
		return loadCommunity(call, s, lang, s.community.kind, true)
	case "communityOpen":
		return openCommunity(call, s, lang, e.Item)
	case "problems":
		s.community = nil
		return nil
	case "refresh":
		return load(call, s, e.Language, false)
	case "daily":
		if err := require(host, &w, lang, "network/request", "credentials/get"); err != nil {
			return err
		}
		client, err := clientFor(host, w, s.site)
		if err != nil {
			return err
		}
		slug, err := client.Daily(ctx)
		if err != nil {
			return err
		}
		s.selected = slug
		s.detail = nil
		s.community = nil
		return preview(call, s, lang)
	case "preview":
		return preview(call, s, lang)
	case "website":
		if err := require(host, &w, lang, "browser/openExternal"); err != nil {
			return err
		}
		url := s.site.Origin() + "/problemset/"
		if leetcode.ValidSlug(s.selected) {
			url = s.site.Origin() + "/problems/" + s.selected + "/"
		}
		_, err := host.Browser().OpenExternal(w, map[string]any{"url": url})
		return err
	case "create":
		return createSolution(call, s, lang)
	case "login":
		if err := require(host, &w, lang, "browser/openExternal"); err != nil {
			return err
		}
		_, err := host.Browser().OpenExternal(w, map[string]any{"url": s.site.Origin() + "/accounts/login/"})
		if err == nil {
			s.notice = tr(lang, "browserLoginOpened")
		}
		return err
	case "connect":
		if s.pending != nil {
			return uiError(lang, "pendingExists")
		}
		return connectAccount(call, s, lang)
	case "account":
		if err := require(host, &w, lang, "network/request", "credentials/get"); err != nil {
			return err
		}
		client, err := clientFor(host, w, s.site)
		if err != nil {
			return err
		}
		name, err := client.Account(ctx)
		if err != nil {
			return err
		}
		s.account = name
		return nil
	case "logout":
		if s.pending != nil {
			return uiError(lang, "pendingExists")
		}
		if err := require(host, &w, lang, "credentials/delete"); err != nil {
			return err
		}
		if err := host.Credentials().Delete(credentialKey(s.site)); err != nil {
			return err
		}
		s.account = ""
		s.credentialStored = false
		s.questions = nil
		s.community = nil
		s.loaded = false
		s.more = false
		s.notice = ""
		return nil
	case "run", "submit":
		return judge(call, s, lang, e.Node == "submit")
	case "check":
		if s.pending == nil {
			return uiError(lang, "noPending")
		}
		return poll(call, s, lang)
	}
	return nil
}
func load(call *lithe.Context, s *state, lang string, more bool) error {
	host := call.Host()
	if err := require(host, &s.workspace, lang, "network/request", "credentials/get"); err != nil {
		return err
	}
	offset := 0
	if more {
		offset = s.nextOffset
		if len(s.questions) >= 400 || !s.more {
			return nil
		}
	}
	client, err := clientFor(host, s.workspace, s.site)
	if err != nil {
		return err
	}
	if s.status != "" && !client.Credential.Valid() {
		return &leetcode.Error{Kind: "signIn"}
	}
	page, err := client.List(call.Cancellation().Context(), leetcode.Search{Query: s.search, Difficulty: s.difficulty, Status: s.status, Tag: s.tag, Offset: offset, Limit: 40})
	if err != nil {
		return err
	}
	if !more {
		s.questions = nil
	}
	seen := make(map[string]bool)
	for _, q := range s.questions {
		seen[q.Slug] = true
	}
	for _, q := range page.Questions {
		if !seen[q.Slug] {
			s.questions = append(s.questions, q)
			seen[q.Slug] = true
		}
	}
	s.total = page.Total
	s.community = nil
	s.notice = ""
	s.nextOffset = offset + len(page.Questions)
	s.more = len(page.Questions) > 0 && s.nextOffset < page.Total && len(s.questions) < 400
	s.loaded = true
	return nil
}
func detail(call *lithe.Context, s *state, lang string) (leetcode.Detail, error) {
	if s.detail != nil && s.detail.Slug == s.selected {
		return *s.detail, nil
	}
	if !leetcode.ValidSlug(s.selected) {
		return leetcode.Detail{}, &leetcode.Error{Kind: "selectFirst"}
	}
	if err := require(call.Host(), &s.workspace, lang, "network/request", "credentials/get"); err != nil {
		return leetcode.Detail{}, err
	}
	c, err := clientFor(call.Host(), s.workspace, s.site)
	if err != nil {
		return leetcode.Detail{}, err
	}
	d, err := c.Detail(call.Cancellation().Context(), s.selected)
	if err == nil {
		s.detail = &d
	}
	return d, err
}
func preview(call *lithe.Context, s *state, lang string) error {
	if err := require(call.Host(), &s.workspace, lang, "preview/openHTML"); err != nil {
		return err
	}
	d, err := detail(call, s, lang)
	if err != nil {
		return err
	}
	text := leetcode.Statement(d, s.site, chinese(lang))
	page := `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'"><style>body{font:16px system-ui;margin:24px;line-height:1.6;color-scheme:light dark}pre{white-space:pre-wrap;overflow-wrap:anywhere;font:inherit}</style></head><body><pre>` + html.EscapeString(text) + `</pre></body></html>`
	if len(page) > 128*1024 {
		return uiError(lang, "tooLarge")
	}
	_, err = call.Host().Preview().OpenHTML(s.workspace, map[string]any{"id": "leetcode." + string(s.site) + "." + d.ID, "title": singleLine(d.FrontendID+". "+d.Name(chinese(lang)), 70), "text": page, "presentation": "previewOnly"})
	return err
}
func connectAccount(call *lithe.Context, s *state, lang string) error {
	host := call.Host()
	if err := require(host, &s.workspace, lang, "window/showInputBox", "credentials/set", "network/request"); err != nil {
		return err
	}
	var values []string
	for _, key := range []string{"session", "csrf"} {
		raw, err := host.Window().ShowInputBox(s.workspace, map[string]any{"title": tr(lang, key+"Title"), "prompt": tr(lang, key+"Prompt"), "password": true})
		if err != nil {
			return err
		}
		if string(raw) == "null" {
			return lithe.ErrCancelled
		}
		var result struct {
			Value string `json:"value"`
		}
		if json.Unmarshal(raw, &result) != nil {
			return uiError(lang, "requestFailed")
		}
		values = append(values, result.Value)
	}
	credential := leetcode.Credential{Session: values[0], CSRF: values[1]}
	if !credential.Valid() {
		return uiError(lang, "invalidCredential")
	}
	client := leetcode.Client{Site: s.site, Credential: credential, Transport: hostTransport{host, s.workspace}}
	name, err := client.Account(call.Cancellation().Context())
	if err != nil {
		return err
	}
	secret, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	if err = host.Credentials().Set(credentialKey(s.site), string(secret)); err != nil {
		return err
	}
	s.account = name
	s.credentialStored = true
	s.notice = ""
	s.questions = nil
	s.community = nil
	s.loaded = false
	s.more = false
	return nil
}
func createSolution(call *lithe.Context, s *state, lang string) error {
	host := call.Host()
	w := s.workspace
	if err := require(host, &w, lang, "workspace/getFolders", "workspace/create", "editor/open", "window/showQuickPick"); err != nil {
		return err
	}
	d, err := detail(call, s, lang)
	if err != nil {
		return err
	}
	text, err := leetcode.Solution(d, s.site, s.language)
	if err != nil {
		return err
	}
	language, _ := leetcode.LanguageFor(s.language)
	raw, err := host.Workspace().GetFolders(w)
	if err != nil {
		return err
	}
	var result struct {
		Folders []struct {
			ID   string `json:"rootID"`
			Name string `json:"name"`
		} `json:"folders"`
	}
	if json.Unmarshal(raw, &result) != nil || len(result.Folders) == 0 {
		return uiError(lang, "noWorkspace")
	}
	root := result.Folders[0].ID
	if len(result.Folders) > 1 {
		items := make([]map[string]any, 0, len(result.Folders))
		for _, r := range result.Folders {
			items = append(items, map[string]any{"id": r.ID, "label": singleLine(r.Name, 120)})
		}
		pick, err := host.Window().ShowQuickPick(w, map[string]any{"title": tr(lang, "rootTitle"), "items": items})
		if err != nil {
			return err
		}
		var chosen struct {
			IDs []string `json:"selectedIDs"`
		}
		if string(pick) == "null" {
			return lithe.ErrCancelled
		}
		if json.Unmarshal(pick, &chosen) != nil || len(chosen.IDs) != 1 {
			return lithe.ErrCancelled
		}
		root = chosen.IDs[0]
	}
	path := ""
	for _, part := range []string{"leetcode", string(s.site), d.Slug} {
		if path != "" {
			path += "/"
		}
		path += part
		_, err = host.Workspace().Create(w, map[string]any{"resource": map[string]any{"rootID": root, "path": path}, "kind": "directory"})
		if err != nil && !hostCode(err, "alreadyExists") {
			return err
		}
	}
	resource := map[string]any{"rootID": root, "path": path + "/" + language.File}
	_, err = host.Workspace().Create(w, map[string]any{"resource": resource, "kind": "file", "text": text})
	if err != nil && !hostCode(err, "alreadyExists") {
		return err
	}
	_, err = host.Editor().Open(w, map[string]any{"resource": resource})
	return err
}
func judge(call *lithe.Context, s *state, lang string, submit bool) error {
	if s.pending != nil {
		return uiError(lang, "pendingExists")
	}
	host := call.Host()
	w := s.workspace
	if err := require(host, &w, lang, "editor/getActive", "documents/getSnapshot", "window/showWarning", "network/request", "credentials/get", "run/start"); err != nil {
		return err
	}
	client, err := clientFor(host, w, s.site)
	if err != nil {
		return err
	}
	if !client.Credential.Valid() {
		return uiError(lang, "signIn")
	}
	active, err := host.Editor().GetActive(w)
	if err != nil {
		return err
	}
	var doc struct {
		ID string `json:"documentID"`
	}
	if json.Unmarshal(active, &doc) != nil || doc.ID == "" {
		return uiError(lang, "notSolution")
	}
	read := func() (string, error) {
		raw, err := host.Documents().GetSnapshot(w, map[string]any{"documentID": doc.ID})
		if err != nil {
			return "", err
		}
		var snap struct {
			Text     string `json:"text"`
			Document struct {
				ReadOnly bool `json:"isReadOnly"`
			} `json:"document"`
		}
		if json.Unmarshal(raw, &snap) != nil {
			return "", uiError(lang, "requestFailed")
		}
		if snap.Document.ReadOnly {
			return "", uiError(lang, "readOnly")
		}
		return snap.Text, nil
	}
	text, err := read()
	if err != nil {
		return err
	}
	site, slug, language, err := leetcode.ParseSolution(text)
	if err != nil {
		return err
	}
	if site != s.site {
		return uiError(lang, "wrongSite")
	}
	key := "confirmRun"
	if submit {
		key = "confirmSubmit"
	}
	confirm, err := host.Window().ShowWarning(w, map[string]any{"text": fmt.Sprintf(tr(lang, key), s.site.Origin()), "actions": []string{tr(lang, "upload"), tr(lang, "cancel")}})
	if err != nil {
		return err
	}
	var answer struct {
		Index *int `json:"selectedIndex"`
	}
	if json.Unmarshal(confirm, &answer) != nil || answer.Index == nil || *answer.Index != 0 {
		return lithe.ErrCancelled
	}
	// Re-read after confirmation so the uploaded text is the current buffer.
	text, err = read()
	if err != nil {
		return err
	}
	nowSite, nowSlug, nowLanguage, err := leetcode.ParseSolution(text)
	if err != nil {
		return err
	}
	if nowSite != site || nowSlug != slug || nowLanguage != language {
		return uiError(lang, "notSolution")
	}
	d, err := client.Detail(call.Cancellation().Context(), slug)
	if err != nil {
		return err
	}
	job, err := client.Start(call.Cancellation().Context(), d, language, text, submit)
	if err != nil {
		return err
	}
	s.pending = &job
	// Preserve the server-issued ID before waiting. Cancellation cannot roll
	// back a remote submission and Check must never send the source again.
	if err := persist(s); err != nil {
		return uiError(lang, "stateFailed")
	}
	return poll(call, s, lang)
}
func poll(call *lithe.Context, s *state, lang string) error {
	host := call.Host()
	if err := require(host, &s.workspace, lang, "network/request", "credentials/get", "run/start"); err != nil {
		return err
	}
	client, err := clientFor(host, s.workspace, s.site)
	if err != nil {
		return err
	}
	// The task remains in state on cancellation/timeout; Check never re-uploads.
	for attempt := 0; attempt < 12; attempt++ {
		if err := call.CheckCancelled(); err != nil {
			return err
		}
		v, err := client.Check(call.Cancellation().Context(), *s.pending)
		if err != nil {
			return err
		}
		if v.State == "SUCCESS" {
			job := *s.pending
			s.lastReport = &judgeReport{Text: report(lang, job, v), Passed: verdictPassed(v), Language: lang, Job: job}
			s.pending = nil
			s.notice = ""
			if err := persist(s); err != nil {
				return uiError(lang, "stateFailed")
			}
			return showRunReport(host, s, lang)
		}
		call.Progress(tr(lang, "polling"))
		if call.Cancellation().WaitTimeout(1500 * time.Millisecond) {
			return lithe.ErrCancelled
		}
	}
	s.notice = tr(lang, "pending")
	return nil
}
