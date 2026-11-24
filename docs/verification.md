# 验证记录与日志接线

## 版本与边界

历史验证平台为 macOS arm64，日期为 2026-11-24。Core、契约与官方 Go SDK 固定在 `75e148091439248e5927c58790019b26de02ca87`，Go 模块版本 `v0.0.0-20261123221629-75e148091439`；当时参考桌面提交 `40cf49cc57507c232c30bb7a56bdca62703245f0`。最新参考哈希见 `sdk.lock.json`，更新参考提交不代表已经运行验证。

用户随后明确要求「不用进行验证，直接写代码即可，实际验证我自己来」。该要求之后没有执行测试、静态检查、浏览器或原生窗口验收；只做实现、格式化与词条生成。随后用户明确要求发布 `0.0.1`，因此另外构建了该版本的 macOS arm64/x64 安装包，构建成功不等于工作流验收通过。最后的 Run 报告、浏览器登录入口、社区交互和页面/后端版本检查均待用户验收。旧版本 `dist/` 产物不能作为本次交付包；本次只发布新构建的 `0.0.1`。

公开工作流通过实际 Core 静态库启动打包后的 Go 后端，使用真实站点网络；工作区、输出和预览等平台适配器由插件仓库中的验收脚本提供。它验证协议和业务流程，不替代真实 Lithe 桌面验收。离线边界测试明确使用固定输入，不提供登录或判题成功证据。

## 历史检查入口

在插件仓库执行：

```sh
go mod verify
go test -race -json -count=1 -timeout 45s ./...
go vet ./...
node --check ui/main.js
go run ./scripts/build/translations.go --check
python3 scripts/build/package.py
python3 scripts/verify/core-workflow.py --core-dir <固定提交的本机-Core-debug-产物目录>
```

每个 Go 测试的耗时保留在 `.artifacts/test-stability/` 的 JSON、HTML 和 JUnit 报告。测试覆盖凭据头注入、代码首行标识、活动题面清理、Start/Check 不重复上传、取消与 403 不重试、中文站字段映射；不扩展到无关宿主测试。

Core 工作流覆盖：清单与安装权限审查、后端激活、两站真实搜索、题面、每日一题、从真实 Python 模板创建文件、再次打开不覆盖、无凭据提交被拒绝、站点与语言在后端重启后恢复、停止后端和插件包只读验证。证据分别保存于每轮 `.artifacts/verify/<本轮ID>/install-review.json`、`evidence.json`、`core-events.json` 与 `statement.html`。

网页验收入口：

```sh
python3 scripts/verify/web-workflow.py --core-dir <同一Core目录> --duration 600
```

该入口只监听 `127.0.0.1`，打印随机路径，通过真实 Core 调用后端；没有账号凭据适配器。`/<随机路径>/zh-CN/light/ui/index.html` 可切换到 `zh-CN/dark`、`en-US/light`、`en-US/dark`。测试页只插入桥接传输，复用包内网页、词条和样式；结果属于浏览器验证，不属于 Lithe WebView、宿主主题注入、密码框或原生日志窗口的验证。服务有明确运行期限，退出时停止后端并校验插件包不变。

## 日志接线

| 来源 | 角色与通道 | 记录规则 |
| --- | --- | --- |
| 打包的 Go 后端 | `backend`，`plugins/io.github.lithe-idea.leetcode/host` | SDK 协商 `litheLogging`，自动上报函数唯一终态；插件不重复记日志 |
| 网页、工具窗口和命令声明 | 宿主安装、激活与加载边界 | 不启动另一种程序，不另建日志出口 |
| 报告模式 `bin/backend --judge-report <宿主临时文件>` | 宿主管理的 Run 进程，复用同一可执行文件 | stdout/stderr 只进入 Run 控制台，退出状态反映结果；不启动 SDK 协议、不额外记业务日志 |
| 题面、用户代码、判题报告 | 用户内容 | 仅进入对应预览、工作区或 Run 控制台，不进入诊断日志 |

| 操作 | 成功 | 失败与取消 |
| --- | --- | --- |
| 打开浏览器登录页、手动关联、退出、创建/打开解题文件、运行、提交 | SDK 自动 INFO `plugins.operation.finished` | 最终失败由 SDK 自动 ERROR `plugins.operation.failed`；取消无 ERROR/WARN |
| 打开页面、加载/搜索/筛选/分页、每日题、选题、题面与社区预览、账户查询、判题查询、再次查看结果、偏好变化 | `SuppressSuccessDiagnostics()` 关闭成功记录 | 失败照常由 SDK 记录唯一终态 |
| 自动重试、降级、自动恢复 | 不适用：首版没有这些路径 | 不会把失败当成功或静默回退 |

受控业务错误标识为 `leetcode.` 加以下类别：`busy`、`stateFailed`、`noWorkspace`、`missingAPI`、`invalidRequest`、`requestFailed`、`schemaChanged`、`forbidden`、`rateLimited`、`http`、`signIn`、`invalidCredential`、`notFound`、`paid`、`noSamples`、`noSnippet`、`notSolution`、`wrongSite`、`readOnly`、`tooLarge`、`pendingExists`、`noPending`、`selectFirst`、`communitySite`、`noResult`、`resultDisplayFailed`。Core 与 SDK 的传输、生命周期错误保留其受控类别；Run 展示失败使用独立业务类别，底层原因由宿主的关联轨迹保留。所有类别都不携带用户输入、网络响应、秘密或路径。

公开 Core 工作流的日志应恰好有两条 `create` 成功 INFO 和一条无凭据 `submit` 失败 ERROR（`leetcode.signIn`），与 evidence 的 operation ID 对应；高频成功不产生 INFO，`dropped` 为 0。没有自建日志文件、stderr 业务日志、原生日志目录或上游服务进程，因此原生日志文件轮转不适用。

## 尚未验收

- 用户截图已确认旧 0.1.0 的从磁盘安装、工具窗口、题库加载和解题文件创建；最后 0.0.1 的重新安装、原生输入框、编辑器缓冲区上传、Run 与日志窗口仍 pending。
- 真实账号登录、样例运行、提交、判题结果、远端任务取消与恢复。
- 真实窗口的性能、内存和关闭面板释放行为；Viva 无头截图与 PR 上可访问的图片证据。
- Windows：按用户要求本轮不执行。

用户正在使用的应用实例保持运行，没有为本次修改另启或重启应用。此前 Core 与网页验收脚本在 `finally` 中停止各自后端；这不能替代最后修改的验收。没有执行或计划执行 Windows 本地验收。
