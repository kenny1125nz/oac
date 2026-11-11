# Human in the Loop

**Status:** Design proposal

Real-world workflows need real-time human feedback — review, approve, choose, or provide input. Humans interact through different devices (laptop, phone, watch) and expect mobility. A messaging system as a separate component is a given — this document assumes it exists, along with presentation devices that render messages appropriately for the form factor.

All four interaction types use the `ask` keyword as the unifying verb. The second keyword sets the interaction type: `to approve`, `to choose`, `for`, or `to edit`.

---

## Approval

A guarded block that executes only if the human approves. If denied, the block is skipped. If no response arrives, the script fails with an error:

```ash
ask release-team to approve "Deploy v2.0 to production?" {
  exec deploy production
}
```

To show the human context for their decision, attach a file or variable:

```ash
ask security to approve "These dependency changes" @audit-report.md {
  exec npm audit fix
}
```

An explicit `timeout` subclause overrides the default error behavior:

```ash
ask release-team to approve "Proceed?" {
  exec deploy
} timeout "5m" {
  print "no response, aborting"
  exit 1
}
```

## Choice

Present a question and branch based on the human's selection. `default` is optional — without it, timeout fails the script:

```ash
ask release-team to choose "Which environment?" {
  case "staging" {
    exec deploy staging
  }
  case "production" {
    ask release-team to approve "Production deploy?" {
      exec deploy production
    }
  }
  case "cancel" {
    print "deployment cancelled"
  }
} timeout "5m" default "cancel"
```

## Input

Ask the human for a free-form value. `default` is optional — without it, timeout fails the script:

```ash
TAG = ask release-team for "What version tag?"
exec git tag TAG
```

With an explicit fallback:

```ash
TAG = ask release-team for "What version tag?" \
  timeout "5m" default "v0.0.0"
```

## Edit

Present a file for the human to modify, capture the result. The human gets the file, edits it, and returns the new content. `default` is optional — without it, timeout fails the script:

```ash
signed = ask legal to edit @release-notes.md \
  timeout "30m" default @release-notes.md

exec commit signed
```

A timeout with `default @file` means "use the original file unchanged if no response." Without `default`, the script errors out.

## Actor declaration

Human participants are a constraint for the workflow to function — they need to be declared upfront so the script can't run without valid actors. A shebang-like header line declares actors and their routing keys:

```ash
#!opencode:1.2.0
#!actor release-team = "slack:#releases" legal = "email:legal@company.com"
```

The actor name becomes a bare identifier in `ask` statements — validated at parse time, not an arbitrary string:

```ash
ask release-team to approve "Deploy?" {
  exec deploy production
}

signed = ask legal to edit @release-notes.md
```

Undeclared or misspelled actors are errors at parse time. The routing key (`slack:`, `email:`, `sms:`, etc.) is resolved by the runtime.

## Design considerations

- **Context.** The human may need to see more than just the message — the last agent output, a diff, relevant variables. The message body needs a way to reference workflow state.
- **Concurrency.** Inside `wait { }`, prompting the human in parallel doesn't make sense — `ask` should either serialize or be disallowed in parallel blocks.
- **Checkpoint and resume.** Each `ask` is a natural yield point. The runtime checkpoints scope and position, exits, and a later invocation resumes from the checkpoint with the human response. The runtime must serialize execution state to a checkpoint file.

## Open question

Should `ask` be a dedicated language construct, or modeled as a special subagent? `do "approve this?" with subagent human` would delegate the prompt/response lifecycle to the engine, keeping the language unchanged but tying behavior to the engine implementation.
