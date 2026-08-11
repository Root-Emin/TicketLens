# External corpus — Bitext customer support intents

Copied from `~/Downloads` on 2026-08-03. Same dataset family as
`bitext/Bitext-customer-support-llm-chatbot-training-dataset` on HuggingFace.

This is the **sample** release (27K rows). Bitext also sells a much larger
commercial version; do not conflate the two when citing.

## ⚠ License — UNVERIFIED

The HuggingFace mirror is published as **CC-BY-4.0**. The copy placed here
arrived as a loose file with no license text alongside it, so the provenance
chain is unverified in the same way the `customer-support-tickets/` download
is. **Confirm the license before this data reaches a report, a model artefact,
or a commit.**

If it is CC-BY-4.0: attribution to Bitext is required, commercial use is
permitted (unlike the CC-BY-NC-4.0 corpus next door). Do not describe it as
real customer data — it is generated.

## `raw/` is immutable

`raw/` is `chmod 444` on purpose. Never edit in place; every transformation
writes a new file elsewhere so the copy stays byte-reproducible.

| File | Rows | Columns |
|---|---:|---:|
| `Bitext_Sample_Customer_Support_Training_Dataset_27K_responses-v11.csv` | 26,872 | 5 |

Row count is 26,872 parsed CSV records (91,336 physical lines — `response`
spans multiple lines).

## Checksum (sha256)

```
6f81102b0100b97b8468eb04368033a23206bf1fde9d53500d5806ec1001a434  Bitext_Sample_Customer_Support_Training_Dataset_27K_responses-v11.csv
```

## Schema

`flags, instruction, category, intent, response`

- **`instruction`** — the user utterance. This is the only field shaped like a
  ticket body, and it is *short* (a single sentence), unlike the multi-paragraph
  bodies in `customer-support-tickets/`. Length distribution differs enough from
  our generated corpus that it should be checked before mixing.
- **`response`** — the ideal agent reply. Not an input feature for either of our
  heads; it is chatbot training data. 26,870 of 26,872 are unique.
- **`flags`** — language-variation tags (`B` colloquial, `Q` typos/QA noise,
  `Z` and others). Letters observed: B C E I K L M N P Q S V W Z. 394 distinct
  combinations. Useful as a robustness axis: the same intent appears in clean
  and deliberately noisy phrasings.
- **`category` / `intent`** — 11 categories, 27 intents, strictly nested
  (each intent belongs to exactly one category).

## Label mismatch — read before using

Do not write a fixed dictionary onto `taxonomy.py`.

**Category.** 11 source categories vs our 10. They are e-commerce/order-flow
shaped (`ORDER`, `SHIPPING`, `DELIVERY`, `REFUND`, `CANCEL`, `INVOICE`,
`SUBSCRIPTION`, `FEEDBACK`, `ACCOUNT`, `PAYMENT`, `CONTACT`). Some are close to
ours — `PAYMENT` ≈ `payment_ops`, `INVOICE` ≈ `billing` — but the source has no
equivalent of `how_to`, `onboarding`, or `compliance`, and splits across
`ORDER`/`CANCEL`/`REFUND` where we would not. `intent` is the finer and more
useful signal; treat both as hints.

**Priority.** **There is no priority column at all.** This corpus cannot
supervise the priority head. That is a hard limitation, not something to work
around — `urgent` in particular must still come from the text per the
definition in `taxonomy.py`.

**Placeholders.** 6,670 of 26,872 instructions (25%) contain `{{...}}` slots —
`{{Order Number}}`, `{{Customer Support Hours}}`, `{{Website URL}}`. Left
in `raw/`; a normalizer must decide to fill, strip, or drop them. Leaving the
literal braces in training text teaches the model a token that never appears in
production tickets.

**Class balance.** Intents are near-uniform (~1,000 each by construction), so
categories are unbalanced only because they hold different intent counts
(`ACCOUNT` 5,986 vs `CANCEL` 950). This is synthetic uniformity, not a prior
worth learning — do not read it as a real-world distribution.

## Eval / train separation

`customer-support-tickets/` is already spoken for as the out-of-distribution
eval source. If this corpus is also used for eval, it is a *second, different*
OOD axis (short utterances, intent labels) — decide and freeze that role before
using any of it for training.
