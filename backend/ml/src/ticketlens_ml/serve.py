"""FastAPI inference service implementing the Go port.Classifier contract.

POST /classify body:  { "subject": "...", "body": "..." }
Response fields match ClassifyResult exactly:
  priority, priority_confidence, category, category_confidence,
  model_name, model_version, raw
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from fastapi import FastAPI
from pydantic import BaseModel, Field

from ticketlens_ml.evaluate import stub_predict
from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES

MODEL_NAME_STUB = "stub-py"
MODEL_VERSION_STUB = "v0"


class ClassifyRequest(BaseModel):
    subject: str = Field(default="")
    body: str = Field(default="")


class ClassifyResponse(BaseModel):
    priority: str
    priority_confidence: float
    category: str
    category_confidence: float
    model_name: str
    model_version: str
    raw: dict[str, Any] | None = None


class ClassifierBackend:
    def classify(self, subject: str, body: str) -> ClassifyResponse:
        raise NotImplementedError


class StubBackend(ClassifierBackend):
    def classify(self, subject: str, body: str) -> ClassifyResponse:
        cat, pri, c_conf, p_conf = stub_predict(subject, body)
        return ClassifyResponse(
            priority=pri,
            priority_confidence=p_conf,
            category=cat,
            category_confidence=c_conf,
            model_name=MODEL_NAME_STUB,
            model_version=MODEL_VERSION_STUB,
            raw={
                "engine": MODEL_NAME_STUB,
                "categories": CATEGORIES,
                "priorities": PRIORITIES,
            },
        )


class TorchBackend(ClassifierBackend):
    def __init__(self, model_dir: Path):
        import torch
        from torch import nn
        from transformers import AutoModel, AutoTokenizer

        meta = json.loads((model_dir / "meta.json").read_text(encoding="utf-8"))
        self.model_name = meta.get("config", {}).get("model_name", "encoder")
        self.model_version = model_dir.name
        self.tokenizer = AutoTokenizer.from_pretrained(model_dir)
        self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        self.temperatures = {
            "category": float(meta.get("temperature_category", 1.0)),
            "priority": float(meta.get("temperature_priority", 1.0)),
        }

        class MultiTaskModel(nn.Module):
            def __init__(self, base_name: str):
                super().__init__()
                self.encoder = AutoModel.from_pretrained(base_name)
                hidden = self.encoder.config.hidden_size
                self.cat_head = nn.Linear(hidden, len(CATEGORIES))
                self.pri_head = nn.Linear(hidden, len(PRIORITIES))

            def forward(self, input_ids, attention_mask):
                out = self.encoder(input_ids=input_ids, attention_mask=attention_mask)
                if hasattr(out, "pooler_output") and out.pooler_output is not None:
                    pooled = out.pooler_output
                else:
                    mask = attention_mask.unsqueeze(-1)
                    summed = (out.last_hidden_state * mask).sum(dim=1)
                    pooled = summed / mask.sum(dim=1).clamp(min=1)
                return self.cat_head(pooled), self.pri_head(pooled)

        base = meta.get("config", {}).get("model_name", "answerdotai/ModernBERT-base")
        self.model = MultiTaskModel(base).to(self.device)
        state = torch.load(model_dir / "model.pt", map_location=self.device)
        self.model.load_state_dict(state)
        self.model.eval()
        self._torch = torch

    def classify(self, subject: str, body: str) -> ClassifyResponse:
        torch = self._torch
        text = f"{subject}\n\n{body}"
        enc = self.tokenizer(
            text, truncation=True, max_length=256, padding=True, return_tensors="pt"
        )
        with torch.no_grad():
            cat_logits, pri_logits = self.model(
                enc["input_ids"].to(self.device),
                enc["attention_mask"].to(self.device),
            )
            cat_logits = cat_logits / self.temperatures["category"]
            pri_logits = pri_logits / self.temperatures["priority"]
            cat_probs = torch.softmax(cat_logits, dim=-1)[0]
            pri_probs = torch.softmax(pri_logits, dim=-1)[0]
            cat_idx = int(cat_probs.argmax().item())
            pri_idx = int(pri_probs.argmax().item())
        return ClassifyResponse(
            priority=PRIORITIES[pri_idx],
            priority_confidence=float(pri_probs[pri_idx].item()),
            category=CATEGORIES[cat_idx],
            category_confidence=float(cat_probs[cat_idx].item()),
            model_name=self.model_name,
            model_version=self.model_version,
            raw={
                "category_probs": {
                    c: float(cat_probs[i].item()) for i, c in enumerate(CATEGORIES)
                },
                "priority_probs": {
                    p: float(pri_probs[i].item()) for i, p in enumerate(PRIORITIES)
                },
            },
        )


def load_backend(model_dir: Path | None) -> ClassifierBackend:
    if model_dir is not None and (model_dir / "model.pt").exists():
        return TorchBackend(model_dir)
    return StubBackend()


def create_app(model_dir: Path | None = None) -> FastAPI:
    backend = load_backend(model_dir)
    app = FastAPI(title="TicketLens Classifier", version="0.1.0")

    @app.get("/healthz")
    def healthz():
        return {
            "status": "ok",
            "backend": type(backend).__name__,
            "categories": CATEGORIES,
            "priorities": PRIORITIES,
        }

    @app.post("/classify", response_model=ClassifyResponse)
    def classify(req: ClassifyRequest) -> ClassifyResponse:
        return backend.classify(req.subject or "", req.body or "")

    return app


def run(host: str = "0.0.0.0", port: int = 8091, model_dir: Path | None = None) -> None:
    import uvicorn

    uvicorn.run(create_app(model_dir), host=host, port=port)
