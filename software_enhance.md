# Software Enhancement — Small/Medium Feature Lifecycle

```
Enhancement
│
├── 1. Clarify
│     YOU → Write a comprehensive brief: problem, scope, and acceptance criteria
│     AI  → Research: find related issues/PRs, scan impacted code, estimate effort, flag risks, propose refinements
│     YOU → Review findings; decide go/no-go
│
├── 2. Design
│     YOU → Sketch the approach: data flow, component touchpoints, API surface
│     AI  → Generate: diagrams from the sketch, trace code paths, enumerate edge cases and failure modes
│     YOU → Walk through with a peer; approve and proceed
│
├── 3. Implement
│     YOU → Write task checklist (each item independently testable) and test specifications (input → expected output)
│     AI  → Write: code, tests, integrations, error handling, wiring
│     YOU → Self-review the diff end-to-end; verify tests pass and lint is clean
│
├── 4. Review
│     YOU → Open the PR with summary, linked issue, and evidence it works
│     AI  → Draft: code changes and replies in response to reviewer feedback
│     YOU → Read every comment; apply or push back; get approval; verify CI is green
│
├── 5. Validate
│     YOU → Deploy to staging
│     AI  → Run: acceptance suites, diff staging output against expected, generate smoke-test checklist
│     YOU → Manual smoke-test with real data and real user flows
│
├── 6. Release
│     YOU → Merge, deploy, authorize
│     AI  → Monitor: surface anomalies, correlate log spikes, suggest rollback if needed
│     YOU → Smoke-test in production; decide on rollback
│
└── 7. Close
      YOU → Update docs, changelogs, and impacted runbooks
      AI  → Draft: changelog entries, new docs; find stale sections and dead code paths; generate cleanup diffs
      YOU → Approve accuracy and completeness; ship cleanup; choose what to build next
```
