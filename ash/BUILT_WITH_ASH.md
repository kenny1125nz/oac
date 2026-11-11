# Built with Ash

> **Ash** fuses deterministic control flow with AI autonomy — a scripting language where the structure is rigid and the steps are intelligent.

---

## The Problem

AI agents are advancing rapidly. They're increasingly capable of handling ambiguity, exercising judgment, and producing quality results from natural language prompts alone. For many tasks, a well-crafted prompt is all you need.

But **for certain classes of work, deterministic control remains the optimum answer** — regardless of how capable the AI becomes:

- **Workflows as code.** A multi-step process — review, fix, test, deploy — should live in a file you can commit, version, share, and rerun. Not buried in a chat thread or someone's head.
- **Repeatability.** A CI pipeline, a failing test suite, a recurring report — these need to run the same way every time, even if the content of each step varies.
- **Guaranteed sequencing.** "First fix, then test, then review, and only deploy if all three pass" — this is a property of the control flow, not the intelligence of any single step.
- **Bounded autonomy.** You want an agent to exercise judgment within a step, not to decide whether the step happens at all, or how many times, or what comes next.
- **Execution efficiency.** Ever watched an AI agent think for ages about a step that should be instant? When the agent drives the entire workflow, every decision — even "what next?" — burns tokens and adds latency. Pre-determined control flow runs at CPU speed: branching, looping, and sequencing happen in microseconds. AI time is spent only where it adds value: the autonomous work within each step.

**What we need is a fusion of the two.** Deterministic control where it's the right tool: sequencing, branching, retry limits, evaluation gates, parallel coordination. AI autonomy where it shines: understanding intent, handling variation, making judgment calls within a step. Not because one is weak, but because each is optimal for different parts of the problem.

Ash scripts can invoke agents. Agents can invoke scripts. The boundary dissolves — you're composing both in a single, version-controlled artifact, each doing what it does best.

**Ash fuses deterministic structure with AI autonomy. Not as a workaround. As the optimal architecture.**

---

## What Ash Does

Ash is a scripting language where the control flow is deterministic but the steps are autonomous. You write the skeleton — the gates, the retry policy, the evaluation criteria — and AI agents fill in the flesh. The same script runs the same way every time, but each agent invocation brings intelligence to the step it owns.

```ash
#!opencode:1.2.0

# Load a prompt from a file — no inline walls of text
# Run a shell command as a step: compile, test, lint, whatever
# Let the agent decide how to handle the result

for FILE in exec git diff --name-only origin/main...HEAD {
  exec prettier --write FILE
  exec eslint FILE

  try {
    do @prompts/review.md with subagent code-reviewer
  } evaluate with {
    do @prompts/quality-check.md with subagent reviewer
  } accept {
    print "${FILE} — passed"
  } partial {
    do "Refine: ${report}" with subagent code-reviewer
  } fail {
    exec git checkout -- FILE
    print "${FILE} — reverted, needs manual review"
  } upto 2
}

try {
  exec npm run build
} fail {
  do "The build failed. Fix the errors below and recompile.

Build output:
${stderr}" with subagent bug-fixer
} upto 3

if $? != 0 {
  print "build still failing after 3 attempts"
  exit 1
}
```

A more deeply fused scenario: the agent itself decides to call an Ash script as a tool, which in turn orchestrates more agents. The prompt file tells the agent what tools are available:

```markdown
# prompts/incident-response.md

You are an on-call diagnostician. Investigate the production incident
using the logs and metrics available on the system.

If a workaround is needed to restore service now:
  - Call `ash apply_fix.ash <component>` to apply the workaround.
  - The script handles the operational change and checks that service is restored.

Always open a follow-up ticket for the permanent fix. This is for
incident resolution, not root cause.
```

The agent bridges the prompt instructions to the script:

```ash
#!opencode:1.2.0

do @prompts/incident-response.md with subagent diagnostician
```

The same pattern turns Ash workflows into reusable skills. Define the skill as a markdown file that the agent reads — describing when to invoke the tool — with an Ash script as the implementation:

```markdown
# skills/content-writer.md

You are a content writer. When asked to create a blog post:

1. Research the topic thoroughly using available sources.
2. Draft the post in a clear, engaging style.
3. Once the draft is complete, call the publish workflow:

   ash publish_workflow.ash TOPIC="<topic>"

The publish workflow handles review, quality checks, and rendering.
Do not attempt to publish the post yourself — always use the workflow.
```

```ash
#!opencode:1.2.0
# publish_workflow.ash — the deterministic pipeline behind the skill

TOPIC = env ASH_TOPIC

do @prompts/review.md with subagent editor
do @prompts/fact-check.md with subagent researcher

exec pandoc draft.md -o "posts/${TOPIC}.pdf"
print "${TOPIC} — published"
```

The skill file is what the agent reads — it defines behavior and delegates the publish step to a script. The Ash script is the deterministic pipeline: review, fact-check, render. Three steps, fixed order, no ambiguity.

---

## Key Capabilities

### Deterministic Structure, Autonomous Steps
Ash gives you rigid, predictable control flow — sequencing, branching, retry limits, evaluation gates, parallel coordination — while each individual step is executed autonomously by an AI agent. You decide *what must happen, in what order, under which conditions, with what guardrails*. The agent decides *how to do it*. The script runs the same way every time; the agent handles the ambiguity.

### Agents Can Call Scripts, Scripts Can Call Agents
This is not a one-way orchestration layer. An Ash script can invoke agents as steps. An agent can invoke an Ash script as a tool. That script might invoke more agents, which might invoke more scripts. This creates a **fused system** — deterministic and intelligent execution woven into a single runtime, not stacked in layers.

