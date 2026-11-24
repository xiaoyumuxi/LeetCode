# 功能矩阵

最近验证日期：2026-11-24。`implemented` 表示已有实现，`verified` 仅用于实际跑过的范围。macOS 的 Core 流程由插件仓库自己的适配器驱动，不等于 Lithe 原生窗口验收。Windows 按用户要求不做本地验收，全部保持 `pending`。

用户最新要求不再执行验证。以下历史 `verified` 对应较早代码；0.0.1 最后新增的 Run 展示、浏览器入口、版本检查与社区交互修正均为 `implemented / pending`，按后续发布请求构建了 `0.0.1` 安装包，仍未运行最终验收。

| 能力 | 实现与证据 | macOS 验证 | Windows 验证 |
| --- | --- | --- | --- |
| 清单校验、安装权限审查、按需启动 Go 后端 | `plugin.json`、`scripts/verify/core-workflow.py` | 真实 Core 已验证；用户已安装 0.1.0，最后 0.0.1 pending | pending |
| 中文站与国际站题库、关键词搜索 | `internal/leetcode/client.go`、Core 公开网络工作流 | verified | pending |
| 难度、标签、做题状态筛选 | 同上；状态筛选要求登录 | implemented；真实账号筛选 pending | pending |
| 两站每日一题、题目详情 | `internal/leetcode/client.go`、Core 公开网络工作流 | verified | pending |
| 只读文本题面、清除活动 HTML | `internal/leetcode/statement.go`、`client_test.go` | Core 生成预览与边界测试 verified；桌面预览 pending | pending |
| 从真实题目模板创建工作区解题文件、不覆盖已有代码 | `internal/plugin/plugin.go`、Core 工作流 | verified；多根选择框 pending | pending |
| Python 3、Java、C++、Go、Rust、JavaScript、TypeScript、C# 模板 | `internal/leetcode/client.go` | Python 3 verified；其余 implemented，取决于题目可用模板 | pending |
| 当前编辑器实时缓冲区、上传前确认、站点/题目/语言标识校验 | `internal/plugin/plugin.go`、`client_test.go` | 标识边界 verified；真实编辑器上传 pending | pending |
| 手动输入 Cookie、验证账号后保存、两站凭据隔离、退出登录 | `internal/plugin/host.go`、`plugin.go` | 无凭据拒绝与注入边界 verified；真实账号 pending | pending |
| 在线样例、提交、查询判题、取消等待 | `internal/leetcode/client.go`、`internal/plugin/plugin.go` | 不重复上传与取消边界 verified；真实账号判题 pending | pending |
| 系统浏览器打开官方登录页，手动关联后验证并保存凭据 | `internal/plugin/plugin.go`、`ui/index.html`、`internal/plugin/i18n.go` | implemented；自动回传未实现，最终入口 pending | pending |
| 中文站题解列表、正文和热门顶层评论，分页与只读预览 | `internal/leetcode/community.go`、`internal/plugin/community.go` | 较早 Core 公开网络流程已读取真实内容；最后交互修改 pending | pending |
| 社区条目选中、方向键和回车、独立计数与返回题库 | `ui/main.js` | implemented / pending | pending |
| 状态原子写入、后端重启恢复站点与语言、保存未完成任务 ID | `internal/plugin/state.go`、Core 工作流 | 站点/语言恢复 verified；真实远端任务恢复 pending | pending |
| 网页工具窗口、中英文、深浅色、键盘选择 | `ui/`、`internal/plugin/i18n.go` | implemented；截图与桌面集成结果见验证记录 | pending |
| 40 条分页、最多 400 条、可见行虚拟化、翻页失败后手动重试 | `ui/main.js`、`internal/plugin/plugin.go` | implemented；真实滚动结果见验证记录 | pending |
| 系统浏览器打开原题 | `internal/plugin/plugin.go` | implemented；桌面集成 pending | pending |
| 判题报告进入「运行」面板、失败退出状态、再次查看不上传 | `internal/plugin/run.go`、`cmd/backend/main.go`、`ui/main.js` | implemented / pending；未新增 Core API | pending |
| 页面与后端版本不一致时明确提示，中文词条保留 | `ui/main.js`、`internal/plugin/view.go`、`internal/plugin/i18n.go` | implemented / pending | pending |
| LeetCode SVG 模板图标与固定来源许可证 | `icons/leetcode.svg`、`resources.lock.json`、`plugin.json` | implemented / pending | pending |
| SDK 结构化日志、关闭高频成功记录 | `internal/plugin/plugin.go`、`docs/verification.md` | Core 通道与事件 verified；桌面日志窗口 pending | pending |
| 工作流前后插件包文件清单与哈希不变 | `scripts/verify/core-workflow.py` | verified | pending |

会员题、本地判题/调试、自定义测试输入、历史提交、复习计划和模拟面试未实现。图片与公式保留原站查看入口，首版题面只呈现文本与代码。
