"""Multi-task fine-tuning: shared encoder + category head + priority head.

Requires the optional [train] extras (torch, transformers). Class-weighted
cross-entropy counters the known imbalance; loss weights between heads are
configurable because priority is the harder problem.
"""

from __future__ import annotations

import json
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path

import numpy as np
import pandas as pd

from ticketlens_ml.split import load_table
from ticketlens_ml.taxonomy import (
    CATEGORIES,
    CATEGORY_INDEX,
    PRIORITIES,
    PRIORITY_INDEX,
)


@dataclass
class TrainConfig:
    model_name: str = "answerdotai/ModernBERT-base"
    max_length: int = 256
    batch_size: int = 16
    epochs: int = 3
    lr: float = 2e-5
    weight_decay: float = 0.01
    category_loss_weight: float = 1.0
    priority_loss_weight: float = 1.0
    seed: int = 20260801


def _require_torch():
    try:
        import torch
        from torch import nn
        from torch.utils.data import DataLoader, Dataset
        from transformers import (
            AutoModel,
            AutoTokenizer,
            get_linear_schedule_with_warmup,
        )
    except ImportError as e:
        raise SystemExit(
            "Training extras missing. Install with: pip install -e '.[train]'"
        ) from e
    return torch, nn, DataLoader, Dataset, AutoModel, AutoTokenizer, get_linear_schedule_with_warmup


def class_weights(labels: list[str], vocab: list[str]):
    import torch

    counts = Counter(labels)
    total = sum(counts.values()) or 1
    weights = []
    for lab in vocab:
        c = counts.get(lab, 1)
        weights.append(total / (len(vocab) * c))
    return torch.tensor(weights, dtype=torch.float)


