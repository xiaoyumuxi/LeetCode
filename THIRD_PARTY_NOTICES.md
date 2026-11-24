# 第三方资源

- Lithe 官方 Go 插件 SDK：`github.com/Lithe-IDEA/Lithe-core-rust/shared/plugins/sdk/go`，Apache-2.0，固定版本见 `go.mod`、`go.sum` 和 `sdk.lock.json`。SDK 随后端静态编译，不包含单独的运行时或下载器。许可证全文随包提供于 [licenses/lithe-sdk-Apache-2.0.txt](licenses/lithe-sdk-Apache-2.0.txt)，原文来自锁定 Core 提交的 `shared/plugins/LICENSE`；该版本没有单独的 NOTICE 文件。SDK 源码未修改。
- `golang.org/x/net/html`：BSD 3-Clause，版本与内容哈希见 `go.mod` 和 `go.sum`，用于解析题面。
- Go 标准库与运行时：BSD 3-Clause，随编译后的后端分发，许可证全文见 [licenses/go-BSD-3-Clause.txt](licenses/go-BSD-3-Clause.txt)。
- LeetCode 图标：来自 Simple Icons 提交 `585b247ff866e6d359124bcf3d63c4346dfaed03`，以模板 SVG 随插件分发；文件来源和 SHA-256 见 [resources.lock.json](resources.lock.json)，CC0-1.0 全文见 [licenses/simple-icons-CC0-1.0.txt](licenses/simple-icons-CC0-1.0.txt)。标识对应的商标权归其所有者；插件不代表 LeetCode 官方。
- 题目、模板与判题结果由用户选定的 LeetCode / 力扣站点在线提供，权利归各自所有者；插件包不包含题库快照。

## Go x/net 许可证

Copyright 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

* Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
* Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
