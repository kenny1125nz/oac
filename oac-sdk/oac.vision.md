# Workspace-Based Runtime for Generative AI Task Execution

When applied to generative use cases, the agent is essentially modifying a complex state (a codebase, a document set, an image canvas, or any structured output). You cannot simply let an untrusted agent write directly to production artifacts. Treating the agent execution as an isolated branch off a repository is exactly how we make generative AI safe and deterministic at the software call site.

Here is how that general agent-calling pattern maps to native git operations, optimized for any generative task.

## The Repository Operations Model

| Operation        | Purpose                                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------------------- |
| Checkout Branch  | Isolates the agent by creating a temporary branch from the target repository.                                  |
| Commit           | Captures the state of the branch after a sub-task before moving to the next.                                   |
| Merge            | Applies the branch changes back to the target branch directly.                                                 |
| Create Pull Request | Opens a pull request for human review; the caller moves on without blocking. |
| Reset            | Resets the branch to a previous commit, discarding failed changes.                                             |
| Evaluate + Retry | Inspects the branch, assesses the report against criteria, and feeds findings back into the agent for a retry. |

## The General Calling Pattern (Git-Based SDK)

To formalize this into a reusable engineering pattern, your software-to-agent client should expose a repository boundary. Below is a structural blueprint of how this pattern looks in practice:

```python
# The parent application invokes the generative agent using this pattern
def run_generative_task(
    intent: str,
    evaluator: Evaluator,
    repo_path: str,
):
    # 1. INIT: Checkout an isolated branch from the target repository
    with Repository(path=repo_path).checkout() as branch:

        retry_count = 0
        max_retries = 3
        current_intent = intent

        while retry_count < max_retries:
            # Capture a commit before the agent attempts work, assume commit returns commit id only if no changes
            commit_id = branch.commit() 

            # 2. EXECUTE: Call the black-box generative agent
            generative_agent.execute(
                intent=current_intent,
                workspace=branch.path,
            )

            # 3. EVALUATE: Inspect then assess
            # Each task brings its own evaluator — it encapsulates both
            # the inspection logic and the decision policy.
            report = evaluator.inspect(branch.path)
            assessment = evaluator.assess(report)

            if assessment.status == "ACCEPTABLE":
                # 4. MERGE or CREATE PULL REQUEST:
                #   - merge:              apply directly (no human gate)
                #   - create_pull_request: open a PR, caller moves on
                if needs_review:
                    branch.create_pull_request()
                else:
                    branch.merge()
                return "Task completed successfully."

            elif assessment.status == "PARTIAL":
                current_intent = (
                    f"Refine previous work. "
                    f"Findings: {assessment.findings}. Full Report: {report}."
                )
                retry_count += 1

            elif assessment.status == "FAIL":
                # 5. RESET: Reset branch back to the last commit
                branch.reset(commit_id)

                current_intent = (
                    f"Previous attempt failed. "
                    f"Report: {report}. Errors: {assessment.errors}"
                )
                retry_count += 1

        raise RuntimeError("Agent failed to produce valid output within retry limits.")
```

## Core Mechanics of the Pattern

- **Isolation by Default**: `Repository.checkout()` creates a temporary branch from the target repository. The agent never interacts with the main branch or live environment.
- **Commits**: A commit captures the branch state at a point in time — a local git commit, a file system snapshot, or any state you can return to instantly.
- **Reset on Failure**: If the evaluator determines the agent produced invalid, unsafe, or low-quality output (FAIL), the branch is reset to the last commit. The failed output is discarded instantly.
- **Feedback-Driven Retries**: Instead of failing the call immediately, the code loops. Each task brings its own evaluator, encapsulating both the inspection logic and the decision policy. `inspect` returns a structured report (e.g., an XML test report, a JSON scan result, or any inspection artifact); `assess` evaluates that report against internal thresholds to produce a decision. Both the report and detailed findings are injected back into the agent as text input, saying: "You tried X, but the inspection found Y. Try again from the last commit."
- **Merge or Create Pull Request**: `branch.merge()` applies the branch directly — no human gate. `branch.create_pull_request()` opens a PR and returns immediately; the caller moves on to other work without blocking on human review.

## Why this Pattern is Necessary

If you do not wrap agent calls in this pattern, your application suffers from **State Pollution**. An agent that fails halfway through a task will leave partial, broken artifacts in your repository, requiring manual developer intervention to clean up.

By scoping each agent call to an isolated branch, your main application loop remains completely pure. The call either succeeds entirely (merge) or acts as if it never happened (discard + reset), maintaining perfect system stability.

Does this architectural boundary match what you're building? If you want, we can map out how the branch state data should look when passing retry history back to the agent, or look at how git-based branching handles the commit mechanics.
