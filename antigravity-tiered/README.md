# CPA Antigravity 3.7 Flash tiered plugin

This is a small, independent CLIProxyAPI native plugin. It registers the
`gemini-3.7-flash-low`, `-medium`, `-high`, and `-tiered` client model IDs,
routes them to the built-in `antigravity` provider's native
`gemini-3.7-flash-tiered` model, and forces the low/medium/high aliases to the
corresponding Antigravity `request.generationConfig.thinkingConfig.thinkingLevel`.

The plugin deliberately does not read, write, refresh, or alter OAuth
credentials. The existing `antigravity-daily` plugin remains responsible for
credential login and storage.

## Build

The target needs Go and a C compiler (the Steam Deck has a cached Go builder
image, so the deployment procedure builds in an isolated rootless container):

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o antigravity-tiered.so .
rm -f antigravity-tiered.h
```

Install the `.so` as `plugins/linux/amd64/antigravity-tiered.so` and enable the
matching `plugins.configs.antigravity-tiered.enabled` entry. A CPA restart is
required after changing the plugin or config.

## Safety behavior

- Only the four exact model IDs above are handled.
- Only an Antigravity-format payload is changed.
- Invalid or non-JSON payloads pass through unchanged.
- `-tiered` is routed without forcing a level; the upstream/default setting is
  retained.
