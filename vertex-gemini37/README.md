# CPA Vertex Gemini 3.7 plugin

This is a small, independent CLIProxyAPI native plugin. It registers
`gemini-3.7-flash` plus `-low`, `-medium`, `-high`, and `-tiered` aliases,
routes them to CPA's built-in `vertex` provider, and writes
`generationConfig.thinkingConfig.thinkingLevel` for the three explicit levels.
It does not handle credentials, headers, URLs, projects, or locations; CPA's
native Vertex executor remains responsible for those.

The plugin does not implement cross-provider load balancing. CPA's normal
round-robin strategy still applies to multiple credentials inside the selected
`vertex` provider. The model router priority only decides which plugin handles
duplicate model IDs; lowering it does not alternate Vertex and Antigravity.

To switch quota providers on demand, leave this plugin enabled and toggle the
Vertex credential in CPA. With no active Vertex credential, the built-in router
skips this plugin's unavailable `vertex` target and the lower-priority
Antigravity route can handle the same model IDs. Re-enabling a Vertex
credential makes Vertex the selected route again.

## Build

The target needs Go and a C compiler:

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./...
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -buildmode=c-shared -o vertex-gemini37.so .
rm -f vertex-gemini37.h
```

Install the `.so` as `plugins/linux/amd64/vertex-gemini37.so` and enable the
matching `plugins.configs.vertex-gemini37.enabled` entry. A CPA restart is
required after changing the plugin or config.

## Safety behavior

- Only the five exact model IDs above are handled.
- Only Gemini/Vertex-format payloads are changed.
- Invalid or non-JSON payloads pass through unchanged.
- The base model and `-tiered` keep the upstream thinking default.
- No credential, API key, project ID, or region is embedded in the plugin.
