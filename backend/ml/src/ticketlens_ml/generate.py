"""Synthetic ticket corpus generator.

Crosses persona × sector × tone × length to break the templatic patterns that
make synthetic data easy to overfit. No LLM required: templates are filled with
sampled slots. An optional LLM path can be wired later without changing the
output schema.

Schema (one row, dual labels):
  subject, body, category, priority, source, lang, generated_by
"""

from __future__ import annotations

import hashlib
import itertools
import random
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path

import pandas as pd

from ticketlens_ml.taxonomy import (
    CATEGORIES,
    PRIORITIES,
    assert_valid_category,
    assert_valid_priority,
)

GENERATED_BY = "template-v1"
SOURCE = "synthetic"
LANG = "en"

PERSONAS = [
    "solo merchant",
    "ops manager at a mid-size retailer",
    "accountant at a wholesale firm",
    "technical integrator at an agency",
    "customer-success lead at a brand",
    "founder of a new DTC shop",
]

SECTORS = [
    "fashion marketplace seller",
    "grocery e-commerce",
    "auto parts catalog",
    "cosmetics brand",
    "B2B wholesale",
    "home furniture store",
    "pet supplies shop",
    "bookstore",
]

TONES = ["calm", "frustrated", "formal", "terse", "polite"]

LENGTHS = ["short", "medium", "long"]

