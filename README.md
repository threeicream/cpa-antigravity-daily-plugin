# CPA Antigravity Daily Credential Plugin

这是面向 CLIProxyAPI v7.2.113 的本地动态库插件。它不会替换 CPA 的请求执行器，只负责通过 `daily-cloudcode-pa.googleapis.com` 完成 Antigravity OAuth、发现真实 `project_id`，再把原生 `antigravity` 凭证交给 CPA Host 保存。

登录页面会读取 CPA 使用的 `cli-proxy-theme` 设置，并跟随“羊毛纸 / 白色 / 深色 / 自动”主题；CPA 在其他标签页切换主题时，插件页面也会同步更新。

## 安全模型

- Google access/refresh token 只经 CPA Host 的 HTTP transport 和本机内存处理。
- 最终凭证由 CPA 的认证存储接口原子写入 AuthDir。
- 浏览器资源页面本身未认证，只承载静态 HTML；启动和轮询接口仍受 CPA 管理密钥保护。
- 页面不会把管理密钥写入 localStorage、sessionStorage 或 Cookie。
- 如果 daily API 没有返回真实 `project_id`，插件会报错并拒绝保存，不使用共享或伪造的回退项目。

## 构建

目标环境需要 Go 和 C 编译器：

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared \
  -o antigravity-daily.so .
rm -f antigravity-daily.h
```

## CPA 配置

从 GitHub Releases 下载 `antigravity-daily-linux-amd64.so`（或按上文自行构建），放入 CPA 插件目录的 `linux/amd64` 子目录，并重命名为 `antigravity-daily.so`。文件基础名必须与插件配置 ID 对应：

```text
plugins/linux/amd64/antigravity-daily.so
```

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    antigravity-daily:
      enabled: true
      priority: 1
      client_id: "YOUR_GOOGLE_OAUTH_CLIENT_ID"
      client_secret: "YOUR_GOOGLE_OAUTH_CLIENT_SECRET"
```

`client_id` 和 `client_secret` 必须来自允许 `http://localhost:51121/oauth-callback` 的 Google OAuth 已安装应用。插件不会内置或公开上游项目的 OAuth 凭据；这两个值只保存在你的 CPA 配置和最终凭证中。管理页面及插件配置 API 应继续由 CPA 管理密钥保护。

重启 CPA 后打开：

```text
/v0/resource/plugins/antigravity-daily/login
```

## 3.7 Flash 档位插件

`antigravity-tiered/` 是独立的 CLIProxyAPI 原生插件：它注册
`gemini-3.7-flash-low`、`-medium`、`-high` 和 `-tiered` 四个客户端模型名，
将请求路由到内置 Antigravity 的 `gemini-3.7-flash-tiered`，并为前三个别名
设置对应的 `thinkingLevel`。它不读取或修改 OAuth 凭证，构建和部署说明见
[`antigravity-tiered/README.md`](antigravity-tiered/README.md)。

## 许可证与来源

本项目复刻了 [GCLI2API](https://github.com/su-kaka/gcli2api) 的 Antigravity daily endpoint 认证流程，因此按其 CNC-1.0 条款以非商业方式发布；详情见 [LICENSE](LICENSE)。CPA 插件 ABI 的接口形状参考 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（MIT）。完整归属说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
