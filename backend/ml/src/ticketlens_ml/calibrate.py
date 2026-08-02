"""Temperature scaling and review-threshold selection.

Fits one temperature per head on the validation set, then sweeps the review
threshold to balance queue length against mistaken auto-routing. The chosen
threshold should be written back to CLASSIFIER_REVIEW_THRESHOLD.
"""

from __future__ import annotations

import json
from pathlib import Path

import numpy as np

from ticketlens_ml.evaluate import reliability_bins
from ticketlens_ml.split import load_table
from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES


def fit_temperature(logits: np.ndarray, labels: np.ndarray) -> float:
    """Grid-search a scalar temperature minimizing NLL on one head."""
    # logits: [N, C]
    best_t, best_nll = 1.0, float("inf")
    for t in np.linspace(0.5, 5.0, 46):
        scaled = logits / t
        # numerically stable softmax NLL
        shifted = scaled - scaled.max(axis=1, keepdims=True)
        exp = np.exp(shifted)
        probs = exp / exp.sum(axis=1, keepdims=True)
        nll = -np.log(np.clip(probs[np.arange(len(labels)), labels], 1e-12, 1.0)).mean()
        if nll < best_nll:
            best_nll, best_t = float(nll), float(t)
    return best_t


def apply_temperature(logits: np.ndarray, temperature: float) -> np.ndarray:
    scaled = logits / temperature
    shifted = scaled - scaled.max(axis=1, keepdims=True)
    exp = np.exp(shifted)
    return exp / exp.sum(axis=1, keepdims=True)


def sweep_threshold(
    cat_conf: np.ndarray,
    pri_conf: np.ndarray,
    cat_correct: np.ndarray,
    pri_correct: np.ndarray,
    thresholds: np.ndarray | None = None,
) -> list[dict]:
    """For each threshold, report review rate and error rate among auto-accepted."""
    if thresholds is None:
        thresholds = np.linspace(0.40, 0.90, 26)
    rows = []
    for t in thresholds:
        auto = (cat_conf >= t) & (pri_conf >= t)
        review_rate = float((~auto).mean())
        if auto.any():
            # An auto-accept is wrong if either head is wrong.
            err = float((~(cat_correct[auto] & pri_correct[auto])).mean())
        else:
            err = None
        rows.append(
            {
                "threshold": float(t),
                "review_rate": review_rate,
                "auto_error_rate": err,
                "n_auto": int(auto.sum()),
                "n_review": int((~auto).sum()),
            }
        )
    return rows


def choose_threshold(sweep: list[dict], *, max_auto_error: float = 0.08) -> float:
    """Pick the lowest threshold whose auto-accept error stays under the budget."""
    candidates = [
        r for r in sweep if r["auto_error_rate"] is not None and r["auto_error_rate"] <= max_auto_error
    ]
    if not candidates:
        return 0.60
    # Prefer lower review rate among safe candidates.
    best = min(candidates, key=lambda r: (r["review_rate"], -r["threshold"]))
    return float(best["threshold"])


def calibrate_from_arrays(
    cat_logits: np.ndarray,
    pri_logits: np.ndarray,
    cat_labels: np.ndarray,
    pri_labels: np.ndarray,
    out_path: Path,
) -> dict:
    t_cat = fit_temperature(cat_logits, cat_labels)
    t_pri = fit_temperature(pri_logits, pri_labels)
    cat_probs = apply_temperature(cat_logits, t_cat)
    pri_probs = apply_temperature(pri_logits, t_pri)

    cat_pred = cat_probs.argmax(axis=1)
    pri_pred = pri_probs.argmax(axis=1)
    cat_conf = cat_probs.max(axis=1)
    pri_conf = pri_probs.max(axis=1)
    cat_ok = cat_pred == cat_labels
    pri_ok = pri_pred == pri_labels

    sweep = sweep_threshold(cat_conf, pri_conf, cat_ok, pri_ok)
    threshold = choose_threshold(sweep)

    report = {
        "temperature_category": t_cat,
        "temperature_priority": t_pri,
        "recommended_review_threshold": threshold,
        "default_was": 0.60,
        "threshold_sweep": sweep,
        "reliability_category": reliability_bins(cat_conf, cat_ok.astype(float)),
        "reliability_priority": reliability_bins(pri_conf, pri_ok.astype(float)),
        "labels": {"categories": CATEGORIES, "priorities": PRIORITIES},
        "note": (
            "Write recommended_review_threshold into CLASSIFIER_REVIEW_THRESHOLD. "
            "Report category and priority metrics separately; priority lag is expected."
        ),
    }
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(report, indent=2), encoding="utf-8")
    return report


def calibrate_stub_proxy(val_path: Path, out_path: Path) -> dict:
    """Build a calibration report from the keyword stub's hit-derived confidences.

    Useful before a trained checkpoint exists: demonstrates the sweep pipeline
    on the same validation rows the model will later use.
    """
    from ticketlens_ml.evaluate import stub_predict

    df = load_table(val_path)
    cat_logits = []
    pri_logits = []
    cat_labels = []
    pri_labels = []

    for _, row in df.iterrows():
        cat, pri, c_conf, p_conf = stub_predict(row["subject"], row["body"])
        # Fabricate peaked logits from the stub's confidence so the temperature
        # fitter has something to work with without a neural model.
        c_vec = np.full(len(CATEGORIES), (1.0 - c_conf) / max(1, len(CATEGORIES) - 1))
        c_vec[CATEGORIES.index(cat)] = c_conf
        p_vec = np.full(len(PRIORITIES), (1.0 - p_conf) / max(1, len(PRIORITIES) - 1))
        p_vec[PRIORITIES.index(pri)] = p_conf
        # Convert probs → fake logits via log
        cat_logits.append(np.log(np.clip(c_vec, 1e-6, 1.0)))
        pri_logits.append(np.log(np.clip(p_vec, 1e-6, 1.0)))
        cat_labels.append(CATEGORIES.index(row["category"]))
        pri_labels.append(PRIORITIES.index(row["priority"]))

    return calibrate_from_arrays(
        np.asarray(cat_logits),
        np.asarray(pri_logits),
        np.asarray(cat_labels),
        np.asarray(pri_labels),
        out_path,
    )
