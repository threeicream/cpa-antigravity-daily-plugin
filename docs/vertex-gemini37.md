# CPA Vertex Gemini 3.7 插件

## 用途

`vertex-gemini37` 只负责模型注册、路由和请求体适配。它不保存或生成 Vertex 凭证，也不修改 Vertex URL、API Key、服务账号 JSON 或区域配置。

注册的客户端模型名：

- `gemini-3.7-flash`
- `gemini-3.7-flash-low`
- `gemini-3.7-flash-medium`
- `gemini-3.7-flash-high`
- `gemini-3.7-flash-tiered`

这些名称路由到 CPA 内置 Vertex provider 的上游模型 `gemini-3.7-flash`。`low`、`medium`、`high` 会写入对应的 `thinkingLevel`；基础名和 `tiered` 保留上游默认设置。

## 与 Antigravity 的关系

示例配置中 Vertex 插件优先级为 20，Antigravity tiered 插件优先级为 10。同名模型在 Vertex 凭证可用时优先走 Vertex；停用所有 Vertex 凭证后，CPA 会跳过不可用的 Vertex 目标，让较低优先级的 Antigravity 路由处理。

优先级不是跨 provider 轮询：

- `routing.strategy: round-robin` 只会在已选定的 provider 内，在可用凭证之间轮询。
- 只有一个 Vertex 凭证时不会产生可见轮换。
- 把 Vertex 插件降到 10 或更低，只会让 Antigravity 先处理，不能让两者交替使用。

## 构建和安装

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o vertex-gemini37.so .
rm -f vertex-gemini37.h
```

将 `vertex-gemini37.so` 放入 CPA 的 `plugins/linux/amd64/`，启用 `vertex-gemini37` 配置并重启 CPA。停用时关闭该配置即可回滚。

## 凭证和区域

Vertex API Key 或服务账号 JSON 由 CPA 原生 Vertex 执行器处理。插件不会把 `global`、`us` 或具体区域硬编码到请求中；项目、区域和认证方式由 CPA 的 Vertex 配置/凭证决定。如果某区域的模型目录不包含 `gemini-3.7-flash`，仅安装本插件不会改变该区域的上游可用性。

仓库只包含源码、测试和说明，不包含真实凭证、API Key、项目密钥、管理密钥或编译产物。
