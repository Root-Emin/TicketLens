# OOD evaluation relabel guide

Source: `Tobi-Bueck/customer-support-tickets` (English sample).

License: **CC-BY-NC-4.0** — non-commercial; cite in reports. This set is also
synthetic; its value is that it was **not** produced by our generator
(out-of-distribution). Do **not** call it "real customer data".

This set is **test-only**. Do not train on it.

Fill `category` and `priority` using the frozen definitions in
`backend/docs/taxonomy.md`.

## Allowed categories

- `technical_issue`
- `integration`
- `payment_ops`
- `billing`
- `onboarding`
- `how_to`
- `account_access`
- `feature_request`
- `sales`
- `compliance`

## Allowed priorities

- `urgent` — business cannot operate
- `high` — real breakage, business still runs
- `normal` — default
- `low` — no breakage (questions, how-tos, suggestions)

## Process

```bash
ticketlens-ml prepare-eval --n 350 --out data/eval/ood_raw.csv
# edit category/priority columns
# save standard schema as data/eval/ood_labeled.parquet
```

`sample_labeled.csv` in this folder is a tiny hand-labeled fixture for pipeline
smoke tests — not a substitute for the full OOD set.
