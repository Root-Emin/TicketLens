# External corpus — Kaggle customer support ticket log

Copied from `~/Downloads` on 2026-08-03 as `customer_support_tickets.csv`.
Kaggle's "Customer Support Ticket Dataset" — a CRM-export-shaped table
(customer demographics, product, channel, SLA timestamps, CSAT).

Named `kaggle-support-tickets-8k` here to keep it distinct from the unrelated
`customer-support-tickets/` corpus in the sibling directory.

## ⚠ License — UNVERIFIED

The Kaggle page was not checked when this file was copied here. **Verify before
this data reaches a report, a model artefact, or a commit.**

## ⚠ This is synthetic data wearing a CRM costume

Read this section before planning any use. The columns look like a real
helpdesk export; the contents are generated, and several fields are noise.

**`Ticket Description` is templated.** All 8,469 rows contain the literal
unfilled placeholder `{product_purchased}` — the generator never substituted
it. 392 descriptions are exact duplicates of another row. Any normalizer must
handle the placeholder explicitly; training on the literal string teaches a
token that cannot occur in production.

**`Ticket Priority` is random.** Cross-tabulated against `Ticket Type` it is
flat to within a point:

| Ticket Type | Critical | High | Low | Medium |
|---|---:|---:|---:|---:|
| Billing inquiry | 25.7 | 23.4 | 24.4 | 26.6 |
| Cancellation request | 25.0 | 23.5 | 24.4 | 27.1 |
| Product inquiry | 24.6 | 24.3 | 24.3 | 26.9 |
| Refund request | 25.3 | 25.6 | 25.1 | 24.0 |
| Technical issue | 25.1 | 26.2 | 23.6 | 25.0 |

A label independent of the text carries no learnable signal. **Do not train the
priority head on this column, and do not evaluate against it** — a model
scoring 25% on it is at chance, which would be misread as a real result. This
is the trap to avoid given `urgent` currently scores F1 = 0.0.

Note the level names differ from ours anyway (`Critical` vs our `urgent`), and
`Resolution` text is generated filler ("Case maybe show recently my computer
follow.").

**PII-shaped columns are fake but still PII-shaped.** `Customer Name`,
`Customer Email` (`@example.com`), `Customer Age`, `Customer Gender`. Drop them
in any normalizer rather than carrying them downstream — they are useless as
features and they make the artefact look like it holds personal data.

## `raw/` is immutable

`raw/` is `chmod 444` on purpose. Never edit in place; every transformation
writes a new file elsewhere so the copy stays byte-reproducible.

| File | Rows | Columns |
|---|---:|---:|
| `customer_support_tickets.csv` | 8,469 | 17 |

## Checksum (sha256)

```
b06a9cde84da65db388bd964d75f88ee1eed96607cf75d0c35f09c3f11bf8bea  customer_support_tickets.csv
```

## Schema and missingness

`Ticket ID, Customer Name, Customer Email, Customer Age, Customer Gender,
Product Purchased, Date of Purchase, Ticket Type, Ticket Subject,
Ticket Description, Ticket Status, Resolution, Ticket Priority, Ticket Channel,
First Response Time, Time to Resolution, Customer Satisfaction Rating`

Nulls are structural, driven by `Ticket Status`:

- `Resolution`, `Time to Resolution`, `Customer Satisfaction Rating` — 5,700
  null (every row not `Closed`).
- `First Response Time` — 2,819 null (every `Open` row).

Filtering to non-null CSAT silently filters to closed tickets only.

## Label mismatch — read before using

**Category.** 5 `Ticket Type` values (`Refund request`, `Technical issue`,
`Cancellation request`, `Product inquiry`, `Billing inquiry`) against our 10
categories — coarser than our taxonomy, so it cannot supervise it directly. The
16 `Ticket Subject` values are finer and closer to intent shape
(`Software bug`, `Network problem`, `Account access`, `Payment issue`,
`Delivery problem`, `Data loss`, …), but `Ticket Subject` is drawn independently
of `Ticket Description` in the same way `Ticket Priority` is — verify the
subject/body correspondence before trusting it as a label.

## What this corpus is actually good for

Given the above, its value is **not** as labelled training data. It is the only
source here with realistic *operational* structure — channel mix, status
lifecycle, SLA timestamps, CSAT — so it is useful for exercising the triage
workspace's UI and aggregate views with plausibly-shaped rows. Treat it as
fixture data, not ground truth.
