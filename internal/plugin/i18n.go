package plugin

import "strings"

func chinese(language string) bool { return strings.HasPrefix(strings.ToLower(language), "zh") }
func tr(language, key string) string {
	pair, ok := messages[key]
	if !ok {
		return key
	}
	if chinese(language) {
		return pair[0]
	}
	return pair[1]
}

var messages = map[string][2]string{
	"connect":             {"关联已登录账号", "Connect signed-in account"},
	"browserLoginOpened":  {"已打开浏览器登录页。网页登录后，请点“关联已登录账号”。当前站点还不能自动向 Lithe 回传登录状态，关联仍需手动输入会话凭据。", "The browser sign-in page is open. After signing in, choose Connect signed-in account. The site cannot return sign-in state to Lithe yet; connecting still requires entering session credentials."},
	"showResult":          {"在“运行”中查看结果", "Show result in Run"},
	"noResult":            {"当前没有已完成的判题结果。", "No completed judge result is available."},
	"reportReadFailed":    {"无法读取判题报告，请回到插件重新打开结果。", "Cannot read the judge report. Reopen the result from the plugin."},
	"resultDisplayFailed": {"判题已完成，但无法打开“运行”面板。请点击“同步当前状态”，再点“在‘运行’中查看结果”；无需重新上传代码。", "Judging is complete, but the Run panel could not be opened. Sync the current state, then choose Show result in Run. Do not upload the code again."},
	"versionMismatch":     {"页面与后端版本不一致。请完整安装同一版本的插件，然后重新启用插件并打开面板；仍有此提示时请重启 Lithe。", "The page and backend versions do not match. Install the complete plugin at one version, then re-enable it and reopen the panel. Restart Lithe if this message remains."},
	"communityCount":      {"已加载 %d / %d 条", "Loaded %d of %d entries"},
	"solutions":           {"题解", "Solutions"},
	"discussion":          {"讨论", "Discussion"},
	"backProblems":        {"返回题库", "Back to problems"},
	"communityInfo":       {"选择题解或评论后，正文会在编辑区只读标签页打开。目前支持中文站；讨论显示按热度排列的顶层评论，不含回复。", "Select a solution or comment to open it in a read-only editor tab. Currently available on the China site. Discussion shows top-level comments by popularity, without replies."},
	"communitySite":       {"国际站题解与讨论尚未接入，请使用“在浏览器打开”。中文站可在 IDE 内查看。", "Global-site solutions and discussion are not integrated yet. Use Open in browser. The China site supports reading them inside the IDE."},
	"communityEmpty":      {"当前没有可显示的题解或评论。", "No solutions or comments are available."},
	"communityLimit":      {"本轮已达到社区内容加载上限，请返回题库后重新打开。", "The community content limit has been reached. Return to problems and reopen the list."},
	"busy":                {"另一个操作正在执行，请等待或取消后重试。", "Another operation is running. Wait or cancel it before retrying."},
	"stateFailed":         {"无法读写插件状态。请检查宿主状态目录的权限；不要重新提交已上传的代码。", "Cannot read or save plugin state. Check host state-directory permissions; do not resubmit uploaded code."},
	"loading":             {"正在处理…", "Working…"},
	"unavailable":         {"请在 Lithe 中打开此插件。", "Open this plugin inside Lithe."},
	"nothingFound":        {"没有匹配的题目。请调整搜索条件。", "No matching problems. Adjust the search filters."},
	"retry":               {"重试", "Retry"},
	"syncState":           {"同步当前状态", "Sync current state"},
	"verdictInfo":         {"运行和提交会上传当前编辑器缓冲区，使用文件首行记录的题目和语言。完成后在“运行”面板显示结果。取消只停止等待，远端任务可能继续。", "Run and submit upload the active editor buffer, using the problem and language in its first-line marker. Completed results appear in Run. Cancel stops waiting; the remote job may continue."},
	"savedCredential":     {"已保存凭据，尚未验证账号", "Stored credentials; account not verified"},
	"account":             {"验证已存凭据", "Verify stored credentials"},
	"site":                {"站点", "Site"}, "language": {"解题语言", "Solution language"}, "search": {"搜索", "Search"}, "query": {"题号或关键词", "Number or keywords"}, "difficulty": {"难度", "Difficulty"}, "all": {"全部", "All"}, "EASY": {"简单", "Easy"}, "MEDIUM": {"中等", "Medium"}, "HARD": {"困难", "Hard"}, "tag": {"标签标识", "Tag slug"}, "tagInfo": {"填写站点的英文标签标识，例如 array、dynamic-programming。留空表示全部。", "Enter a site tag slug, such as array or dynamic-programming. Leave empty for all tags."}, "status": {"做题状态", "Problem status"}, "AC": {"已通过", "Accepted"}, "TRIED": {"尝试过", "Attempted"}, "NOT_STARTED": {"未开始", "Not started"}, "refresh": {"加载题库", "Load problems"}, "daily": {"每日一题", "Daily challenge"}, "login": {"在浏览器登录", "Sign in with browser"}, "logout": {"退出登录", "Sign out"}, "preview": {"查看题面", "Statement"}, "create": {"打开解题文件", "Open solution"}, "website": {"在浏览器打开", "Open in browser"}, "run": {"在线跑示例", "Run samples online"}, "submit": {"提交当前文件", "Submit active solution"}, "check": {"查询判题结果", "Check pending verdict"}, "problems": {"题库", "Problems"}, "empty": {"点击“加载题库”，或输入关键词后回车。", "Load problems, or enter keywords and press Return."}, "selected": {"当前题目", "Selected problem"}, "none": {"请选择题目", "Select a problem"}, "count": {"已加载 %d / %d 题", "Loaded %d of %d problems"}, "signedOut": {"未登录：可浏览公开题目", "Signed out: public problems available"}, "signedIn": {"已登录：%s", "Signed in: %s"}, "paidLabel": {"会员题", "Premium"}, "more": {"继续加载", "Load more"}, "sessionTitle": {"LeetCode 会话凭据", "LeetCode session credential"}, "sessionPrompt": {"在对应站点登录后，从浏览器开发者工具复制 LEETCODE_SESSION 的值。只保存到系统凭据库。", "After signing in to this site, copy LEETCODE_SESSION from browser developer tools. Stored only in the system credential store."}, "csrfTitle": {"CSRF 校验值", "CSRF token"}, "csrfPrompt": {"从同一个站点复制 csrftoken 的值。不会读取其他浏览器数据。", "Copy csrftoken from the same site. No other browser data is read."}, "confirmTitle": {"确认上传代码", "Confirm code upload"}, "confirmRun": {"运行会把当前编辑器缓冲区中的完整代码上传到 %s，使用该题官方示例。", "Running uploads the full live editor buffer to %s using the problem's example input."}, "confirmSubmit": {"提交会把当前编辑器缓冲区中的完整代码上传到 %s，并创建一次真实提交。", "Submitting uploads the full live editor buffer to %s and creates a real submission."}, "upload": {"确认上传", "Upload"}, "cancel": {"取消", "Cancel"}, "polling": {"等待远端判题；取消后远端任务可能仍会继续。", "Waiting for the remote judge; cancelling may leave the remote job running."}, "pending": {"判题尚未完成，请点击“查询判题结果”，不会重复上传代码。", "Verdict pending. Check the pending verdict without uploading the code again."}, "result": {"判题结果", "Judge result"}, "passed": {"通过", "Passed"}, "failed": {"未通过", "Failed"}, "reportRun": {"示例运行", "Sample run"}, "reportSubmit": {"代码提交", "Submission"}, "runtime": {"耗时", "Runtime"}, "memory": {"内存", "Memory"}, "tests": {"测试用例", "Test cases"}, "output": {"程序输出", "Program output"}, "expected": {"预期输出", "Expected output"}, "stdout": {"标准输出", "Standard output"}, "input": {"未通过的用例", "Failing input"}, "compiler": {"编译错误", "Compile error"}, "runtimeError": {"运行时错误", "Runtime error"}, "noWorkspace": {"请先打开本地刷题工程。", "Open a local practice workspace first."}, "missingAPI": {"当前宿主缺少本插件所需的公共 API：%s", "The host is missing a required public API: %s"}, "requestFailed": {"请求失败。请检查网络、权限与宿主日志后重试。", "Request failed. Check network access, permissions and host logs."}, "schemaChanged": {"站点接口返回了不兼容的数据，需要更新插件。", "The site returned incompatible data. The plugin needs an update."}, "forbidden": {"站点拒绝请求（403）。请先在浏览器确认账号或风控验证，不会自动重试。", "The site refused the request (403). Check your account or browser challenge. No automatic retry was sent."}, "rateLimited": {"站点请求过于频繁，请稍后重试。", "The site rate limited requests. Try again later."}, "http": {"站点返回非成功 HTTP 状态，请稍后重试。", "The site returned an unsuccessful HTTP status. Try again later."}, "signIn": {"请先登录；已有凭据可能过期。", "Sign in first; stored credentials may have expired."}, "invalidCredential": {"凭据格式无效，请复制 Cookie 的值，不要包含名称或分号。", "Invalid credential. Copy only the cookie value, without its name or semicolons."}, "notFound": {"未找到这道题。", "Problem not found."}, "paid": {"第一版只支持公开题目。请在站点查看会员题。", "This version supports public problems. Open premium problems on the site."}, "invalidRequest": {"输入无效，请检查搜索条件或解题文件。", "Invalid input. Check the search filters or solution file."}, "noSamples": {"站点没有提供这道题的示例输入。", "The site did not provide sample input for this problem."}, "noSnippet": {"该题没有所选语言的代码模板，请切换语言。", "No template exists for the selected language. Choose another language."}, "notSolution": {"请先打开本插件生成的解题文件，并保留首行标识。", "Open a solution generated by this plugin and keep its first-line marker."}, "wrongSite": {"当前文件属于另一个站点，请先切换站点。", "The active solution belongs to another site. Switch the site first."}, "readOnly": {"当前文档只读，不能作为解题文件上传。", "The active document is read-only and cannot be uploaded as a solution."}, "tooLarge": {"代码、题面或结果超出当前 API 的大小限制。", "The code, statement or result exceeds the current API size limit."}, "pendingExists": {"已有未完成的远端任务，请先查询它的结果。", "A remote job is pending. Check its verdict first."}, "noPending": {"当前没有等待中的判题任务。", "No judge job is pending."}, "selectFirst": {"请先选择一道题。", "Select a problem first."}, "loginInfo": {"先在系统浏览器登录官网，再点“关联已登录账号”。力扣尚未提供 Lithe 授权回调，仍需手动输入会话凭据；验证账号成功后才保存，两站互相隔离。", "Sign in on the official site in your system browser, then connect the account. LeetCode has no Lithe authorization callback yet, so session credentials must still be entered manually. They are stored only after account verification, separately for each site."}, "createInfo": {"在当前工程的 leetcode/<站点>/<题目>/ 下创建代码文件；已有文件直接打开，不会覆盖。", "Create code under leetcode/<site>/<problem>/ in the selected workspace root. Existing files are opened without overwriting."}, "previewInfo": {"题面显示文本与代码；公式和图片请在浏览器查看。", "Statements show text and code. Open the browser for formulas and images."}, "languageInfo": {"选择站点支持的解题语言。在线运行使用文件首行记录的语言。", "Choose a solution language supported by the site. Online execution uses the language in the file's first line."}, "loadLimit": {"本轮已加载 400 题，请使用搜索或筛选缩小范围。", "400 problems loaded. Narrow the results with search or filters."}, "rootTitle": {"选择刷题目录", "Choose the workspace root"}, "SITE_CN": {"力扣中文站", "LeetCode China"}, "SITE_COM": {"LeetCode 国际站", "LeetCode Global"},
}

// Translations returns the same catalog used by the backend for the packaged UI.
func Translations() map[string]map[string]string {
	result := map[string]map[string]string{"zh": {}, "en": {}}
	for key, value := range messages {
		result["zh"][key], result["en"][key] = value[0], value[1]
	}
	return result
}
