# 构建资源与生命周期

| 路径 | 所有者与用途 | 生命周期 | 可跨检出复制 |
| --- | --- | --- | --- |
| `go.mod`, `go.sum`, `sdk.lock.json` | 插件固定 SDK、HTML 解析库、参考宿主版本 | 版本控制 | 可以，须保持同一提交 |
| Go 全局 module/build cache | Go 工具链管理的下载与编译缓存 | 工具链管理 | 不另设、不复制、不清空 |
| `ui/messages.js` | 从后端中英文词条生成的静态资源 | 构建前生成并提交 | 相同词条版本可以 |
| `icons/leetcode.svg`, `licenses/simple-icons-CC0-1.0.txt`, `resources.lock.json` | 固定 Simple Icons 提交和内容哈希的 LeetCode 模板图标及许可证 | 构建时校验并打入包，运行时只读 | 同一来源提交和哈希可以 |
| `.artifacts/package/<系统>-<架构>` | 构建脚本组装的只读插件包 | 本地构建产物 | SDK/源码/平台/架构/Go 工具链完全相同可以 |
| `dist/*.zip`, `*.sha256` | `package.py --all` 交叉编译 macOS/Windows/Linux arm64/x64 六个包及校验文件 | 构建产物，显式上传 GitHub Release；已发布的包不替换 | 源码、SDK、平台、架构与 Go 工具链完全相同可以 |
| `.artifacts/release/v0.0.1-source/` | 从 `git archive v0.0.1` 导出的固定发布输入，补齐 Windows/Linux 安装包 | 本次发布构建私有目录，不是 Git 检出；可在发布结束后移除 | 不复制其中运行时状态；Go 全局缓存照常共用 |
| 宿主 `stateDirectory/practice.json` | 站点、语言和未完成远端任务 ID | 运行时，随插件/工作区状态清理 | 不可复制到另一个工作区或检出 |
| 宿主 credentials | 按插件与站点隔离的账号秘密 | 宿主系统凭据库，退出登录删除 | 不可复制 |
| 工作区 `leetcode/` | 用户解题代码 | 用户所有 | 用户自行管理 |
| `run/start.temporaryFiles` 的 `judge-report` | 宿主在自己的临时目录创建 UTF-8 JSON，交给同一打包程序的报告模式，只含已完成判题报告和任务标识，不含凭据或源码 | 宿主在运行结束后清理，不在插件状态目录持久化；单报告正文最多 128 KiB | 不可复制 |

只写宿主通过 `workspace/getContext` 明确授予的状态目录；不从可执行文件、app bundle、插件目录推导可写路径。状态文件采用同目录临时文件、同步、原子替换；不存代码、题面、判题输出或凭据。状态没有变化时不写磁盘。

后端和网页只从已安装插件读取。不会改写插件包，不新增应用运行时资源，不影响 app 签名或 Sparkle delta。端到端验证比较工作流前后的包文件清单、权限、长度和 SHA-256。验证数据只在插件 `.artifacts/verify/`，不放入宿主/Core 的缓存或源码。

Windows 包声明 `bin/backend.exe`，macOS/Linux 包声明 `bin/backend`；Run 报告模式复用对应目标的同一程序。交叉编译不执行目标二进制，不需要 Windows/Linux 本地运行时，也不代表宿主 WebView、凭据库或原生窗口已验收。

判题报告保留在后端内存中；关闭工作区、切换站点或后端退出时释放。Run API 按需启动同一个包内可执行文件，直接输出真实报告，退出状态反映判题通过与否，不另建下载目录、日志或常驻程序。它与 SDK 协议模式是两个入口，只有报告模式的 stdout 交给「运行」控制台；后端正常模式的 stdout 仍只写协议帧。
