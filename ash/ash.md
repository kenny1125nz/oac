# Ash — Agent Shell Language Specification

**Version:** 0.1.0
**File Extension:** `.ash`

Ash is a scripting language for composing AI coding agents (Copilot, OpenCode, Claude Code, etc.) into automated code modification workflows.

---

## Engine Declaration

The first line declares the target agent and version. An optional `#!compact` line configures context management globally:

```ash
#!opencode:1.2.0
#!opencode:1.2.0:sonnet

#!opencode:1.2.0
#!compact mode=on window=64000 strategy=truncate
```

Format:

| Line | Pattern                          | Description                                                         |
| ---- | -------------------------------- | ------------------------------------------------------------------- |
| 1    | `#!<engine>:<version>[:<model>]` | **Required.** Target agent engine with version.                     |
| 2 | `#!compact <key>=<val> ...` | **Optional.** Global compact settings. Space-separated key=val pairs. |

The runtime checks the installed agent version matches the declared version.

---

## Variables

Variables are assigned without a prefix and referenced by bare name:

```ash
FILES = exec find src -name '*.ts'
DIFF = exec git diff origin/main...HEAD
```

Reference without prefix:

```ash
MSG = "diff: ${DIFF} files: ${FILES}"
if DIFF == "" {
  print "no changes"
}
```

The `$` prefix is also accepted for backward compatibility (e.g. `$FILES`, `$DIFF`). `$?` remains the canonical form for the exit code variable.

### Built-in Runtime Variables

| Variable    | Set by                              | Description                                 |
| ----------- | ----------------------------------- | ------------------------------------------- |
| `$?`        | any statement                       | Exit code of the last command or agent call |
| `${stderr}` | `try { } fail { }`                  | Stderr output from the failed attempt       |
| `${stdout}` | `try { } fail { }`                  | Stdout output from the failed attempt       |
| `${report}` | `try { } evaluate with { }`         | Stdout from the evaluator block             |

### Array variables