# Per-category subject/body templates. Placeholders: {persona}, {sector},
# {detail}, {system}, {amount}, {date}.
TEMPLATES: dict[str, list[tuple[str, str, str]]] = {
    # (subject, body, default_priority)
    "technical_issue": [
        (
            "Admin panel returns a 500 error",
            "As a {persona} in {sector}, the admin panel has been throwing a 500 error since {date}. Pages fail to load and we see a timeout in the console.",
            "high",
        ),
        (
            "Reports page will not load",
            "The sales reports screen stays blank for our {sector} account. We are a {persona} and cannot export anything today.",
            "high",
        ),
        (
            "Site is completely down",
            "Our storefront is down. Customers are affected and I cannot sell. This is urgent — please escalate.",
            "urgent",
        ),
        (
            "Dashboard charts empty / slow",
            "Charts on the home dashboard load empty or very slowly. Occasional timeout errors. {detail}",
            "normal",
        ),
        (
            "Excel export fails",
            "Trying to download a report as Excel ends with an error. Nothing is exported. {detail}",
            "high",
        ),
        (
            "Image upload fails on product page",
            "Uploading a product image fails every time. The file never sticks. We are a {persona}.",
            "high",
        ),
        (
            "Mobile app white screen",
            "The mobile app stays on a white screen at launch and throws an error. Cannot proceed.",
            "high",
        ),
        (
            "Panel extremely slow, frequent timeouts",
            "Operations in the panel are extremely slow today with frequent timeout errors. {detail}",
            "high",
        ),
    ],
    "integration": [
        (
            "{system} sync is not transferring products",
            "Our {system} marketplace integration has not been transferring products since {date}. The sync keeps failing with an API error.",
            "high",
        ),
        (
            "Stock synchronization halted halfway",
            "Stock counts are not updating via the {system} connector. The sync job halts halfway. {detail}",
            "high",
        ),
        (
            "Orders not pulled from marketplace",
            "The {system} integration is not pulling orders into the panel. The API returns an authorization error on our key.",
            "high",
        ),
        (
            "Shipping label won't print",
            "The cargo integration is not creating labels. The shipping service returns an error and we cannot ship.",
            "urgent",
        ),
        (
            "ERP records not transferring",
            "The {system} ERP integration is not transferring records to accounting. No sync since {date}.",
            "high",
        ),
        (
            "Webhook notifications missing",
            "Order webhook notifications have not reached our server for two days. Nothing in the integration logs.",
            "high",
        ),
        (
            "SDK update broke marketplace sync",
            "After we updated the SDK version, marketplace synchronization started failing. {detail}",
            "high",
        ),
        (
            "Hitting API rate limit errors",
            "Our integration service keeps hitting rate limit errors when calling the public API. Need guidance on backoff.",
            "normal",
        ),
    ],
    "payment_ops": [
        (
            "Yesterday's payout did not reach my account",
            "The payout that closed on {date} was not credited to my account ({amount}). I cannot receive payment.",
            "urgent",
        ),
        (
            "Refund not reflected on customer card",
            "The refund we approved ({amount}) has not been reflected on the customer's card. The refund process is stuck.",
            "high",
        ),
        (
            "Chargeback / dispute notice received",
            "We received a chargeback notice for a transaction of {amount}. Please advise on the dispute process.",
            "high",
        ),
        (
            "Settlement report incomplete",
            "Some transactions appear missing in this week's settlement report. Reconciliation does not match the panel.",
            "high",
        ),
        (
            "Payout amount calculated short",
            "This month's payout is lower than expected by about {amount}. There is a reconciliation difference.",
            "high",
        ),
        (
            "Money transfer delayed",
            "The payout normally arrived in two days; this time the deposit did not land. {detail}",
            "normal",
        ),
    ],
    "billing": [
        (
            "Cannot see my invoice for {date}",
            "Our invoice for {date} does not appear in the panel. We request a copy of the invoice.",
            "normal",
        ),
        (
            "Want to upgrade subscription package",
            "We would like to upgrade our current subscription package to a higher plan. What are the options?",
            "low",
        ),
        (
            "Commission rate on invoice looks wrong",
            "The commission rate in our contract differs from the rate reflected on the latest invoice ({amount}).",
            "normal",
        ),
        (
            "Cancel subscription request",
            "Please cancel our subscription at the end of the current billing period. Confirm the final invoice.",
            "low",
        ),
    ],
    "onboarding": [
        (
            "Stuck during setup",
            "We cannot progress through the setup steps for our {sector} account. Need support during onboarding.",
            "normal",
        ),
        (
            "How long does data migration take?",
            "How long does data migration from the old system take? What is the migration plan before go-live?",
            "normal",
        ),
        (
            "What is needed to go live?",
            "What activation steps must we complete before going live? We are a {persona} preparing launch.",
            "normal",
        ),
        (
            "Account setup checklist",
            "Looking for the account setup checklist and getting-started guide for a new {sector} tenant.",
            "low",
        ),
    ],
    "how_to": [
        (
            "How do I create a campaign?",
            "We want to learn how to create a seasonal discount campaign, step by step. Is there documentation?",
            "low",
        ),
        (
            "How do I do a bulk product upload?",
            "How can we upload our catalog in bulk? Is there a guide or training for this?",
            "low",
        ),
        (
            "Where is the user manual?",
            "Where is the panel's user manual or documentation located? We are onboarding a new teammate.",
            "low",
        ),
        (
            "How do I filter reports by date?",
            "How do I filter reports by date on the report screen? Looking for a short tutorial.",
            "low",
        ),
        (
            "How do I configure notification settings?",
            "How do I set notification preferences? Is there documentation to configure them?",
            "low",
        ),
    ],
    "account_access": [
        (
            "Cannot reset my password",
            "The password reset email is not arriving. I cannot log in to the panel. {detail}",
            "high",
        ),
        (
            "Cannot add a new user",
            "When we try to add a new user to the team we get a permission error and cannot assign a role.",
            "normal",
        ),
        (
            "Team has no access to the panel",
            "The accounting team has no access to the panel; permissions cannot be defined for their role.",
            "normal",
        ),
        (
            "Session keeps expiring / 2FA issue",
            "Sessions expire immediately after login and 2FA prompts fail. Need account access restored.",
            "high",
        ),
    ],
    "feature_request": [
        (
            "Feature request: bulk discounts",
            "We would like to submit a feature request for applying bulk discounts to product groups. {detail}",
            "low",
        ),
        (
            "Is multi-warehouse on the roadmap?",
            "Is multi-warehouse management on the roadmap? Can you share timing?",
            "low",
        ),
        (
            "Do you support multiple currencies?",
            "Do you support selling in different currencies? Filing this as a suggestion.",
            "low",
        ),
        (
            "Please add advanced inventory alerts",
            "It would be great if you could add advanced low-stock alerts per warehouse. Feature request.",
            "low",
        ),
    ],
    "sales": [
        (
            "Quote for an add-on module",
            "We want to purchase the reporting add-on module. Kindly send a price quote for a {sector} account.",
            "low",
        ),
        (
            "Demo request for enterprise edition",
            "We have a demo request for the enterprise edition. Could we meet at a convenient time?",
            "low",
        ),
        (
            "Please share a current price list",
            "We request a current price list and a quote for the new package options.",
            "low",
        ),
        (
            "Interested in buying for a second brand",
            "As a {persona}, we want to buy licenses for a second brand. Sales contact requested.",
            "low",
        ),
    ],
    "compliance": [
        (
            "Data deletion request under GDPR/KVKK",
            "Under KVKK/GDPR we request deletion of the data belonging to our company. Please confirm the process.",
            "normal",
        ),
        (
            "Privacy notice and contract copy",
            "We kindly request the privacy notice and a current copy of the contract for our records.",
            "low",
        ),
        (
            "Document request for annual audit",
            "We have a document request for our annual audit process. Please share the required compliance pack.",
            "normal",
        ),
        (
            "Audit trail export for compliance review",
            "Need an export of audit logs for a compliance review next week. {detail}",
            "normal",
        ),
    ],
}

SYSTEMS = [
    "Trendyol",
    "Hepsiburada",
    "N11",
    "Logo",
    "Mikro",
    "Netsis",
    "Aras Cargo",
    "virtual POS",
]

DETAILS = [
    "I can share screenshots if needed.",
    "This started after yesterday's deploy on our side.",
    "We already tried logging out and back in.",
    "No recent config changes on our end.",
    "Happens for every user on the account.",
    "Only one warehouse is affected.",
]

