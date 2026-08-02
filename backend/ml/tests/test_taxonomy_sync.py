"""Fail if the taxonomy drifts between its copies.

The label set is declared in three places that cannot import each other:

  * Go     — backend/internal/domain/triage/model/{category,ticket}.go (source of truth)
  * Python — ml/src/ticketlens_ml/taxonomy.py                          (this package)
  * TS     — frontend/src/lib/api/labels.ts                            (filter lists)

This module is the only executable check that they agree, so it deliberately
reaches across all three. Prose lives in backend/docs/taxonomy.md.
"""

from __future__ import annotations

import re
from pathlib import Path

import pytest

from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES

# This file is backend/ml/tests/, so parents[2] is backend/ — the ML workspace
# lives inside the backend, not next to it. Resolve the Go sources from there
# instead of prefixing "backend" again. The repo root is one level further up,
# which is where the frontend sits.
BACKEND_ROOT = Path(__file__).resolve().parents[2]
REPO_ROOT = BACKEND_ROOT.parent
MODEL_DIR = BACKEND_ROOT / "internal" / "domain" / "triage" / "model"
CATEGORY_GO = MODEL_DIR / "category.go"
TICKET_GO = MODEL_DIR / "ticket.go"
LABELS_TS = REPO_ROOT / "frontend" / "src" / "lib" / "api" / "labels.ts"


def test_go_sources_are_where_we_think_they_are():
    """Guard the guard: a path typo must fail loudly, not skip the sync check."""
    assert CATEGORY_GO.is_file(), f"category.go not found at {CATEGORY_GO}"
    assert TICKET_GO.is_file(), f"ticket.go not found at {TICKET_GO}"


def _ts_string_array(text: str, name: str) -> list[str]:
    """Extract the string literals from `export const NAME: T[] = [...]`."""
    block = re.search(
        rf"export const {name}\s*:[^=]*=\s*\[(.*?)\]", text, re.DOTALL
    )
    assert block, f"{name} not found in {LABELS_TS.name}"
    return re.findall(r'"([a-z_]+)"', block.group(1))


def _parse_string_consts(path: Path, assign_pattern: str) -> list[str]:
    text = path.read_text(encoding="utf-8")
    return re.findall(assign_pattern, text)


def test_categories_match_go_all_categories():
    # Pull the AllCategories slice literals in order.
    text = CATEGORY_GO.read_text(encoding="utf-8")
    block = re.search(r"var AllCategories = \[\]Category\{(.*?)\}$", text, re.DOTALL | re.MULTILINE)
    assert block, "AllCategories not found in category.go"
    names = re.findall(r"Category(\w+)", block.group(1))
    # Map Go exported names back to snake values via the const block.
    const_vals = dict(
        re.findall(r"Category(\w+)\s+Category\s*=\s*\"([a-z_]+)\"", text)
    )
    go_labels = [const_vals[n] for n in names]
    assert go_labels == CATEGORIES, (go_labels, CATEGORIES)


def test_priorities_match_go():
    text = TICKET_GO.read_text(encoding="utf-8")
    vals = re.findall(
        r"TicketPriority\w+\s+TicketPriority\s*=\s*\"([a-z]+)\"", text
    )
    assert vals == PRIORITIES, (vals, PRIORITIES)


def test_category_count_is_ten():
    assert len(CATEGORIES) == 10
    assert len(set(CATEGORIES)) == 10


# ─── Frontend copy ────────────────────────────────────────────────────────────
#
# labels.ts is the fourth copy of the taxonomy and the only one nothing else
# validated. TypeScript catches a missing CATEGORY_LABELS entry, but a stale
# ALL_CATEGORIES array fails silently — the category just disappears from the
# queue filter. These tests close that hole.


@pytest.mark.skipif(
    not LABELS_TS.is_file(), reason="frontend not checked out next to backend/"
)
def test_frontend_categories_match():
    labels = _ts_string_array(LABELS_TS.read_text(encoding="utf-8"), "ALL_CATEGORIES")
    assert labels == CATEGORIES, (labels, CATEGORIES)


@pytest.mark.skipif(
    not LABELS_TS.is_file(), reason="frontend not checked out next to backend/"
)
def test_frontend_priorities_match():
    labels = _ts_string_array(LABELS_TS.read_text(encoding="utf-8"), "ALL_PRIORITIES")
    assert labels == PRIORITIES, (labels, PRIORITIES)


@pytest.mark.skipif(
    not LABELS_TS.is_file(), reason="frontend not checked out next to backend/"
)
def test_frontend_category_labels_cover_every_category():
    """A CATEGORY_LABELS entry missing here means a blank chip in the UI."""
    text = LABELS_TS.read_text(encoding="utf-8")
    block = re.search(
        r"export const CATEGORY_LABELS\s*:[^=]*=\s*\{(.*?)\}", text, re.DOTALL
    )
    assert block, "CATEGORY_LABELS not found in labels.ts"
    keys = re.findall(r"^\s*([a-z_]+)\s*:", block.group(1), re.MULTILINE)
    assert sorted(keys) == sorted(CATEGORIES), (sorted(keys), sorted(CATEGORIES))
