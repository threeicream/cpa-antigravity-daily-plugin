# CPA Antigravity Daily plugin

This directory contains the complete `antigravity-daily` credential plugin.
It uses the daily endpoint to discover a real project ID and asks CPA Host to
store the resulting native Antigravity credential. It does not provide Vertex
credentials and does not replace CPA's built-in Vertex executor.

## Build

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o antigravity-daily.so .
rm -f antigravity-daily.h
```

Install the resulting `.so` as `plugins/linux/amd64/antigravity-daily.so` and
enable the matching `plugins.configs.antigravity-daily` entry. After restarting
CPA, open `/v0/resource/plugins/antigravity-daily/login` from the protected
management page.

The OAuth `client_id` and `client_secret` belong in CPA's protected plugin
configuration only. Never commit them or any resulting token.
