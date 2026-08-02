# TicketLens ML

Multi-task ticket classifier workspace: synthetic data, evaluation, training,
calibration, and a FastAPI inference service that implements the Go
`port.Classifier` contract.

This workspace lives **inside the backend** (`backend/ml/`), not at the repo
root: everything AI/ML belongs to the backend side of the system. Paths below
are relative to `backend/ml` unless stated otherwise.

## Layout

```
backend/ml/
  src/ticketlens_ml/
    taxonomy.py     # mirrors Go AllCategories / priorities
    generate.py     # synthetic corpus (persona × sector × tone × length)
    split.py        # stratified train/val/test splits
    evaluate.py     # head-wise metrics + stub baseline
    train.py        # shared encoder + two classification heads
    calibrate.py    # temperature scaling + review-threshold sweep
    serve.py        # FastAPI POST /classify
    cli.py          # typer entrypoint
  data/{generated,eval,splits}/
  models/           # gitignored checkpoints
  notebooks/
```

## Quick start

```bash
cd backend/ml
# 3.12 to match the runtime image (Dockerfile: python:3.12-slim).
python3.12 -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"

# Generate ~4000 synthetic examples (no LLM required)
ticketlens-ml generate --per-category 400 --out data/generated/corpus.parquet

# Stratified splits (seed tickets never enter train)
ticketlens-ml split \
  --corpus data/generated/corpus.parquet \
  --out-dir data/splits

# Train (needs [train] extras + a GPU or patience)
pip install -e ".[train]"
ticketlens-ml train \
  --train data/splits/train.parquet \
  --val data/splits/val.parquet \
  --model-name answerdotai/ModernBERT-base \
  --out models/modernbert-multitask

# Serve locally (loads checkpoint if present, else keyword stub)
ticketlens-ml serve --model-dir models/modernbert-multitask --port 8091
```

## Taxonomy sync

`taxonomy.py` must stay identical to
`backend/internal/domain/triage/model/category.go`. The test
`tests/test_taxonomy_sync.py` parses that Go file and fails on drift. It also
checks the third copy, `frontend/src/lib/api/labels.ts`, since nothing on the
TypeScript side validates it.

Run it from the backend with `make ml-test` (or `make test-all` for Go + Python).

## Service

The inference container is the `classifier` service in
`backend/deployments/docker-compose.yml`. Its build context is `../ml`, i.e.
this directory. Start it with `./dev.sh classifier` from `backend/`, or
`./start.sh classifier` from the repo root; it is opt-in, so the rest of the
infrastructure comes up without waiting on a Python image build.

The Go backend talks to it only when `CLASSIFIER_URL` is set — `dev.sh server`
sets it automatically when the container is running, and leaves it empty
otherwise so the in-process keyword stub is used.

## Evaluation set

`Tobi-Bueck/customer-support-tickets` is also synthetic; its value is that it
was **not** produced by our generator (out-of-distribution). Relabel English
rows into our 10 categories and keep them **test-only**. License: CC-BY-NC-4.0 —
state this in any report; do not claim commercial use.

```bash
ticketlens-ml prepare-eval --n 350 --out data/eval/ood_raw.parquet
# Then manually relabel into data/eval/ood_labeled.parquet
```

## Schema

Every training/eval row:

| column | notes |
|---|---|
| `subject` | short title |
| `body` | ticket body |
| `category` | one of the 10 taxonomy labels |
| `priority` | `low` / `normal` / `high` / `urgent` |
| `source` | `synthetic` / `ood_relabel` / `seed` |
| `lang` | always `en` |
| `generated_by` | generator id / model name |

Rows with `source=seed` are never used for training.