def train(
    train_path: Path,
    val_path: Path,
    out_dir: Path,
    cfg: TrainConfig | None = None,
) -> Path:
    (
        torch,
        nn,
        DataLoader,
        Dataset,
        AutoModel,
        AutoTokenizer,
        get_linear_schedule_with_warmup,
    ) = _require_torch()

    cfg = cfg or TrainConfig()
    torch.manual_seed(cfg.seed)
    np.random.seed(cfg.seed)

    train_df = load_table(train_path)
    val_df = load_table(val_path)
    tokenizer = AutoTokenizer.from_pretrained(cfg.model_name)

    class TicketDataset(Dataset):
        def __init__(self, df: pd.DataFrame):
            self.subjects = df["subject"].tolist()
            self.bodies = df["body"].tolist()
            self.cat = [CATEGORY_INDEX[c] for c in df["category"]]
            self.pri = [PRIORITY_INDEX[p] for p in df["priority"]]

        def __len__(self):
            return len(self.subjects)

        def __getitem__(self, idx):
            text = f"{self.subjects[idx]}\n\n{self.bodies[idx]}"
            enc = tokenizer(
                text,
                truncation=True,
                max_length=cfg.max_length,
                padding="max_length",
                return_tensors="pt",
            )
            return {
                "input_ids": enc["input_ids"].squeeze(0),
                "attention_mask": enc["attention_mask"].squeeze(0),
                "category": torch.tensor(self.cat[idx], dtype=torch.long),
                "priority": torch.tensor(self.pri[idx], dtype=torch.long),
            }

    class MultiTaskModel(nn.Module):
        def __init__(self):
            super().__init__()
            self.encoder = AutoModel.from_pretrained(cfg.model_name)
            hidden = self.encoder.config.hidden_size
            self.cat_head = nn.Linear(hidden, len(CATEGORIES))
            self.pri_head = nn.Linear(hidden, len(PRIORITIES))

        def forward(self, input_ids, attention_mask):
            out = self.encoder(input_ids=input_ids, attention_mask=attention_mask)
            # Prefer pooler_output when present; otherwise mean-pool last hidden.
            if hasattr(out, "pooler_output") and out.pooler_output is not None:
                pooled = out.pooler_output
            else:
                mask = attention_mask.unsqueeze(-1)
                summed = (out.last_hidden_state * mask).sum(dim=1)
                pooled = summed / mask.sum(dim=1).clamp(min=1)
            return self.cat_head(pooled), self.pri_head(pooled)

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    model = MultiTaskModel().to(device)

    cat_w = class_weights(train_df["category"].tolist(), CATEGORIES).to(device)
    pri_w = class_weights(train_df["priority"].tolist(), PRIORITIES).to(device)
    cat_crit = nn.CrossEntropyLoss(weight=cat_w)
    pri_crit = nn.CrossEntropyLoss(weight=pri_w)

    train_loader = DataLoader(TicketDataset(train_df), batch_size=cfg.batch_size, shuffle=True)
    val_loader = DataLoader(TicketDataset(val_df), batch_size=cfg.batch_size)

    optim = torch.optim.AdamW(model.parameters(), lr=cfg.lr, weight_decay=cfg.weight_decay)
    total_steps = max(1, len(train_loader) * cfg.epochs)
    sched = get_linear_schedule_with_warmup(optim, int(0.1 * total_steps), total_steps)

    history = []
    best_val = -1.0
    out_dir.mkdir(parents=True, exist_ok=True)

    for epoch in range(cfg.epochs):
        model.train()
        running = 0.0
        for batch in train_loader:
            optim.zero_grad()
            cat_logits, pri_logits = model(
                batch["input_ids"].to(device), batch["attention_mask"].to(device)
            )
            loss = cfg.category_loss_weight * cat_crit(
                cat_logits, batch["category"].to(device)
            ) + cfg.priority_loss_weight * pri_crit(pri_logits, batch["priority"].to(device))
            loss.backward()
            optim.step()
            sched.step()
            running += float(loss.item())

        val_macro = _val_macro_f1(model, val_loader, device)
        history.append(
            {
                "epoch": epoch + 1,
                "train_loss": running / max(1, len(train_loader)),
                "val_macro_f1_mean": val_macro,
            }
        )
        if val_macro > best_val:
            best_val = val_macro
            torch.save(model.state_dict(), out_dir / "model.pt")

    meta = {
        "config": asdict(cfg),
        "categories": CATEGORIES,
        "priorities": PRIORITIES,
        "history": history,
        "best_val_macro_f1_mean": best_val,
    }
    (out_dir / "meta.json").write_text(json.dumps(meta, indent=2), encoding="utf-8")
    tokenizer.save_pretrained(out_dir)
    return out_dir


def _val_macro_f1(model, loader, device) -> float:
    import torch
    from sklearn.metrics import f1_score

    model.eval()
    y_cat_t, y_cat_p, y_pri_t, y_pri_p = [], [], [], []
    with torch.no_grad():
        for batch in loader:
            cat_logits, pri_logits = model(
                batch["input_ids"].to(device), batch["attention_mask"].to(device)
            )
            y_cat_p.extend(cat_logits.argmax(-1).cpu().tolist())
            y_pri_p.extend(pri_logits.argmax(-1).cpu().tolist())
            y_cat_t.extend(batch["category"].tolist())
            y_pri_t.extend(batch["priority"].tolist())
    cat_f1 = f1_score(y_cat_t, y_cat_p, average="macro", zero_division=0)
    pri_f1 = f1_score(y_pri_t, y_pri_p, average="macro", zero_division=0)
    return float((cat_f1 + pri_f1) / 2)


def list_candidates() -> list[dict[str, str]]:
    """Encoder candidates for the English-only multi-task setup."""
    return [
        {
            "name": "answerdotai/ModernBERT-base",
            "why": "Strong modern encoder baseline; ~150M params; Colab T4 friendly.",
        },
        {
            "name": "microsoft/deberta-v3-base",
            "why": "Often best accuracy; heavier than ModernBERT.",
        },
        {
            "name": "distilbert/distilroberta-base",
            "why": "Speed/latency edge for CPU inference budgets.",
        },
    ]
