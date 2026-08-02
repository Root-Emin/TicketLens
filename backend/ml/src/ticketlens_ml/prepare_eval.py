"""Prepare an out-of-distribution evaluation sample.

Downloads English rows from Tobi-Bueck/customer-support-tickets (also synthetic;
value is that it was not produced by our generator), samples N rows, and writes
a CSV ready for manual relabeling into our 10-category taxonomy.

License: CC-BY-NC-4.0 — cite in reports; do not claim commercial use or call
the set "real customer data".
"""

from __future__ import annotations

from pathlib import Path

import pandas as pd

from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES

HF_DATASET = "Tobi-Bueck/customer-support-tickets"
LICENSE_NOTE = "CC-BY-NC-4.0 — non-commercial; out-of-distribution synthetic eval only."


def prepare_eval(
    out_path: Path,
    n: int = 350,
    seed: int = 20260801,
) -> Path:
    try:
        from datasets import load_dataset
    except ImportError as e:
        raise SystemExit("datasets package required: pip install datasets") from e

    ds = load_dataset(HF_DATASET, split="train")
    df = ds.to_pandas()

    # Prefer English when a language column exists.
    lang_cols = [c for c in df.columns if c.lower() in {"language", "lang", "locale"}]
    if lang_cols:
        col = lang_cols[0]
        english = df[df[col].astype(str).str.lower().str.startswith("en")]
        if len(english) >= n:
            df = english

    # Heuristic text columns.
    subject_col = next(
        (c for c in df.columns if c.lower() in {"subject", "title", "headline"}), None
    )
    body_col = next(
        (
            c
            for c in df.columns
            if c.lower() in {"body", "text", "message", "description", "content"}
        ),
        None,
    )
    if body_col is None:
        # Fall back to the longest string column.
        str_cols = [c for c in df.columns if df[c].dtype == object]
        body_col = max(str_cols, key=lambda c: df[c].astype(str).str.len().mean())

    sample = df.sample(n=min(n, len(df)), random_state=seed).reset_index(drop=True)
    out = pd.DataFrame(
        {
            "subject": (
                sample[subject_col].astype(str)
                if subject_col
                else sample[body_col].astype(str).str.slice(0, 80)
            ),
            "body": sample[body_col].astype(str),
            "category": "",  # fill during manual relabel
            "priority": "",  # fill during manual relabel
            "source": "ood_relabel",
            "lang": "en",
            "generated_by": HF_DATASET,
            "orig_type": sample["type"].astype(str) if "type" in sample.columns else "",
            "orig_queue": sample["queue"].astype(str) if "queue" in sample.columns else "",
            "orig_priority": (
                sample["priority"].astype(str) if "priority" in sample.columns else ""
            ),
            "license": LICENSE_NOTE,
        }
    )

    out_path.parent.mkdir(parents=True, exist_ok=True)
    if out_path.suffix == ".parquet":
        out.to_parquet(out_path, index=False)
    else:
        out.to_csv(out_path, index=False)

    guide = out_path.with_name("relabel_guide.md")
    if not guide.exists():
        guide.write_text(_guide_text(), encoding="utf-8")
    return out_path


def _guide_text() -> str:
    cats = "\n".join(f"- `{c}`" for c in CATEGORIES)
    pris = "\n".join(f"- `{p}`" for p in PRIORITIES)
    return f"""# OOD evaluation relabel guide

Source: `{HF_DATASET}` (English sample). License: {LICENSE_NOTE}.

This set is **test-only**. Do not train on it.

Fill `category` and `priority` using the frozen definitions in
`backend/docs/taxonomy.md`. Allowed values:

## Categories
{cats}

## Priorities
{pris}

## Mapping tips from the source columns
- Source `queue` / `type` are hints only — remap into our ten labels by reading
  the text, not by a fixed dictionary.
- Source `priority` vocabularies differ; re-decide from business impact (site
  down → urgent; how-to → low).
- When unsure between neighbours (`payment_ops` vs `billing`, `integration` vs
  `technical_issue`), apply the boundary rules in taxonomy.md.

After labeling, save as `data/eval/ood_labeled.parquet` (or `.csv`) with the
standard schema columns only:
`subject, body, category, priority, source, lang, generated_by`.
"""
