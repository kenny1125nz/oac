# Ash — Agent-Native Script Language: Gap Analysis

## Current State

| Capability | Status |
|---|---|
| Agent invocation (`do ... with subagent`) | ✅ implemented |
| Retry with learning (`try/fail/upto` + `${stderr}`) | ✅ implemented |
| Evaluated quality gates (`evaluate with/accept/partial/fail`) | ✅ implemented |
| Context compaction (`truncate/summarize/window/drop`) | ✅ implemented |
| Parallel + background execution (`wait { }`, `&`) | ✅ implemented |
| Shell integration (`exec`, `$(cmd)`, `env`) | ✅ implemented |
| Functions (`fn`, `return`, scoping) | ✅ implemented |
| File path loading (`@"prompt.md"`) | ✅ implemented |
| Engine/version declaration (shebang header) | ✅ implemented |

## Hard Gaps — Needed for Production Use

### 1. Artifact Management

Agents produce files — patches, diffs, generated code, reports. The script needs to know what was changed, verify it, and potentially roll it back.

```
# Proposed syntax:
changes = diff of src/      # declare tracked artifacts
verify $changes with "npm test"    # gate on verification
publish $changes to "branch/fix"   # publish to a target
```

Why: Without artifact tracking, each agent call is a black box. You can't chain agent outputs, compare results across retries, or audit what changed.

### 2. Human Approval Gates

Not every step should auto-proceed. A pipeline needs checkpoints where a human reviews and approves before continuing.

```
# Proposed syntax:
approve changes by reviewer {
  print "Review the diff and approve to continue"
  print $changes
}
# Pipeline pauses here, waits for human input
print "continuing after approval"
```

Why: Production deployments, security-sensitive changes, and architectural decisions need human judgment. No LLM should auto-approve a database migration or auth system change.

### 3. Tool/Sandbox Declarations

Different subagent roles need different permissions. A `bug-fixer` needs file write + shell; a `reviewer` only needs read.

```
# Proposed syntax:
tools for bug-fixer {
  allow: write("src/"), exec("npm", "git"), read(all)
  deny: write("*.env"), exec("rm"), network(external)
}
```

Why: Running untrusted or AI-generated commands without sandboxing is a security risk. Declaring permissions at the script level makes intent explicit and enables enforcement.

### 4. Observability

In production, you need cost tracking, timing, and success metrics. How much did this run cost? Which step is the bottleneck?

```
# Proposed syntax:
report {
  step: "review",
  tokens_in: 4500,
  tokens_out: 1200,
  cost: 0.003,
  duration_ms: 2300,
  result: "pass"
}
```

Why: LLM API costs are real and variable. Without observability, you're flying blind on cost and performance. A single failed retry loop could silently burn $50.

### 5. Session/State Persistence

Real agent workflows span multiple invocations. A multi-day refactoring project needs context to persist across script runs.

```
# Proposed syntax:
session "refactor-2026" {
  context: checkpoint("after-review")
  resume: true
}
```

Why: Not all work fits in one script execution. Long-running tasks need checkpoint/restore, cumulative context management, and state that survives restarts.

## Soft Gaps — Would Elevate the Model

### 6. Structured Prompt Assembly

Prompts are flat strings. Real prompts have structure — task description, rules, context, examples.

```
# Proposed syntax:
prompt for reviewer {
  task: "Review uncommitted changes for bugs"
  rules: "Focus on correctness, security, edge cases"
  context: ${stdout}
  examples: ["Good: handles null input", "Bad: SQL injection"]
}
```

Why: Composition of prompt sections from different sources (environment, prior output, configuration) is messy with string concatenation. Structured blocks make prompts maintainable and composable.

### 7. Dynamic Agent/Model Dispatch

Can't choose agent or model at runtime. If one model fails, you want to escalate to a stronger one.

```
# Proposed syntax:
if $SCORE < 0.8 {
  model = "claude-opus"       # escalate
} else {
  model = "claude-sonnet"     # default
}
do "fix this" with subagent bug-fixer using $model
```

Why: Cost optimization and quality-based routing need variables in `using` and `subagent` clauses. You shouldn't pay for the expensive model when the cheap one suffices.

### 8. Token Budgeting

No way to declare or enforce token limits. Models charge by token; scripts should budget by token.

```
# Proposed syntax:
do $TASK with subagent reviewer {
  budget: 16000 tokens
  on_budget_exceeded: compact("truncate 8000")
}
```

Why: Without budget enforcement, a bad prompt or wrong agent can consume thousands of dollars in tokens in a loop. Budgets make cost predictable.

### 9. Inter-Agent Structured Context

Agent A's findings can't be structured input to Agent B beyond `${stdout}`. Need typed slots for composability.

```
# Proposed syntax:
review = do "review auth.ts" as findings {
  type: "code-review",
  format: "bullet-list"
}
fix = do $review with subagent fixer
```

Why: Chaining agents on unstructured text is fragile. Structured outputs (findings, patches, reports) make agent-to-agent handoffs reliable.

### 10. Incremental Refinement Loops

`partial { }` exists in retry but it's fix-then-retry. Missing long-running agent sessions with progress hooks.

```
# Proposed syntax:
agent "refactorer" working on "large-refactor" {
  on progress: print "still working..."
  on checkpoint: persist context
  timeout: 30min
}
```

Why: Some tasks are too large for a single call. An agent should be able to work incrementally, checkpoint progress, and report back without blocking the pipeline.

## Assessment

Ash's value proposition is durable against improving LLMs for three reasons:

| Reason | Detail |
|---|---|
| **Context management is structural** | Better models don't eliminate the need to truncate — they just mean compaction strategies can be smarter. The compaction primitives are the right abstraction regardless. |
| **Determinism is the runtime's job** | `if tests pass → deploy, else → rollback` should never be probabilistic. Control flow is the orchestration layer's responsibility. |
| **Role-based composition is the pattern** | Real teams use reviewer → fixer → evaluator → deployer. Different prompts, models, permissions per role. Ash formalizes the handoff. |

The 5 hard gaps above are where ash moves from a spec to a production runtime.
