# 个人维护说明

仓库：[xiaoyumuxi/LeetCode](https://github.com/xiaoyumuxi/LeetCode)，默认分支 `main`。插件的源码、测试、翻译、资源与功能矩阵都在本仓库维护。宿主和 Core 只提供公共能力，不放力扣业务。

## 从哪里改

| 位置 | 用途 |
| --- | --- |
| `plugin.json` | 插件身份、权限、工具窗口与后端入口 |
| `internal/leetcode/` | 两站 GraphQL、题目模板、判题与中文社区接口 |
| `internal/plugin/` | 公共宿主 API 接线、凭据、工作区状态、预览与 Run 报告 |
| `ui/` | 工具窗口页面、交互和宿主主题适配 |
| `internal/plugin/i18n.go` | 唯一中英词条来源；修改后生成 `ui/messages.js` |
| `scripts/build/package.py` | 构建本机平台的开发安装包 |
| `sdk.lock.json` | Core/SDK 与参考宿主提交；固定版本不等于已经验证 |
| `docs/features.md`、`docs/issues.md`、`docs/verification.md` | 功能矩阵、已知缺口、历史验证与日志接线 |

GitHub 所有者与 Go module 为 `xiaoyumuxi/LeetCode`。插件 ID 保持 `io.github.lithe-idea.leetcode`，以保留已有安装、工作区状态和系统凭据的身份；仓库归属变化不等于插件身份迁移。未来若要换 ID，需同时迁移清单里的窗口/命令/激活事件及状态，不能仅替换一处字符串。

## 构建和安装

使用 `go.mod` 指定的 Go 工具链和 Python 3，在本仓库执行：

```sh
go mod download
python3 scripts/build/package.py
```

打包脚本生成词条、核对固定图标哈希并编译后端。产物为 `dist/LeetCode-0.0.1-<系统>-<架构>.zip` 及 SHA-256 文件。Intel 架构文件名使用 `x64`；可通过 `GOOS=darwin GOARCH=amd64 python3 scripts/build/package.py` 交叉构建。在 Lithe 中从磁盘安装整个 zip；升级时需要完整更新后端和页面，重新启用并重开面板。出现版本不一致提示时重启 Lithe。

本次按用户要求发布 `0.0.1`，从最终源码构建 macOS arm64/x64 包；不复用此前本地开发安装包。所需宿主修复见下方“宿主依赖”，使用旧宿主无法获得窗口图标、错误分类和浮层修复。

## 日常维护

从 `main` 切功能分支，中文说明具体改动，按仓库 PR 模板提交。凭据、日志、编译产物及个人解题代码不得提交。公共网络与真实账号验收单独进行，离线协议测试不冒充登录或判题成功。

修改文案后执行 `go run ./scripts/build/translations.go`。需要自行验证时，使用 README 和 `docs/verification.md` 的入口，只运行受影响范围。GitHub 默认提供 Linux 检查，不启动 macOS/Windows runner；检查工作流不代替桌面和真实账号验收。

功能增加升次版本号，修复升修订号；依赖未发布宿主能力时仅提供开发安装或预发布，不直接进入市场。首次公开版本按用户指定为 `0.0.1`，作为 GitHub 预发布提供，不进入插件市场。源代码尚未添加授权许可证，公开仓库不代表已选择开源许可；第三方许可见 `THIRD_PARTY_NOTICES.md`。

## 本次已接线和剩余缺口

- 题库、每日题、题面与解题文件；中文站题解和热门顶层评论进入编辑区只读预览。
- 完成后的真实判题报告进入宿主「运行」面板，可重新查看而不上传；Run 停止按钮只管理报告进程。
- 浏览器登录入口已接线，但浏览器会话尚不能自动回传。仍需手动关联 Cookie 并验证账号；不得把打开登录页当成登录成功。
- 国际站社区、真实账号端到端判题、最终桌面外观和资源消耗尚未验收或实现，详情见问题清单。

用户要求自行验收，最后实现没有运行测试、静态检查或原生验证；发布请求之后只进行了词条生成和安装包构建。

## 宿主依赖

公共 API 来自宿主 [#2372](https://github.com/1lck/Lithe-IDEA/pull/2372) 和 Core [#194](https://github.com/Lithe-IDEA/Lithe-core-rust/pull/194)。本轮通用修复见宿主 [#2383](https://github.com/1lck/Lithe-IDEA/pull/2383)（已合入）；`sdk.lock.json` 已固定包含最后图标接线的宿主提交。宿主 PR 只包含通用修复，插件无需修改 Core 协议。
