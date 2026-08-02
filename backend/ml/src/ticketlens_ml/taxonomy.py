"""Frozen taxonomy — must match backend/internal/domain/triage/model/category.go.

Do not invent labels here. If the set must change, update
backend/docs/taxonomy.md first, then category.go, then this file.
"""

from __future__ import annotations

CATEGORIES: list[str] = [
    "technical_issue",
    "integration",
    "payment_ops",
    "billing",
    "onboarding",
    "how_to",
    "account_access",
    "feature_request",
    "sales",
    "compliance",
]

PRIORITIES: list[str] = [
    "low",
    "normal",
    "high",
    "urgent",
]

CATEGORY_INDEX = {c: i for i, c in enumerate(CATEGORIES)}
PRIORITY_INDEX = {p: i for i, p in enumerate(PRIORITIES)}

# Short definitions used by the generator and eval relabel guide.
# Full boundary rules live in backend/docs/taxonomy.md.
CATEGORY_DEFS: dict[str, str] = {
    "technical_issue": "The platform itself is broken, erroring, or slow.",
    "integration": "A third-party link is failing (marketplace, ERP, cargo, API, webhook).",
    "payment_ops": "Money movement after a sale — payout, settlement, refund, reconciliation.",
    "billing": "What the customer pays us — invoice, subscription, our commission.",
    "onboarding": "Getting a new customer live — setup, migration, go-live.",
    "how_to": "Usage question; nothing is broken. Answer is documentation.",
    "account_access": "Login, password, permissions, users, roles.",
    "feature_request": "Something the product does not do yet.",
    "sales": "Pre-purchase commercial intent — quote, demo, price list.",
    "compliance": "Legal/regulatory — KVKK/GDPR, contract, audit, data deletion.",
}

PRIORITY_DEFS: dict[str, str] = {
    "urgent": "Business cannot operate — site down, cannot take orders or get paid.",
    "high": "Real breakage, business still runs (workaround or partial function).",
    "normal": "Standard request, no stoppage, no explicit urgency.",
    "low": "No breakage — questions, how-tos, suggestions, pre-sales.",
}


def assert_valid_category(label: str) -> str:
    if label not in CATEGORY_INDEX:
        raise ValueError(f"unknown category {label!r}; expected one of {CATEGORIES}")
    return label


def assert_valid_priority(label: str) -> str:
    if label not in PRIORITY_INDEX:
        raise ValueError(f"unknown priority {label!r}; expected one of {PRIORITIES}")
    return label
