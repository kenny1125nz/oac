# OAC

A workspace-based runtime for generative AI task execution, together with related agent tooling and workflow documentation.

## Contents

- `oac-sdk/` — Python package `oac`, a workspace-based runtime for generative AI task execution
- `ash/` — Ash integration (Go)
- `ash-vscode/` — VS Code tooling for Ash
- `software_enhance.md` — human + AI software enhancement lifecycle
- `passport-photo-requirements.ash` — example Ash workflow
- `merge_passport_data.py` — data merge helper

## Getting started

Python SDK:

```bash
cd oac-sdk
pip install -e .
```

Ash components:

```bash
cd ash
go build ./...
```

## Documentation

- `oac-sdk/oac.vision.md` — SDK vision
- `software_enhance.md` — clarify / design / implement / review lifecycle

## Status

Mixed workspace of an SDK, integrations, and workflow experiments.
