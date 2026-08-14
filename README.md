# CLIProxyAPI 扩展插件集合

本仓库包含三个可以独立编译、独立安装和独立启停的 CLIProxyAPI 原生插件。仓库共用许可证和构建说明，但每个插件都有自己的目录、动态库文件名和 CPA 配置 ID。

## 插件选择

| 插件 | 源码位置 | 动态库文件 | CPA 配置 ID | 用途 |
| --- | --- | --- | --- | --- |
| Antigravity Daily | 仓库根目录 | `antigravity-daily.so` | `antigravity-daily` | Antigravity OAuth 登录、项目发现和凭证保存 |
| Antigravity 3.7 Tiered | `antigravity-tiered/` | `antigravity-tiered.so` | `antigravity-tiered` | `gemini-3.7-flash` 的 Antigravity 档位别名 |
| Vertex Gemini 3.7 | `vertex-gemini37/` | `vertex-gemini37.so` | `vertex-gemini37` | `gemini-3.7-flash` 的 Vertex 路由和请求适配 |

只使用某个插件时，只构建并启用对应的动态库和配置项；不需要把三个插件全部启用。

## 构建

目标环境需要 Go 和 C 编译器。三个目标必须分别构建：

```bash
# Antigravity Daily（仓库根目录）
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o antigravity-daily.so .
rm -f antigravity-daily.h

# Antigravity 3.7 Tiered
cd antigravity-tiered
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o antigravity-tiered.so .
rm -f antigravity-tiered.h

# Vertex Gemini 3.7
cd ../vertex-gemini37
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o vertex-gemini37.so .
rm -f vertex-gemini37.h
```

将需要的 `.so` 文件放入 CPA 插件目录的 `linux/amd64` 子目录。动态库的基础名必须与配置 ID 一致：

```text
plugins/linux/amd64/antigravity-daily.so
plugins/linux/amd64/antigravity-tiered.so
plugins/linux/amd64/vertex-gemini37.so
```

## CPA 配置

下面是完整结构示例；只启用实际需要的条目。示例中的 OAuth 值是占位符，不要把真实凭证提交到 Git：

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
    antigravity-tiered:
      enabled: true
      priority: 10
    vertex-gemini37:
      enabled: true
      priority: 20
```

修改插件文件或配置后重启 CPA。Antigravity Daily 登录入口为：

```text
/v0/resource/plugins/antigravity-daily/login
```

Vertex 插件不生成、不保存也不修改 Vertex API Key、服务账号 JSON、项目 ID 或区域；这些由 CPA 原生 Vertex 执行器和管理页面负责。

## 模型与路由

`antigravity-tiered` 注册并路由：

- `gemini-3.7-flash-low`
- `gemini-3.7-flash-medium`
- `gemini-3.7-flash-high`
- `gemini-3.7-flash-tiered`

`vertex-gemini37` 注册并路由：

- `gemini-3.7-flash`
- `gemini-3.7-flash-low`
- `gemini-3.7-flash-medium`
- `gemini-3.7-flash-high`
- `gemini-3.7-flash-tiered`

当前示例优先级为 Vertex 20、Antigravity 10。同名模型在 Vertex 凭证可用时优先走 Vertex；停用 Vertex 凭证后，CPA 会跳过不可用的 Vertex 目标并允许较低优先级的 Antigravity 路由处理。

这不是跨 provider 轮询。CPA 的 `routing.strategy: round-robin` 只在选定的 provider 内轮询多个可用凭证；如果要在 Vertex 和 Antigravity 之间交替，需要另行实现跨 provider 调度逻辑。降低 Vertex 插件优先级不会产生这种轮询。

详细的 Vertex 边界说明见 [`docs/vertex-gemini37.md`](docs/vertex-gemini37.md) 和 [`vertex-gemini37/README.md`](vertex-gemini37/README.md)。

## 安全模型

- 不要把 Google OAuth secret、API Key、服务账号 JSON、项目密钥或 CPA 管理密钥提交到仓库。
- 认证文件只应由 CPA 的受保护管理接口或本机受限目录保存。
- 构建产物 `.so` 和生成的 `.h` 文件默认被 `.gitignore` 排除；发布时应通过受控 Release/Artifact 分发。

## 许可证与来源

本项目复刻了 [GCLI2API](https://github.com/su-kaka/gcli2api) 的 Antigravity daily endpoint 认证流程，因此按其 CNC-1.0 条款以非商业方式发布；详情见 [LICENSE](LICENSE)。CPA 插件 ABI 的接口形状参考 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（MIT）。完整归属说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
