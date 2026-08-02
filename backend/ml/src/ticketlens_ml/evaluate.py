"""Head-wise evaluation and stub baseline comparison.

Never report a single blended accuracy. Category and priority are separate
problems; priority macro-F1 lagging is expected.
"""

from __future__ import annotations

import json
from collections.abc import Callable
from dataclasses import asdict, dataclass
from pathlib import Path

import numpy as np
import pandas as pd
from sklearn.metrics import (
    classification_report,
    confusion_matrix,
    f1_score,
)

from ticketlens_ml.split import load_table
from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES


@dataclass
class HeadMetrics:
    name: str
    macro_f1: float
    micro_f1: float
    per_class_f1: dict[str, float]
    support: dict[str, int]


def text_of(row: pd.Series) -> str:
    return f"{row['subject']}\n\n{row['body']}"


# ---------------------------------------------------------------------------
# Keyword stub — English mirror of the Go stub classifier, for baseline.
# ---------------------------------------------------------------------------

_CATEGORY_KEYWORDS: dict[str, list[str]] = {
    "integration": [
        "integration", "marketplace", "webhook", "sdk", "api", "sync",
        "erp", "cargo", "shipping", "trendyol", "hepsiburada", "n11",
    ],
    "payment_ops": [
        "payout", "settlement", "reconciliation", "refund", "chargeback",
        "not credited", "dispute",
    ],
    "billing": ["invoice", "subscription", "commission", "billing", "upgrade", "plan"],
    "technical_issue": [
        "error", "500", "timeout", "down", "won't open", "not loading", "crash", "bug",
    ],
    "onboarding": ["setup", "onboarding", "migration", "go-live", "go live", "activation"],
    "how_to": ["how to", "how do", "documentation", "guide", "training", "manual", "tutorial"],
    "account_access": ["password", "log in", "login", "permission", "role", "access", "2fa"],
    "feature_request": ["feature request", "roadmap", "suggestion", "do you support"],
    "sales": ["quote", "demo", "price list", "purchase", "buy", "sales"],
    "compliance": ["gdpr", "kvkk", "audit", "privacy notice", "data deletion", "contract"],
}

_URGENT = ["site is down", "site down", "cannot ship", "cannot sell", "customers are affected", "urgent", "production down"]
_HIGH = ["error", "500", "not transferring", "not working", "failed", "refund", "chargeback"]
_LOW = ["how to", "how do", "documentation", "guide", "training", "feature request", "demo", "quote"]


def _score(text: str, keywords: list[str]) -> int:
    t = text.lower()
    return sum(1 for k in keywords if k in t)


def stub_predict(subject: str, body: str) -> tuple[str, str, float, float]:
    text = f"{subject} {body}"
    subject_l = subject.lower()
    body_l = body.lower()

    best_cat, best_hits = "how_to", 0
    for cat in CATEGORIES:
        hits = 2 * _score(subject_l, _CATEGORY_KEYWORDS.get(cat, [])) + _score(
            body_l, _CATEGORY_KEYWORDS.get(cat, [])
        )
        if hits > best_hits:
            best_cat, best_hits = cat, hits

    if any(k in text.lower() for k in _URGENT):
        priority = "urgent"
        p_hits = 2
    elif any(k in text.lower() for k in _HIGH):
        priority = "high"
        p_hits = 1
    elif any(k in text.lower() for k in _LOW):
        priority = "low"
        p_hits = 1
    else:
        priority = "normal"
        p_hits = 0

    def conf(hits: int) -> float:
        return min(0.95, 0.50 + hits * 0.15)

    return best_cat, priority, conf(best_hits), conf(p_hits)


def metrics_for_head(
    y_true: list[str],
    y_pred: list[str],
    labels: list[str],
    name: str,
) -> HeadMetrics:
    per_class = f1_score(y_true, y_pred, labels=labels, average=None, zero_division=0)
    support = {lab: int(sum(1 for y in y_true if y == lab)) for lab in labels}
    return HeadMetrics(
        name=name,
        macro_f1=float(f1_score(y_true, y_pred, labels=labels, average="macro", zero_division=0)),
        micro_f1=float(f1_score(y_true, y_pred, labels=labels, average="micro", zero_division=0)),
        # strict=True: per_class comes back one score per label, so a length
        # mismatch means the label set and the scores disagree — silently
        # truncating there would produce a metrics dict that looks fine.
        per_class_f1={lab: float(v) for lab, v in zip(labels, per_class, strict=True)},
        support=support,
    )


Predictor = Callable[[str, str], tuple[str, str]]


def evaluate_frame(
    df: pd.DataFrame,
    predict: Predictor,
) -> dict:
    y_cat_true, y_cat_pred = [], []
    y_pri_true, y_pri_pred = [], []
    for _, row in df.iterrows():
        cat, pri = predict(row["subject"], row["body"])
        y_cat_true.append(row["category"])
        y_cat_pred.append(cat)
        y_pri_true.append(row["priority"])
        y_pri_pred.append(pri)

    cat_m = metrics_for_head(y_cat_true, y_cat_pred, CATEGORIES, "category")
    pri_m = metrics_for_head(y_pri_true, y_pri_pred, PRIORITIES, "priority")

    return {
        "n": len(df),
        "category": asdict(cat_m),
        "priority": asdict(pri_m),
        "category_confusion": confusion_matrix(
            y_cat_true, y_cat_pred, labels=CATEGORIES
        ).tolist(),
        "priority_confusion": confusion_matrix(
            y_pri_true, y_pri_pred, labels=PRIORITIES
        ).tolist(),
        "category_report": classification_report(
            y_cat_true, y_cat_pred, labels=CATEGORIES, zero_division=0
        ),
        "priority_report": classification_report(
            y_pri_true, y_pri_pred, labels=PRIORITIES, zero_division=0
        ),
    }


def evaluate_stub(test_path: Path, out_path: Path | None = None) -> dict:
    df = load_table(test_path)

    def predict(subject: str, body: str) -> tuple[str, str]:
        cat, pri, _, _ = stub_predict(subject, body)
        return cat, pri

    report = evaluate_frame(df, predict)
    report["model"] = "stub-keyword-v0"
    if out_path is not None:
        out_path.parent.mkdir(parents=True, exist_ok=True)
        # Drop the free-text sklearn reports from JSON; keep them as .txt siblings.
        json_safe = {k: v for k, v in report.items() if not k.endswith("_report")}
        out_path.write_text(json.dumps(json_safe, indent=2), encoding="utf-8")
        (out_path.with_suffix(".category.txt")).write_text(report["category_report"], encoding="utf-8")
        (out_path.with_suffix(".priority.txt")).write_text(report["priority_report"], encoding="utf-8")
    return report


def reliability_bins(
    confidences: np.ndarray,
    correct: np.ndarray,
    n_bins: int = 10,
) -> dict:
    """Simple calibration curve inputs for a head."""
    edges = np.linspace(0.0, 1.0, n_bins + 1)
    bins = []
    for i in range(n_bins):
        mask = (confidences >= edges[i]) & (confidences < edges[i + 1] + (1e-9 if i == n_bins - 1 else 0))
        if not mask.any():
            bins.append({"mean_conf": float((edges[i] + edges[i + 1]) / 2), "acc": None, "n": 0})
            continue
        bins.append(
            {
                "mean_conf": float(confidences[mask].mean()),
                "acc": float(correct[mask].mean()),
                "n": int(mask.sum()),
            }
        )
    return {"bins": bins}