AMOUNTS = ["$120", "$1,450", "€890", "₺12,500", "$45.00"]
DATES = ["Monday", "last night", "February 12", "yesterday morning", "last Friday"]


@dataclass(frozen=True)
class Example:
    subject: str
    body: str
    category: str
    priority: str
    source: str = SOURCE
    lang: str = LANG
    generated_by: str = GENERATED_BY


def _fill(template: str, rng: random.Random, **extra: str) -> str:
    return template.format(
        persona=rng.choice(PERSONAS),
        sector=rng.choice(SECTORS),
        system=extra.get("system", rng.choice(SYSTEMS)),
        detail=rng.choice(DETAILS),
        amount=rng.choice(AMOUNTS),
        date=rng.choice(DATES),
    )


def _apply_tone(body: str, tone: str, rng: random.Random) -> str:
    prefixes = {
        "calm": "",
        "frustrated": "This is blocking us again. ",
        "formal": "Dear support team, ",
        "terse": "",
        "polite": "Hello — ",
    }
    suffixes = {
        "calm": "",
        "frustrated": " Please treat this as high priority.",
        "formal": " Thank you for your assistance.",
        "terse": "",
        "polite": " Thanks in advance.",
    }
    text = prefixes[tone] + body + suffixes[tone]
    if tone == "terse":
        # Drop a sentence-ish chunk to shorten.
        parts = text.split(". ")
        if len(parts) > 1:
            text = parts[0] + "."
    if tone == "frustrated" and rng.random() < 0.3:
        text = text.replace("please", "PLEASE")
    return text


def _apply_length(body: str, length: str, rng: random.Random) -> str:
    if length == "short":
        return body.split(". ")[0] + ("." if not body.endswith(".") else "")
    if length == "long":
        extras = [
            " We have attached order IDs in our internal tracker.",
            " Impact is limited to one channel for now.",
            " Happy to jump on a call if that helps.",
            f" Reference: {rng.randint(10000, 99999)}.",
        ]
        return body + "".join(rng.sample(extras, k=2))
    return body


def generate_examples(
    per_category: int = 400,
    seed: int = 20260801,
    priorities_override: dict[str, list[str]] | None = None,
) -> list[Example]:
    """Generate a balanced-ish corpus across categories with axis variation."""
    rng = random.Random(seed)
    out: list[Example] = []

    axes = list(itertools.product(PERSONAS[:4], SECTORS[:4], TONES, LENGTHS))
    rng.shuffle(axes)

    for category in CATEGORIES:
        templates = TEMPLATES[category]
        for i in range(per_category):
            subject_t, body_t, default_priority = templates[i % len(templates)]
            tone, length = axes[(i + category_index_offset(category)) % len(axes)][2:]
            row_rng = random.Random(
                seed + int(hashlib.md5(f"{category}:{i}".encode()).hexdigest()[:8], 16)
            )
            # Bias the row RNG's first draws toward the axis picks so templates
            # that mention persona/sector still vary systematically.
            row_rng.choice(PERSONAS)  # warm
            system = row_rng.choice(SYSTEMS)
            subject = _fill(subject_t, row_rng, system=system)
            body = _fill(body_t, row_rng, system=system)
            body = _apply_tone(body, tone, row_rng)
            body = _apply_length(body, length, row_rng)

            if priorities_override and category in priorities_override:
                priority = row_rng.choice(priorities_override[category])
            else:
                # Keep template default most of the time; occasionally nudge toward
                # normal so the priority head sees some natural mix.
                priority = default_priority
                if row_rng.random() < 0.15 and default_priority != "urgent":
                    priority = "normal"

            out.append(
                Example(
                    subject=subject.strip(),
                    body=body.strip(),
                    category=assert_valid_category(category),
                    priority=assert_valid_priority(priority),
                )
            )

    rng.shuffle(out)
    return out


def category_index_offset(category: str) -> int:
    return CATEGORIES.index(category) * 17


def examples_to_frame(examples: Iterable[Example]) -> pd.DataFrame:
    rows = [e.__dict__ for e in examples]
    df = pd.DataFrame(rows)
    # Stable column order matching the contract.
    return df[["subject", "body", "category", "priority", "source", "lang", "generated_by"]]


def write_corpus(path: Path, per_category: int = 400, seed: int = 20260801) -> pd.DataFrame:
    path.parent.mkdir(parents=True, exist_ok=True)
    df = examples_to_frame(generate_examples(per_category=per_category, seed=seed))
    if path.suffix == ".csv":
        df.to_csv(path, index=False)
    else:
        df.to_parquet(path, index=False)
    return df


def class_balance_report(df: pd.DataFrame) -> str:
    lines = ["# Class balance", "", "## Category"]
    for c in CATEGORIES:
        n = int((df["category"] == c).sum())
        lines.append(f"- `{c}`: {n}")
    lines.append("")
    lines.append("## Priority")
    for p in PRIORITIES:
        n = int((df["priority"] == p).sum())
        lines.append(f"- `{p}`: {n}")
    return "\n".join(lines) + "\n"
