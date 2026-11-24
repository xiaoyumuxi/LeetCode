package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go/lithe"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
	"strings"
)

type hostTransport struct {
	host      *lithe.Host
	workspace lithe.WorkspaceContext
}

func (t hostTransport) Do(ctx context.Context, method, url string, headers map[string]string, body []byte) (leetcode.Response, error) {
	if err := ctx.Err(); err != nil {
		return leetcode.Response{}, err
	}
	params := map[string]any{"url": url, "method": method, "headers": headers, "timeoutMilliseconds": 15000, "maximumResponseBytes": 131072}
	if method != "GET" && method != "HEAD" {
		params["body"] = string(body)
	}
	raw, err := t.host.Network().Request(t.workspace, params)
	if err != nil {
		return leetcode.Response{}, err
	}
	var r struct {
		Status   int    `json:"status"`
		Body     string `json:"body"`
		Encoding string `json:"encoding"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return leetcode.Response{}, errors.New("invalid host network result")
	}
	data := []byte(r.Body)
	if r.Encoding == "base64" {
		data, err = base64.StdEncoding.DecodeString(r.Body)
	} else if r.Encoding != "utf8" {
		err = errors.New("invalid host network encoding")
	}
	if err != nil {
		return leetcode.Response{}, err
	}
	return leetcode.Response{Status: r.Status, Body: data}, nil
}
func require(host *lithe.Host, workspace *lithe.WorkspaceContext, language string, methods ...string) error {
	if workspace == nil {
		return uiError(language, "noWorkspace")
	}
	for _, method := range methods {
		ok, err := host.Supports("lithe/host/" + method)
		if err != nil {
			return err
		}
		if !ok {
			return lithe.NewPluginError("leetcode.missingAPI", fmt.Sprintf(tr(language, "missingAPI"), method))
		}
	}
	return nil
}
func uiError(language, key string) error {
	return lithe.NewPluginError("leetcode."+key, tr(language, key))
}
func safeError(language string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, lithe.ErrCancelled) || errors.Is(err, context.Canceled) {
		return lithe.ErrCancelled
	}
	var p *lithe.PluginError
	if errors.As(err, &p) {
		return err
	}
	var sdk *lithe.SdkError
	if errors.As(err, &sdk) {
		switch sdk.Code() {
		case "sdk.call.cancelled", "plugin.host.cancelled":
			return lithe.ErrCancelled
		case "plugin.host.limitExceeded":
			return uiError(language, "tooLarge")
		case "plugin.host.contextExpired":
			return uiError(language, "noWorkspace")
		}
	}
	return uiError(language, leetcode.Category(err))
}
func credentialKey(site leetcode.Site) string { return "leetcode." + string(site) + ".account" }
func clientFor(host *lithe.Host, workspace lithe.WorkspaceContext, site leetcode.Site) (leetcode.Client, error) {
	secret, present, err := host.Credentials().Get(credentialKey(site))
	if err != nil {
		return leetcode.Client{}, err
	}
	var credential leetcode.Credential
	if present && (json.Unmarshal([]byte(secret), &credential) != nil || !credential.Valid()) {
		return leetcode.Client{}, &leetcode.Error{Kind: "invalidCredential"}
	}
	return leetcode.Client{Site: site, Transport: hostTransport{host, workspace}, Credential: credential}, nil
}
func hostCode(err error, code string) bool {
	var sdk *lithe.SdkError
	return errors.As(err, &sdk) && sdk.Code() == "plugin.host."+code
}
func singleLine(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, s)
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit-1]) + "…"
	}
	return s
}
