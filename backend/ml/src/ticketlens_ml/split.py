"""Stratified train/val/test splits.

Rows with source=seed are never placed in train — they remain available for
smoke tests only. Class imbalance is documented alongside the split files.
"""

from __future__ import annotations

from pathlib import Path

import pandas as pd
from sklearn.model_selection import train_test_split

from ticketlens_ml.generate import class_balance_report
from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES


def load_table(path: Path) -> pd.DataFrame:
    if path.suffix == ".csv":
        return pd.read_csv(path)
    return pd.read_parquet(path)


def save_table(df: pd.DataFrame, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.suffix == ".csv":
        df.to_csv(path, index=False)
    else:
        df.to_parquet(path, index=False)


def stratified_split(
    df: pd.DataFrame,
    *,
    val_size: float = 0.1,
    test_size: float = 0.1,
    seed: int = 20260801,
) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]:
    """Split on the joint (category, priority) key so both heads stay balanced."""
    trainable = df[df["source"] != "seed"].copy()
    held_seed = df[df["source"] == "seed"].copy()

    trainable["strat"] = trainable["category"] + "|" + trainable["priority"]
    # Collapse rare joint cells so stratify does not fail.
    counts = trainable["strat"].value_counts()
    rare = counts[counts < 3].index
    trainable.loc[trainable["strat"].isin(rare), "strat"] = trainable.loc[
        trainable["strat"].isin(rare), "category"
    ]

    train_val, test = train_test_split(
        trainable,
        test_size=test_size,
        random_state=seed,
        stratify=trainable["strat"],
    )
    relative_val = val_size / (1.0 - test_size)
    train, val = train_test_split(
        train_val,
        test_size=relative_val,
        random_state=seed,
        stratify=train_val["strat"],
    )

    for part in (train, val, test):
        part.drop(columns=["strat"], inplace=True)

    # Seed rows go only to a side file; never train/val/test.
    return train.reset_index(drop=True), val.reset_index(drop=True), test.reset_index(drop=True), held_seed


def write_splits(
    corpus_path: Path,
    out_dir: Path,
    *,
    val_size: float = 0.1,
    test_size: float = 0.1,
    seed: int = 20260801,
) -> dict[str, Path]:
    df = load_table(corpus_path)
    train, val, test, held_seed = stratified_split(
        df, val_size=val_size, test_size=test_size, seed=seed
    )

    out_dir.mkdir(parents=True, exist_ok=True)
    paths = {
        "train": out_dir / "train.parquet",
        "val": out_dir / "val.parquet",
        "test": out_dir / "test.parquet",
        "seed_holdout": out_dir / "seed_holdout.parquet",
        "balance": out_dir / "CLASS_BALANCE.md",
    }
    save_table(train, paths["train"])
    save_table(val, paths["val"])
    save_table(test, paths["test"])
    if len(held_seed):
        save_table(held_seed, paths["seed_holdout"])

    report = [
        class_balance_report(df),
        "",
        f"Train rows: {len(train)}",
        f"Val rows: {len(val)}",
        f"Test rows: {len(test)}",
        f"Seed holdout (never train): {len(held_seed)}",
        "",
        "Known skew (from seed demo corpus, for context): integration and",
        "technical_issue dominate (~16 each); six rare classes sit at 3.",
        "Synthetic generation targets a flatter category prior; priority still",
        "skews toward high/low because of template defaults — report priority",
        "metrics separately.",
        "",
        f"Categories: {', '.join(CATEGORIES)}",
        f"Priorities: {', '.join(PRIORITIES)}",
    ]
    paths["balance"].write_text("\n".join(report) + "\n", encoding="utf-8")
    return paths