### Retry with Learning
Agents fail. Ash's `try/evaluate/fail` blocks give retries context from previous attempts via `${stderr}`, `${stdout}`, and `${report}`, so each retry learns from the last failure instead of repeating the same mistake.

### Parallel Execution
`wait { ... }` runs all enclosed statements concurrently and waits for completion. Fire-and-forget with `&` for background tasks. Review 20 files in parallel, then run tests.

### Context Compacting
Agent context windows are finite. Ash provides `compact` directives — per-agent, standalone, or global — to truncate or summarize context before it overflows. No more degraded output from bloated context.

### Natural but Deterministic
Ash reads like pseudocode, not configuration. Variables, `if`/`else`, `for`/`while`, functions with parameters, string interpolation — familiar constructs that anyone can follow at a glance. But the execution is rigid: branching, looping, and sequencing are controlled by the script, not by an LLM deciding what to do next. The syntax is approachable; the behavior is predictable.

### Works With Your Tools
`.ash` files are plain text. Commit them, review them in PRs, run them in CI, run them on a schedule — they work with whatever version control, automation, and collaboration tools you already use.

---

## Language Features at a Glance

| Feature | Syntax |
|---|---|
| Variables | `NAME = "value"`, reference by bare name |
| Arrays | `["a", "b", "c"]`, index with `arr[0]`, concatenate with `+` |
| Agent call | `do "prompt" with subagent profile` |
| Shell commands | `exec cmd`, inline `$(cmd)` |
| String interpolation | `"hello ${NAME}"` |
| Conditionals | `if` / `else if` / `else` |
| Loops | `for VAR in LIST`, `while COND` |
| Functions | `fn name(params) { ... }` |
| Retry | `try { } fail { } upto N` |
| Evaluated retry | `try { } evaluate with { } accept { } partial { } fail { } upto N` |
| Parallel | `wait { ... }`, `{ ... } &` |
| Working directory | `within <path> { ... }`, per-agent `in <path>` |
| Context management | `compact "truncate 32000"` |
| Includes | `include "lib/prompts.ash"` |
| File prompts | `do @skills/review.md with subagent reviewer` |
| Exit code | `$?` |
| Engine declaration | `#!opencode:1.2.0` |

---

## Why Not Just Use a Shell Script?

Shell scripts give you determinism. AI agents give you autonomy. Neither alone is the optimal architecture for multi-step intelligent workflows.

- **Shell scripts** are deterministic and repeatable, but every step must be explicitly coded — they can't handle the ambiguity, judgment, or creative variation that agents excel at.
- **Prompt-only workflows** are flexible and intelligent, but they're a single opaque step — no structured retry, no evaluation gates, no guaranteed sequencing.
- **Layering one on the other** (a script that calls an agent once) is shallow — you get one autonomous step inside a rigid shell, not a deeply fused system.

Ash is the fusion point:

- **Structured retry** — not "call this again", but "retry up to N times, feeding the last attempt's output as learning context, with a deterministic evaluation gate"
- **Evaluation gates** — an agent step must pass a check (by another agent) before the workflow proceeds; the criteria are encoded, the judgment is autonomous
- **Scoped parallelism** — `wait { ... }` groups concurrent work with proper scope isolation and deterministic completion semantics
- **Cross-engine portability** — the shebang line decouples your workflow from any specific agent implementation
- **Bidirectional calling** — agents can invoke Ash scripts as tools, scripts can invoke agents as steps; the runtime is shared, not stacked

Ash is a few hundred lines of Go, compiles to a single static binary, and runs anywhere.

---

## Use Cases

Ash is domain-agnostic. Here are examples across different domains:

### Software Engineering
**Automated PR Review.** Run a reviewer across changed files, a quality-checker evaluator to verify, and a fixer on any failures — all in parallel.

**Batch Refactoring.** "Rename this pattern across 200 files." Split the work, run it in parallel, compact context between batches.

**CI Pipeline.** Run tests, capture failures, feed them to a bug-fixer, retry with learning, and only fail the build if the agent can't fix it after N attempts.

### Content & Publishing
**Automated Newsletter.** Pull topics from a data source, generate drafts with a writer agent, have an editor evaluate quality, retry weak drafts, and compile the final issue.

**Documentation Sync.** For each changed API endpoint, have a documenter update the reference docs, a reviewer check for accuracy, and publish on pass.

### Research & Analysis
**Competitive Intelligence.** Run a researcher agent across N competitors in parallel, collect findings, feed to an analyst agent for synthesis, produce a report.

**Data Pipeline.** Extract data via `exec` commands, feed to an analyst agent for interpretation, iterate with `evaluate` to sharpen insights, output structured results.

### Operations
**Incident Response.** On alert, run a diagnostician agent against logs, a remediator if root cause found, escalate to human if unresolved after N retries.

**Scheduled Audits.** Weekly: check configs for drift, have a compliance agent review against policy, generate a report card.

### Onboarding
Give new team members `.ash` scripts that encode your team's AI workflows — code review, writing style, research process — so they can run a proven process from day one instead of learning by trial and error.

---

## Getting Started

```bash
# Install
go install github.com/agentic-coding/ash/cmd/ash@latest

# Write a workflow
cat > review.ash << 'EOF'
#!opencode:1.2.0
for FILE in exec git diff --name-only origin/main...HEAD {
  do "Review ${FILE}" with subagent code-reviewer
}
EOF

# Run it
ash review.ash
```

Ash is **open source**, **language-agnostic**, and **agent-agnostic**. Whatever your agents can do, Ash can orchestrate.
