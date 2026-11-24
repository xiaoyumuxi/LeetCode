package leetcode

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type transportFunc func(context.Context, string, string, map[string]string, []byte) (Response, error)

func (f transportFunc) Do(ctx context.Context, method, target string, headers map[string]string, body []byte) (Response, error) {
	return f(ctx, method, target, headers, body)
}

// These are offline boundary tests, not evidence of a real login or verdict.
func TestCredentialsRejectHeaderInjection(t *testing.T) {
	for _, token := range []string{"", "value\r\nInjected: yes", "value;other=x", "name=value,other", "with space", strings.Repeat("a", 6001)} {
		if (Credential{Session: token, CSRF: "protocol-fixture"}).Valid() {
			t.Fatal("invalid credential accepted")
		}
	}
}

func TestSolutionIdentityAndSafeStatement(t *testing.T) {
	d := Detail{Question: Question{Slug: "protocol-fixture"}, Snippets: []Snippet{{Language: "python3", Code: "pass"}}}
	text, err := Solution(d, CN, "python3")
	if err != nil {
		t.Fatal(err)
	}
	site, slug, language, err := ParseSolution(text)
	if err != nil || site != CN || slug != d.Slug || language != "python3" {
		t.Fatal("solution identity was lost")
	}
	if _, _, _, err = ParseSolution("# unrelated\n" + text); err == nil {
		t.Fatal("a displaced marker was accepted")
	}
	got := PlainStatement(`<p>A &lt; B</p><script>active-script</script><iframe>frame-body</iframe><pre>  code\nnext</pre><img src="https://invalid.test/image">`)
	if strings.Contains(got, "active-script") || strings.Contains(got, "frame-body") || strings.Contains(got, "invalid.test") || !strings.Contains(got, "A < B") {
		t.Fatalf("unsafe statement: %q", got)
	}
}

func TestJudgeIdentifierAndCheckNeverUploads(t *testing.T) {
	var methods []string
	client := Client{Site: COM, Credential: Credential{Session: "protocol-fixture", CSRF: "protocol-fixture"}, Transport: transportFunc(func(_ context.Context, method, target string, _ map[string]string, body []byte) (Response, error) {
		methods = append(methods, method)
		if len(methods) == 1 {
			var payload map[string]any
			if json.Unmarshal(body, &payload) != nil || payload["data_input"] != "1" {
				t.Fatal("sample input missing")
			}
			return Response{200, []byte(`{"interpret_id":"runcode_123_abc:fixture"}`)}, nil
		}
		if method != "GET" || !strings.HasSuffix(target, "/runcode_123_abc:fixture/check/") {
			t.Fatal("Check tried to upload or changed the job")
		}
		return Response{200, []byte(`{"state":"SUCCESS","correct_answer":true}`)}, nil
	})}
	job, err := client.Start(t.Context(), Detail{Question: Question{ID: "1", Slug: "protocol-fixture"}, SampleTestCase: "1"}, "python3", "pass", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Check(t.Context(), job)
	if err != nil || result.Correct == nil || !*result.Correct {
		t.Fatal("verdict decode failed")
	}
	if len(methods) != 2 || methods[0] != "POST" {
		t.Fatal("unexpected request sequence")
	}
	for _, bad := range []string{"../other", "id/path", "with space", ""} {
		if ValidJobID(bad) {
			t.Fatal("unsafe job ID accepted")
		}
	}
}

func TestCancellationAndForbiddenAreNotRetried(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		requests := 0
		client := Client{Site: CN, Transport: transportFunc(func(context.Context, string, string, map[string]string, []byte) (Response, error) {
			requests++
			if cancelled {
				return Response{}, context.Canceled
			}
			return Response{403, []byte("upstream-private-content")}, nil
		})}
		_, err := client.Daily(t.Context())
		if err == nil || requests != 1 || strings.Contains(err.Error(), "upstream-private-content") {
			t.Fatal("failure leaked content or retried")
		}
		if cancelled && !errors.Is(err, context.Canceled) {
			t.Fatal("cancellation was not preserved")
		}
	}
}

func TestChinaListUsesCanonicalFieldNames(t *testing.T) {
	client := Client{Site: CN, Transport: transportFunc(func(context.Context, string, string, map[string]string, []byte) (Response, error) {
		return Response{200, []byte(`{"data":{"problemsetQuestionList":{"total":1,"questions":[{"frontendQuestionId":"1","title":"Protocol fixture","titleCn":"协议边界用例","titleSlug":"protocol-fixture","difficulty":"EASY","paidOnly":true}]}}}`)}, nil
	})}
	page, err := client.List(t.Context(), Search{Limit: 1})
	if err != nil || len(page.Questions) != 1 {
		t.Fatalf("canonical response rejected: %v", err)
	}
	q := page.Questions[0]
	if q.FrontendID != "1" || q.Name(true) != "协议边界用例" || !q.Paid {
		t.Fatal("canonical fields lost")
	}
}
