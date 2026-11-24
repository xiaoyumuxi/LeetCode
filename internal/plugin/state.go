package plugin

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go/lithe"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
)

// Only preferences and an opaque remote job ID live here. Credentials remain
// in the host store; source and judge output are never persisted by the plugin.
type savedState struct {
	Site     leetcode.Site `json:"site"`
	Language string        `json:"language"`
	Pending  *leetcode.Job `json:"pending,omitempty"`
}

func restore(call *lithe.Context, s *state, lang string) error {
	if err := require(call.Host(), &s.workspace, lang, "workspace/getContext", "credentials/get"); err != nil {
		return err
	}
	raw, err := call.Host().Workspace().GetContext(s.workspace)
	if err != nil {
		return err
	}
	var info struct {
		Directory string `json:"stateDirectory"`
	}
	if json.Unmarshal(raw, &info) != nil || !filepath.IsAbs(info.Directory) {
		return uiError(lang, "stateFailed")
	}
	// This path is explicitly granted by the host, never derived from the
	// executable, package directory, app bundle, or a workspace file.
	s.directory = info.Directory
	client, err := clientFor(call.Host(), s.workspace, s.site)
	if err != nil {
		return err
	}
	s.credentialStored = client.Credential.Valid()
	f, err := os.Open(filepath.Join(s.directory, "practice.json"))
	if os.IsNotExist(err) {
		s.initialized = true
		return nil
	}
	if err != nil {
		return uiError(lang, "stateFailed")
	}
	defer f.Close()
	var saved savedState
	if json.NewDecoder(io.LimitReader(f, 8193)).Decode(&saved) != nil || !saved.Site.Valid() || !leetcode.SupportedLanguage(saved.Language) || (saved.Pending != nil && !leetcode.ValidJobID(saved.Pending.ID)) {
		return uiError(lang, "stateFailed")
	}
	s.site, s.language, s.pending = saved.Site, saved.Language, saved.Pending
	client, err = clientFor(call.Host(), s.workspace, s.site)
	if err != nil {
		return err
	}
	s.credentialStored = client.Credential.Valid()
	encoded, _ := json.Marshal(saved)
	s.saved = string(encoded)
	s.initialized = true
	return nil
}

func persist(s *state) error {
	data, err := json.Marshal(savedState{s.site, s.language, s.pending})
	if err != nil {
		return err
	}
	if s.saved == string(data) {
		return nil
	}
	if err = os.MkdirAll(s.directory, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.directory, ".practice-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(s.directory, "practice.json")); err != nil {
		return err
	}
	s.saved = string(data)
	return nil
}
