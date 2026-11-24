# Lithe LeetCode / 力扣插件

个人维护仓库：[xiaoyumuxi/LeetCode](https://github.com/xiaoyumuxi/LeetCode)。维护入口见 [维护说明](docs/maintenance.md)，问题可在 [Issues](https://github.com/xiaoyumuxi/LeetCode/issues) 提交。

[`0.0.1` 预发布](https://github.com/xiaoyumuxi/LeetCode/releases/tag/v0.0.1)。插件代码与测试独立维护；不需要在宿主添加力扣特判。**还未发布到市场，也不承诺兼容当前稳定版宿主。** 所依赖的契约、SDK、宿主提交见 [sdk.lock.json](sdk.lock.json)。本轮按用户要求只构建发布包，未运行测试或桌面验收；Release 提供 macOS Apple Silicon 和 Intel 安装包。宿主通用修复见 [#2383](https://github.com/1lck/Lithe-IDEA/pull/2383)。

实现了中英文站公开题库、关键词/难度/标签/状态筛选、每日一题、文本题面预览，以及从真实题目模板生成解题文件。中文站题解与热门顶层评论在编辑区只读标签页打开；在线示例运行、提交和结果查询已接线，完成后在宿主「运行」工具窗口显示报告，仍需使用真实账号验收。详见 [功能矩阵](docs/features.md)、[问题清单](docs/issues.md) 和 [验证记录](docs/verification.md)。

## 使用

1. 在 Lithe 中打开一个本地刷题目录，从磁盘安装本仓库构建的 zip，审查后端程序和权限。
2. 从工具窗口或命令面板打开 `LeetCode`，选择中文站或国际站，点击“加载题库”。首次打开不会自动联网。
3. 搜索与筛选，单击题目打开题面；方向键选择、回车查看。滚动到底部自动获取下一批题目，每批 40 题，最多保留 400 题；更多题目请缩小搜索范围。
4. 选择解题语言后点击“打开解题文件”。文件位于工作区的 `leetcode/<站点>/<题目>/`；已有文件直接打开，不覆盖。
5. 点击「在浏览器登录」打开对应站点的官方登录页。网页登录后，点「关联已登录账号」，手动输入同一站点的 `LEETCODE_SESSION` 和 `csrftoken` Cookie 值，验证账号成功后才存入宿主系统凭据库。当前尚不能自动把浏览器会话回传 Lithe；打开网页不会把插件标记为已登录。两站凭据隔离，不读取浏览器数据库。
6. 保留文件首行标识，打开该文件后点击“在线跑示例”或“提交当前文件”。确认上传后使用编辑器当前缓冲区，未保存的修改也会上传。运行语言取自文件标识。
7. 取消只停止本地等待，远端任务可能继续；点击“查询判题结果”继续查询，**不要因为等待超时重新提交**。服务端已返回的任务 ID 会保存在宿主授予的工作区状态目录。
8. 完成后自动打开「运行」工具窗口，显示判题状态、耗时、内存、输出及站点返回的失败详情。再次查看使用「在‘运行’中查看结果」，不会重新上传代码。报告只在当前后端会话中保留；「运行」里的停止按钮只管理报告显示进程，远端等待使用插件自己的取消按钮。
9. 中文站选题后点击「题解」或「讨论」，单击条目或用方向键和回车打开正文。「返回题库」回到题目列表；每页 10 条，最多保留 100 条，讨论暂不含回复。国际站社区仍使用原网站。

首版不支持会员题、本地判题/调试、自定义测试用例、历史提交、复习计划或模拟面试。题面只保留文本和代码，图片与公式请打开原网站查看。

## 构建与检查

需要 Go `1.27.4`、Python 3、Node.js（仅开发检查）：

```sh
go mod download
go test -race -timeout 45s ./...
go vet ./...
node --check ui/main.js
go run ./scripts/build/translations.go --check
python3 scripts/build/package.py
```

修改中英文文案后执行 `go run ./scripts/build/translations.go`。唯一词条来源为 `internal/plugin/i18n.go`，网页使用生成的 `ui/messages.js`，不请求远程语言包。清单中的 `LeetCode` 是两种界面语言共用的产品名；当前契约不能翻译工具窗口/命令标题，因此不伪造 `l10n` 字段。

产物在 `dist/`，解包目录在 `.artifacts/package/<系统>-<架构>/`。不需要 Node 或 Java 后端，不在运行时下载依赖。资源生命周期与缓存复用规则见 [构建资源](docs/resources.md)。

## 权限

| 权限 | 用途 |
| --- | --- |
| `backend.launch` | 按需启动打包的 Go 后端 |
| `ui.contribute`, `ui.web` | 工具窗口、命令入口、插件网页 |
| `workspace.metadata` | 获取绑定工作区、根目录与宿主授予的状态目录 |
| `workspace.files.write`, `editor.open` | 创建目录/解题文件并打开；不覆盖已存在文件 |
| `documents.read` | 读取当前解题文件的实时缓冲区 |
| `credentials.store` | 保存、读取和删除插件私有凭据 |
| `network.request` | 请求固定的 `https://leetcode.cn` 或 `https://leetcode.com`；不追随重定向 |
| `preview.open`, `browser.open` | 只读题面、在系统浏览器打开题目 |
| `window.input`, `window.messages` | 密码输入、选工作区根、上传前确认 |
| `run.start` | 通过公共 Run API 在「运行」工具窗口显示真实判题报告；复用打包后端的报告模式，无 shell 或额外运行时 |

API 等级为 `0`，只声明 `backend:0` 和 `ui:0`。所有宿主方法先做能力发现，缺能力显示具体方法；不会偷偷改走本地文件、网络或私有宿主接口。

从旧版升级需完整安装同一份插件包，再重新启用插件、重开面板；若提示页面与后端版本不一致，请重启 Lithe。图标使用固定版本的 Simple Icons LeetCode SVG，来源和许可证见 [第三方资源](THIRD_PARTY_NOTICES.md)。
