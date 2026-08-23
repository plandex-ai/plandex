---
sidebar_position: 3
sidebar_label: Build Remote Agent
---

# Pair a phone with Build Remote Agent

Plandex can use **Build Remote Agent** as a pairing device: the paid
iOS/Android app spectates (and can inject into) this desktop agent through the
free MIT `gbr-agent`. Phone and PC never open ports to each other.

Website: https://grokbuildremote.com/
Agent: https://github.com/LinespottingOrg/GrokBuildRemote-Agents (MIT)
Protocol: `gbr/1` · need agent **v0.6.0+**

Not affiliated with xAI or SpaceX.

This is not a Plandex Cloud or self-host replacement. Pairing stays on the machine that runs the Plandex CLI: install `gbr-agent`, pair, then attach only loopback.

## Install + pair

```bash
# macOS / Linux
curl -fsSL https://grokbuildremote.com/install.sh | bash
gbr-agent version          # must print v0.6.0 or newer
gbr-agent pair             # QR in browser + printed 8-char code
gbr-agent run              # leave running
```

```powershell
# Windows
irm https://grokbuildremote.com/install.ps1 | iex
gbr-agent version
gbr-agent pair
gbr-agent run
```

Phone: open Build Remote Agent → **Scan QR from computer** (or type the 8-char
code). Sessions appear in the app. **Unpair** in Settings before changing PCs.
Force-close is not enough.

## Attach

After `gbr-agent run`:

- HTTP Bot API: `http://127.0.0.1:8788`
- MCP stdio: clone the agent repo and run `node mcp/gbr-mcp/bin/gbr-mcp.js`

```bash
curl -sS http://127.0.0.1:8788/health
curl -sS http://127.0.0.1:8788/v1/sessions
```

Phone is spectator. Orchestration stays on this Plandex session (or a Grok bot /
Claude Cowork talking to the same Bot API).

Do not commit mailbox keys. Phone **Settings → Bot API** is the only place the
relay key is copied.
