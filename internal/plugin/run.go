package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go/lithe"
	"github.com/xiaoyumuxi/LeetCode/internal/leetcode"
)

const reportTextLimit = 128 * 1024

// judgeReport is the real remote verdict passed to the host-owned console
// process. It contains no credentials or source buffer and is never persisted
// in the plugin package or its state directory.
type judgeReport struct {
	Text     string       `json:"text"`
	Passed   bool         `json:"passed"`
	Language string       `json:"language"`
	Job      leetcode.Job `json:"job"`
}

func verdictPassed(v leetcode.Verdict) bool {
	if v.Correct != nil {
		return *v.Correct
	}
	return v.StatusCode == 10
}

func showRunReport(host *lithe.Host, s *state, lang string) error {
	if err := require(host, &s.workspace, lang, "run/start"); err != nil {
		return err
	}
	r := s.lastReport
	if r == nil {
		return uiError(lang, "noResult")
	}
	if len(r.Text) > reportTextLimit {
		return uiError(lang, "tooLarge")
	}
	contents, err := json.Marshal(r)
	if err != nil {
		return uiError(lang, "requestFailed")
	}
	executable := "bin/backend"
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	kind := "reportRun"
	if r.Job.Submit {
		kind = "reportSubmit"
	}
	// run/start is the public Run entry point. The same packaged executable
	// has a console mode that prints only this completed judge report. The
	// host creates and removes the temporary input and owns the Run entry;
	// no shell, Output channel, or additional backend runtime is needed.
	_, err = host.Request("lithe/host/run/start", map[string]any{
		"context": s.workspace,
		"label":   fmt.Sprintf("LeetCode · %s #%s", tr(lang, kind), r.Job.ID),
		"process": map[string]any{
			"executable":       map[string]string{"kind": "packaged", "path": executable},
			"arguments":        []string{"--judge-report", "${temporaryFile:judge-report}"},
			"workingDirectory": ".",
			"environment":      map[string]string{},
			"outputEncoding":   "utf8",
		},
		"temporaryFiles": []map[string]string{{"name": "judge-report", "contents": string(contents)}},
	})
	if err != nil {
		var sdk *lithe.SdkError
		if errors.Is(err, lithe.ErrCancelled) || errors.Is(err, context.Canceled) || (errors.As(err, &sdk) && (sdk.Code() == "sdk.call.cancelled" || sdk.Code() == "plugin.host.cancelled")) {
			return err
		}
		// Retain the completed report even when presentation fails. The next
		// state read exposes Result again without repeating the remote upload.
		return uiError(lang, "resultDisplayFailed")
	}
	return nil
}

// PrintJudgeReport is the console mode of the packaged executable. Its input
// path is explicitly supplied by run/start, never derived from the bundle or
// plugin directory. A failed remote verdict produces a nonzero Run exit code.
func PrintJudgeReport(path string, stdout, stderr io.Writer) int {
	var report judgeReport
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		decoder := json.NewDecoder(io.LimitReader(f, 1024*1024+1))
		err = decoder.Decode(&report)
		if err == nil {
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				err = fmt.Errorf("invalid report input")
			}
		}
	}
	if err != nil || report.Text == "" || len(report.Text) > reportTextLimit {
		fmt.Fprintln(stderr, tr(report.Language, "reportReadFailed"))
		return 1
	}
	if _, err := io.WriteString(stdout, report.Text+"\n"); err != nil {
		return 1
	}
	if !report.Passed {
		return 1
	}
	return 0
}
