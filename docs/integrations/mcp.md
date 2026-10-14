# MCP Boundary

MCP is not part of the browser product flow. It is a technical boundary owned by the `liki` domain project and consumed by trusted runtimes or external Skills through explicit paths.

## Engine MCP

Canonical path:

```text
/engine/mcp/{domain}
```

Supported Engine domains include:

```text
bazi
ziwei
qimen
liuyao
aux
```

Engine MCP exposes deterministic tools only.

## Counsel MCP

Canonical paths:

```text
/counsel/mcp/{domain}
```

Supported Counsel domains:

```text
bazi
ziwei
naming
liuyao
qimen
```

## Boundaries

- MCP tools do not perform payment authorization.
- MCP tools do not expose cross-user profile data.
- Browser Liki Chat does not directly connect to MCP.
- Public MCP routes are governed by Caddy, logging, and backend rate limits.
