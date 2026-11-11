# Enhanced Agent Abstraction

**Status:** Design proposal

The current `engine.Engine` interface flatly assumes "agent = local CLI binary you shell out to." This breaks for non-CLI agent types. A clean abstraction separates how Ash communicates with an agent from how each agent type implements that contract.

---

## Agent types

| # | Type | Example | Interface to Ash |
|---|------|---------|-----------------|
| 1 | **Local CLI** | opencode, claude, codex | Subprocess (`os/exec`) |
| 2 | **Remote API** | bolt.new, lovable, custom SaaS | HTTP/WebSocket API |
| 3 | **Containerized** | Workers in Docker, k8s pods | Container runtime API (Docker, k8s) |
| 4 | **In-browser** | JS agent running with user's LLM key | WebSocket / CDP / postMessage |

---

## Two-layer abstraction

**`Adapter`** — how to reach and execute an agent:

```go
type Adapter interface {
    Type() string                          // "local-cli" | "api" | "container" | "browser"
    Validate(ctx context.Context, config *AgentConfig) error
    Setup(ctx context.Context, config *AgentConfig) error    // start container, connect WS, etc.
    Execute(ctx context.Context, config *AgentConfig, req *ExecuteRequest) (*ExecuteResponse, error)
    Teardown(ctx context.Context, config *AgentConfig) error // stop container, disconnect, etc.
}

type ExecuteRequest struct {
    Prompt  string
    Model   string
    Dir     string
    Timeout time.Duration
}

type ExecuteResponse struct {
    Stdout   string
    Stderr   string
    ExitCode int
}
```

**`LocalCLIDriver`** — maps abstract request to CLI flags (for type 1 only):

```go
type LocalCLIDriver interface {
    Name() string                                    // "opencode", "claude", etc.
    BuildArgs(req *ExecuteRequest) (cmd string, args []string, env []string)
    ParseVersion(output string) string               // extract semver from --version output
}
```

This separates two concerns the current code mixes: (1) how the adapter reaches its target, and (2) what CLI flags a specific agent uses.

---

## Agent config

```yaml
agents:
  # Type 1: Local CLI
  coder:
    type: local-cli
    driver: opencode
    cmd: opencode
    args: [run]

  # Type 2: Remote API
  saas-coder:
    type: api
    base_url: https://api.bolt.new/v1
    auth:
      type: bearer
      token_env: BOLT_KEY
    endpoints:
      execute:
        method: POST
        path: /sessions/run
        request_template:
          prompt: "$.input.text"
          model: "$.model"
        response_mapping:
          stdout: "$.output.text"
          exit_code: "$.status.code"

  # Type 3: Containerized
  sandbox-coder:
    type: container
    runtime: docker                        # or: k8s
    image: opencode-worker:latest
    mode: run                              # lifecycle: run (create+destroy) or exec (attach)
    volumes:
      - .:/workspace

  # Type 4: In-browser
  ui-agent:
    type: browser
    transport: websocket
    url: ws://localhost:9999/agent
```

---

## Project-level config

Transport and infrastructure wiring belongs in a separate project file (`ash-project.yaml`), not in the script — the script references agents by name only:

```yaml
# ash-project.yaml
version: 1

agents:
  coder:
    type: local-cli
    driver: opencode

services:
  postgres-test:
    image: postgres:16
    ports: ["5432:5432"]
    env:
      POSTGRES_PASSWORD: test
    health_check: "pg_isready"

secrets:
  - ASH_API_TOKEN
  - DEPLOY_KEY
```

The CLI reads the project config, starts dependency services, sets up agent transports, runs the script, then tears down.

### Agent name resolution at compile time

The compiler reads `ash-project.yaml` at parse time and registers every agent name. When it encounters `with <agent>` in a script, it validates the name against the registry immediately:

| Scenario | Compiler behavior |
|----------|------------------|
| `ash-project.yaml` present + `with <agent>` in script | Validate agent name against config. Unknown name = compile-time error. |
| No project config + `with <agent>` in script | Compile-time error — "no agents configured" |
| No project config + no agent calls | Works fine (pure shell/exec workflow) |

This makes agent names **parser-aware first-class identifiers**. A VS Code extension (or any LSP) can highlight, autocomplete, and cross-reference them without running the script. The project config is the single source of truth — no redundant declaration in the script itself.

---

## Key design properties

- **Adapters are orthogonal to drivers** — any driver runs through any adapter. OpenCode can run locally, in Docker, or via a remote API. The script never changes.
- **Minimal language change** — `do <prompt> with <agent> [subagent <name>]` replaces `do <prompt> with subagent <name>`. The `<agent>` identifier selects an agent instance from the config. `with subagent <name>` remains as shorthand for `with <shebang-engine> subagent <name>`.
- **`exec` stays local** — shell commands always run on the host, regardless of where the agent executes.
- **API adapter is template-driven** — new SaaS agents can be added via config alone, no Go recompilation.
- **Container lifecycle is explicit** — `mode: run` (Ash manages start/stop) vs `mode: exec` (attach to existing).
- **Browser agents are first-class** — same `Execute` interface over WebSocket. The agent could be a plain HTML/JS page using the user's own API key.
