# External corpus — customer support tickets

Downloaded from Kaggle as `archive.zip` and extracted on 2026-08-03.
Same dataset family as `Tobi-Bueck/customer-support-tickets` on HuggingFace,
which `ticketlens_ml.prepare_eval` was already written against — the column
schema (`subject, body, answer, type, queue, priority, language, tag_*`)
matches exactly.

## ⚠ License — UNVERIFIED

The HuggingFace mirror is **CC-BY-NC-4.0** (non-commercial). The Kaggle page
was not checked when these files were copied here. **Verify the Kaggle license
before any of this data reaches a report, a model artefact, or a commit.**

If it is CC-BY-NC-4.0:
- cite the source in any report,
- do not claim commercial use,
- do not describe it as "real customer data" — it is synthetic.

## `raw/` is immutable

Files in `raw/` are `chmod 444` on purpose. Never edit them in place. Every
transformation writes a **new** file elsewhere (`data/eval/`, `data/splits/`,
or a normalized parquet) so the download stays byte-reproducible.

| File | Rows | English rows |
|---|---:|---:|
| `aa_dataset-tickets-multi-lang-5-2-50-version.csv` | 28,587 | 16,338 |
| `dataset-tickets-multi-lang-4-20k.csv` | 20,000 | 11,923 |
| `dataset-tickets-multi-lang3-4k.csv` | 4,000 | 1,391 |
| `dataset-tickets-german_normalized.csv` | 2,125 | 0 |
| `dataset-tickets-german_normalized_50_5_2.csv` | 13,178 | 0 |

The three multi-lang files overlap: `4-20k` shares 4,595 English bodies with
`5-2-50-version`. Deduplicated union of English bodies: **25,056**.

The two `german_normalized` files are 100% German and carry no English rows.
They are kept only so `raw/` mirrors the download completely; the project
standardised on English in Phase 0.

## Checksums (sha256)

```
f187c090e59581c2bbf3aa1377c8db4dd647464ecf2ae51bf8966e42e0ed6bc0  aa_dataset-tickets-multi-lang-5-2-50-version.csv
80b63df950c0b831971381b9c6ee064633f49095654c0fd8cb8818390cbc2ee8  dataset-tickets-german_normalized.csv
22580337aed864d8c0485f16a7fe683d48d8adfc3af0cd1a6fe1e240f728735f  dataset-tickets-german_normalized_50_5_2.csv
9be3bf810584fe01e8e83383e83dfd33f4c3910938ecad03ef151da79d8f0635  dataset-tickets-multi-lang-4-20k.csv
9aae7120cf459fc27561febe29c7757c6d222bfebff50e8baa868991e57b87d1  dataset-tickets-multi-lang3-4k.csv
```

## Label mismatch — read before using

Source labels do **not** map onto the frozen taxonomy in
`src/ticketlens_ml/taxonomy.py`. Do not write a fixed dictionary.

**Category.** The source has 10 `queue` values, we have 10 categories — this is
a coincidence, not an alignment. Source queues are org-chart shaped
(`Technical Support`, `IT Support`, `Product Support`, `Human Resources`); our
categories are intent shaped (`payment_ops` vs `billing`, `how_to`,
`onboarding`, `compliance`). `queue` and `type` are hints only.

**Priority.** The source has three levels (`low` / `medium` / `high`). We have
four. **There is no `urgent` in the source** — which is exactly the class that
scores F1 = 0.0 today. `urgent` has to be derived from the text using the
definition in `taxonomy.py` (business cannot operate: site down, cannot take
orders or get paid), not lifted from a column.

**Data hygiene.** 2,607 of the English rows have an empty `subject`; `body` is
populated on all of them.

## Eval / train separation

This dataset was the project's only planned out-of-distribution evaluation
source. If rows from it are used for training, an evaluation slice drawn from
the same source is no longer out-of-distribution.

Split it **once, before any other use**, and freeze the eval slice. Splitting
after the fact is not trustworthy.
