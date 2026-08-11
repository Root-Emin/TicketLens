# External corpus — CFPB Consumer Complaint Database

Copied from `~/Downloads/complaints.csv` on 2026-08-03. A full export of the
US Consumer Financial Protection Bureau's public complaint database.

This is the largest corpus in the project by three orders of magnitude, and the
**only one written by real people** — everything else under `data/external/` is
generated. That is its value and also the source of every caveat below.

## License — public domain

CFPB is a US federal agency and the database is published as open government
data; the records are not copyrightable. This is the one corpus here **without**
an unverified-licence warning, unlike `customer-support-tickets/` (CC-BY-NC-4.0
mirror), `bitext-support-intents/` and `kaggle-support-tickets-8k/`.

Two obligations still apply, and they are CFPB's own publishing terms rather
than licence terms:

- Complaints are **unverified allegations**. CFPB does not confirm the facts of
  a complaint before publishing it. Never present a narrative as an established
  fact about the named company.
- Company names are real and unredacted (`Pauls Auto Sales & Service Inc`,
  `TRANSUNION INTERMEDIATE HOLDINGS, INC.`). Do not surface them in demo UI,
  screenshots, or reports — a synthetic-looking ticket naming a real firm
  alongside an allegation is a defamation shape, not a data-quality problem.

## `raw/` is immutable

`raw/` is `chmod 444` on purpose. Never edit in place; every transformation
writes a new file elsewhere so the download stays byte-reproducible.

| File | Size | Rows | Unique Complaint IDs |
|---|---:|---:|---:|
| `complaints.csv` | 9.05 GB | 16,906,905 | 16,900,994 |

The copy was verified against the source with sha256 after `cp`.

## Checksum (sha256)

```
61ffa6e3a44f70b1680e1e679f4112ece88b9c25e65ed13e24ee798ae80221dd  complaints.csv
```

## Schema

`Date received, Product, Sub-product, Issue, Sub-issue,
Consumer complaint narrative, Company public response, Company, State,
ZIP code, Tags, Submitted via, Date sent to company,
Company response to consumer, Timely response?, Complaint ID`

## Only 22.7% of rows carry text

**3,830,002 of 16,906,905 rows** have a non-empty
`Consumer complaint narrative` (~3.91 GB of text, mean 1,021 chars). The other
13.1M rows are structured metadata with no body — a narrative is only published
when the consumer opts in.

Any pipeline that reads this file must filter on narrative presence first.
Sampling `complaints.csv` uniformly yields ~77% empty bodies.

## Verified clean — two things that look wrong but are not

Both were checked rather than assumed, because the file's shape invites the
wrong conclusion:

**It is not a concatenation of overlapping exports.** 16.9M rows is far above
CFPB's historical volume and the row count grows steeply by year (2023: 1.29M,
2024: 2.73M, 2025: 5.44M, 2026: 4.22M), which looks like stacked downloads.
It is not: **zero duplicate Complaint IDs, zero repeated header rows.** The
growth is real — it tracks the credit-reporting complaint surge visible in the
Product distribution below. No deduplication pass is needed.

**`Date received` has two formats, in two clean blocks.** Rows 1–106,878 are
ISO-8601 with time (`2025-07-15T12:57:20.000Z`); rows 106,879–16,906,905 are
date-only (`2023-10-26`). Exactly two contiguous runs, no interleaving — a
recent-additions block ahead of the bulk export. Harmless, but **any date
parsing must handle both formats**; a single `strptime` pattern silently fails
on one block or the other.

5,911 rows have a blank `Complaint ID`.

## Redaction — the text is pre-scrubbed, and it shows

CFPB redacts PII before publishing, leaving literal artefacts in the text.
Measured on a 300,000-narrative sample:

| Pattern | Share | Example |
|---|---:|---|
| `XXXX` (names, companies, account numbers) | 75.5% | `Received a phone call from XXXX with United` |
| `XX/XX/XXXX`, `XX/XX/year>` (dates) | 30.2% | `my first payment was due on XX/XX/XXXX` |
| `{$1234.00}` (money) | 21.6% | `I owed over {$3000.00} on a {$1500.00} loan` |

Three quarters of narratives contain `XXXX`. Training on this raw teaches a
token that cannot occur in a production ticket — the same failure mode as
`{product_purchased}` in `kaggle-support-tickets-8k/` and `{{Order Number}}` in
`bitext-support-intents/`, but at much higher prevalence. A normalizer must
decide to strip, mask-token, or synthesise these before any training use.

Note `XX/XX/year>` — a malformed redaction leaking part of an HTML entity.
Pattern-matching on `XX/XX/XXXX` alone will miss it.

## Label mismatch — read before using

**Category.** 21 `Product` values and 173 `Issue` values, all US consumer
finance. These are regulatory product lines, not support intents, and they do
not map onto `taxonomy.py` — the overlap with our categories is close to nil
outside `compliance` and arguably `billing`. Do not write a dictionary.

The distribution is severely skewed. Of narrative-bearing rows:

| Product | Narrative rows |
|---|---:|
| Credit reporting or other personal consumer reports | 1,671,558 |
| Credit reporting, credit repair services, or other personal consumer reports | 807,499 |
| Debt collection | 438,766 |
| Checking or savings account | 185,448 |
| Mortgage | 145,784 |
| Credit card | 125,651 |
| *(15 more, tapering to 16 rows for `Virtual currency`)* | |

The top two labels are **the same product line under two naming conventions**
(CFPB renamed it), and together they are 65% of all narratives. Any sampling
must merge those two and stratify, or the corpus is effectively
"credit reporting disputes" with a long tail of noise. `Issue` has the same
problem: `Problem with a company's investigation into an existing problem` and
`Problem with a credit reporting company's investigation into an existing
problem` are a renamed pair, 599,558 rows combined.

**Priority.** **There is no priority column**, as with
`bitext-support-intents/`. `Timely response?` is about the *company's* SLA, not
consumer urgency, and `Tags` (`Servicemember`, `Older American`) is a
demographic flag — neither is a priority label. `urgent` must still be derived
from text per `taxonomy.py`.

**Domain.** This is the deepest caveat. TicketLens triages product-support
tickets; these are financial regulatory complaints. Register, length (mean
~1,000 chars vs single-sentence Bitext instructions), and subject matter all
differ. It is not a drop-in training source.

## What this corpus is actually good for

- **Real human writing.** The only non-synthetic text available. Useful for
  checking whether a model trained on generated corpora degrades on genuine
  prose — an axis none of the other three can test.
- **The `compliance` category**, which is thinly covered elsewhere.
- **Deriving `urgent`.** Real financial distress ("repossessed my vehicle
  before any payment was due") is closer to the `urgent` definition in
  `taxonomy.py` than anything in the synthetic corpora, where `urgent` is the
  class scoring F1 = 0.0. Requires manual labelling — there is no column to
  lift.

## Eval / train separation

`customer-support-tickets/` is already designated the out-of-distribution eval
source. This corpus is a *different* and much harsher OOD axis. Decide its role
— eval-only, or a labelled training slice — and freeze it **before** drawing
any rows. Splitting after the fact is not trustworthy.