Variables can also hold arrays — see [§3 Arrays](#3-arrays).

---

## Arrays

Arrays are ordered, comma-separated lists of values enclosed in square brackets:

```ash
COUNTRIES = ["United States", "United Kingdom", "Canada"]
SCORES    = [95, 87, 92, 78]
MIXED     = ["hello", 42, true]
EMPTY     = []
```

### Index access

Access elements with zero-based index using square brackets:

```ash
COUNTRIES = ["US", "UK", "Canada"]
print COUNTRIES[0]   # "US"
print COUNTRIES[2]   # "Canada"
```

### Built-in functions

| Function  | Description                            |
|-----------|----------------------------------------|
| `len(a)`  | Returns the number of elements in `a`  |

### Array concatenation

The `+` operator concatenates two arrays:

```ash
A = [1, 2, 3]
B = [4, 5]
C = A + B    # [1, 2, 3, 4, 5]
```

### Arrays in for loops

`for` loops iterate over array elements directly:

```ash
for ITEM in ["apple", "banana", "cherry"] {
  print ITEM
}
```

### Strings still work as lists

Existing newline-separated string iteration continues to work:

```ash
for ITEM in "a\nb\nc" {
  print ITEM
}
```

---

## Strings & Text Blocks

### Double-quoted strings

```ash
MSG = "task ${ID} completed in ${TIME}s"
```

### Escape sequences

| Escape | Result                            |
| ------ | --------------------------------- |
| `\"`   | Literal `"`                       |
| `\$`   | Literal `$` (stops interpolation) |
| `\\`   | Literal `\`                       |

### Text blocks

Multi-line literals delimited by triple backticks. No escape sequences — only ` ``` ` closes the block:

````
PROMPT = ```
Fix the login bug in src/auth/login.ts.
The error is "Invalid token" when the JWT expires.
Use the existing retry utility in lib/retry.ts.
```
````

Variable interpolation `${NAME}` and inline commands `$(cmd)` work inside both string types.

---

## Includes

```ash
include "lib/prompts.ash"
include "lib/git.ash"
include "config/projects/${REPO_NAME}.ash"
```

---

## Control Flow

All control structures use `{ }` for bodies, matching `try { }` and `fn { }`:

### If / else if / else

```ash
if $? == 0 {
  print "build passed"
}

if $? == 0 {
  print "pass"
} else if SCORE > 0.8 {
  print "good enough"
} else {
  print "fail"
  exit 1
}
```

### For

```ash
for FILE in FILES {
  exec eslint FILE
}
```

### While

```ash
while RETRIES < MAX_RETRIES {
  do "Fix remaining issues" with subagent bug-fixer
  RETRIES = RETRIES + 1
}
```

### Break / Continue

Inside loops, `break` exits the loop and `continue` skips to the next iteration:

```ash
for FILE in FILES {
  if FILE == "skip_this.ts" {
    continue
  }
  do "Review ${FILE}" with subagent reviewer
  if $? != 0 {
    break
  }
}
```

### Expressions

Comparisons and arithmetic use shell-like operators:

```ash
$? == 0            # equality
COUNT > 5          # comparison
OK and READY       # boolean
not FAILED
TOTAL + 1          # addition
N * 2              # multiplication
```

Use parentheses for grouping: `(X > 0 and Y < 10)`.

### For Splitting

`for VAR in LIST` iterates over each element. When `LIST` is an array, it yields elements directly. When `LIST` is a string, it splits on **newlines** (one item per line). This matches the output of `exec find`, `exec ls`, and similar commands.

### Working directory: `within`

`within <path> { ... }` sets the working directory for all agent calls inside the block. The directory must already exist — the engine fails with an error if it doesn't. Inside the block, a per-agent `in <path>` clause overrides the block's directory for that single call.

```ash
within "/home/kenny/apps/agents/passport-photo" {
  do "Research passport requirements"    # runs in that directory
  do "Take measurements"                 # same directory
  do "Crop image" in "/tmp"             # per-agent override
}
```

See also the [`do` agent call syntax](#7-agent-call) for the per-agent `in` clause.

### Parallel execution

`wait { ... }` runs all statements in the block concurrently and waits for every one to finish:

```ash
wait {
  do "Review src/a.ts" with subagent reviewer
  do "Review src/b.ts" with subagent reviewer
  do "Review src/c.ts" with subagent reviewer
}
print "all reviews done"
```

Works with any statement — `exec`, `do`, function calls:

```ash
wait {
  exec npm run build
  review_and_fix("src/a.ts", "gpt-4o")
  review_and_fix("src/b.ts", "gpt-4o")
}
```

The `&` suffix fires a single statement in the background and continues immediately (fire-and-forget):

```ash
exec npm install &
do "Deploy to staging" with subagent deployer &
print "fire and forget — script continues immediately"
```

---

## Agent Call

The core action — invoke an agent with instructions.

### Syntax

```
do <prompt>
  [with <agent> [subagent <name>]]
  [using <model>]
  [in <path>]
  [compact <directive>]
```

All subclauses are optional. Their order is fixed: `with` → `subagent` → `using` → `in` → `compact`.

| Part | Default | Description |
|------|---------|-------------|
| `do <prompt>` | required | Prompt as a string, text block, variable, or `@file.md` |
| `with <agent>` | shebang engine | Agent instance from the [agent abstraction](enhanced-agent-abstraction.md). Selects which configured agent to invoke (e.g. `local-opencode`, `remote-coder`, `sandbox-reviewer`). |
| `subagent <name>` | `default` | Sub-agent capability/profile within the selected agent instance |
| `using <model>` | agent default | Override the AI model |
| `in <path>` | `within` block's dir or none | Working directory for the agent subprocess. Directory must exist. |
| `compact <directive>` | none | Post-agent context management (always last — runs after the agent finishes) |

The `with <agent>` clause refers to an agent instance declared in the project config (see [enhanced-agent-abstraction.md](enhanced-agent-abstraction.md)). This is a **big shift** — a script is no longer tied to a single shebang engine. It can orchestrate tasks across different agent types in one script.

For backward compatibility, `with subagent <name>` is shorthand for `with <shebang-engine> subagent <name>`.

### Examples

Basic usage with a single local agent:

```ash
do "Fix the login bug in src/auth/login.ts"
do "Fix this" with local-opencode subagent bug-fixer
do "Fix this" with local-opencode subagent bug-fixer using claude-sonnet-4
do "Fix this" with local-opencode subagent bug-fixer in "/workspace" compact "truncate 32000"
```

Orchestrating across different agent types — the script selects which agent instance to use at each call:

```ash
# Code generation via local CLI
do "Implement the user auth module" with local-opencode subagent coder

# Security review via a remote SaaS agent
do "Review auth module for vulnerabilities" with remote-security-agent subagent reviewer

# Run tests inside a containerized agent
do "Fix failing tests and verify" with sandbox-tester subagent bug-fixer

# Deploy via a containerized deployer
do "Deploy to production" with sandbox-deployer subagent deployer
```

The prompt can be a variable, text block, or external file:

````
do PROMPT with remote-coder subagent bug-fixer

do ```
  Review all uncommitted changes.
  Check for: correctness, security, style, edge cases.
``` with local-opencode subagent code-reviewer using gpt-4o

do @skills/refactor.md with sandbox-refactorer subagent refactorer in "/project"
do @skills/review.md with cloud-reviewer subagent code-reviewer using gpt-4o compact "truncate 16000"
do @'play_step${n}.md' with local-opencode subagent coder    # variable in path
do @${DIR}/prompts/task.md with remote-tester subagent tester
````

`@<path>` loads the prompt from a markdown file (relative to the script). Variables in the path are interpolated: `@'play_step${n}.md'` or `@${DIR}/file.md`.

The `in <path>` clause is typically used inside a [`within` block](#6-control-flow) to override the directory for a single agent call. When used outside a `within` block, it sets the working directory for that call only.

---

## Retry

### Binary retry (exit code based)

Retries an agent call when it exits with non-zero. The `fail` block lets each retry learn from the previous failure using `${stderr}` and `${stdout}`:

````
try {
  do "Fix all TypeScript errors in src/" with subagent bug-fixer
} fail {
  do ```
Fix the remaining errors:
${stderr}
Files: $(find src -name '*.ts')
  ``` with subagent bug-fixer compact "truncate 32000"
} upto 3
````

Without `fail`, retrying the exact same prompt is pointless. The `fail` block lets each retry learn from the previous failure — inject error output, adjust instructions, or switch subagent.

### Evaluated retry (three outcomes)

For tasks where exit code alone is not enough, provide an evaluator that inspects the workspace and returns an outcome.

**Execution flow** — the agent runs first. If it fails (exit code non-zero), the evaluator is **skipped** and execution jumps straight to `fail`:

```
1. Agent runs
2. exit != 0  →  skip evaluator, go to fail { }
3. exit == 0  →  run evaluate with { }
   exit 0  = accept
   exit 1  = partial
   exit 2+ = fail
```

The evaluator uses its **exit code** to signal the outcome. Its stdout is available as `${report}`:

````
try {
  do "Refactor src/auth/" with subagent refactorer
} evaluate with {
  do @skills/check-quality.md with subagent quality-checker
} accept {
  print "refactor accepted"
} partial {
  do "Refine based on evaluation: ${report}" with subagent refactorer
} fail {
  do "Reset and try different approach: ${report}" with subagent refactorer
} upto 3
````

| Clause | Triggered when | Effect |
|--------|---------------|--------|
| `evaluate with { }` | Agent succeeded (exit 0) | Runs the evaluator block. Its exit code routes to accept/partial/fail. |
| `accept { }` | Evaluator exit 0 | Exits the retry loop, continues script. |
| `partial { }` | Evaluator exit 1 | Retries. `${report}` has evaluation findings (evaluator stdout). |
| `fail { }` | Agent exit != 0 **or** evaluator exit >= 2 | Retries from clean state. `${report}` has error details. |
| `${report}` | In partial / fail | Stdout from the evaluator block. |

Without `evaluate with`, the simpler `fail` form applies (exit code 0 = accept, non-zero = fail).

---

## Context Compacting

Coding agents accumulate large contexts. Compact can be specified in three ways:

### Per-agent subclause

Attach `compact` directly to a `do` call. The engine runs the compact operation **after** the agent finishes, compressing the context before the next step:

```ash
do "Fix bugs in src/" with subagent bug-fixer compact "truncate 32000"
do PROMPT with subagent debugger compact "window 64000 keep src/"
```

### Standalone directive

Triggers a one-time compact on the current context:

```ash
compact "truncate 32000"
compact "summarize"
compact "drop node_modules/,dist/"
```

### Global declaration

Declared in the header via a `#!compact` line — applies to every agent call in the script:

```ash
#!opencode:1.2.0
#!compact mode=auto window=32000 strategy=truncate
```

| Key        | Values                        | Description               |
| ---------- | ----------------------------- | ------------------------- |
| `mode`     | `auto` (default), `on`, `off` | Whether compact is active |
| `window`   | token count                   | Context window limit      |
| `strategy` | `truncate`, `summarize`       | How to reduce context     |

Global and standalone compact can be overridden by a per-agent `compact` clause.

---

## Tools (Built-in)

Coding agents operate on code. Ash exposes these built-ins:

| Call         | Description                      |
| ------------ | -------------------------------- |
| `exec cmd`   | Run shell command, return stdout |
| `include "file"` | Include another ash script       |
| `env KEY`    | Environment variable             |
| `print msg`  | Print message                    |
| `exit code`  | Exit script                      |

### Inline command substitution

Inside strings and text blocks, `$(cmd)` runs a shell command and substitutes its stdout:

````ash
MSG = "current branch: $(git branch --show-current)"
PROMPT = ```
Changes:
$(git diff --name-only)
```
````

This differs from the statement-level `exec cmd` — `$(cmd)` is an inline expression that expands to a string.

---

## Functions

### Declaration

```ash
fn get_src_files() {
  result = exec find src -name '*.ts'
  return result
}

fn review(FILE, MODEL) {
  do "Review ${FILE}" with subagent code-reviewer using MODEL
}

fn deploy(ENV, TAG) {
  exec npm run build
  do "Deploy ${TAG} to ${ENV}" with subagent deployer
}
```

- `fn NAME(PARAM1, PARAM2) { ... }` — named parameters in parentheses, body in braces.
- Parameters are local variables inside the block.
- Empty parens `()` for no-argument functions.

### Calling

```ash
src = get_src_files()
review("src/x.ts", "gpt-4o")
deploy("staging", "v1.2")
```

- Brackets **required** — distinguishes `name()` from `exec cmd`.
- Arguments are positional, comma-separated.
- Return value captured via `$?` or variable assignment.

### Return

```ash
fn get_src_files() {
  result = exec find src -name '*.ts'
  return result
}

fn validate(CODE) {
  if CODE == "" {
    return "empty"
  }
}
```

- `return <expr>` — exits function, sets `$?` and optionally a value.
- No `return` → function returns exit code of the last statement.

### Scoping

- Variables created inside `{ }` are **local** — not visible outside.
- Outer variables are readable but not writable inside a function.

### Error handling

Functions work naturally with `try` and `compact`:

```ash
fn review_files(PATTERN, MODEL) {
  for FILE in exec find . -name PATTERN {
    try {
      do "Review ${FILE}" with subagent code-reviewer using MODEL
    } fail {
      do "Fix errors in ${FILE}: ${stderr}" with subagent bug-fixer compact "truncate 16000"
    } upto 2
  }
}

review_files("*.ts", "gpt-4o")
```

---

## Complete Example

````ash
#!opencode:1.2.0
#!compact mode=on strategy=truncate window=64000

include "lib/prompts.ash"
include "lib/review.ash"

fn review_and_fix(FILE, MODEL) {
  try {
    do "Review ${FILE} for bugs and performance issues" with subagent code-reviewer using MODEL
  } evaluate with {
    do "Check if review addressed all issues in ${FILE}" with subagent quality-checker
  } accept {
    print "${FILE} looks good"
  } partial {
    do "Refine: ${report}" with subagent code-reviewer compact "truncate 16000"
  } fail {
    do "Reset and re-review: ${report}" with subagent code-reviewer
  } upto 2
}

for FILE in exec find src -name '*.ts' {
  review_and_fix(FILE, "gpt-4o")
}

exec npm test
if $? != 0 {
  try {
    do "Fix test failures in $(git diff --name-only -- '*.test.ts')" with subagent bug-fixer
  } fail {
    do ```
Previous fix failed. Errors:
${stderr}
    ``` with subagent bug-fixer
  } upto 3

  exec npm test
  if $? != 0 {
    print "tests still failing, aborting"
    exit 1
  }
}

compact "truncate 16000"
do "Deploy to staging" with subagent deployer
````

---

## TODO

### 1. Human in the Loop

See [human-in-the-loop.md](human-in-the-loop.md) for the design proposal covering approval, choice, input, edit, actor declaration, and open questions.

### 2. Enhanced Agent Abstraction

See [enhanced-agent-abstraction.md](enhanced-agent-abstraction.md) for the design proposal covering the Adapter/LocalCLIDriver abstraction, agent types (local CLI, remote API, containerized, in-browser), agent config, project-level config, and key design properties.
