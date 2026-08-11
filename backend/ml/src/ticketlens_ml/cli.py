"""Typer CLI entrypoint: ticketlens-ml <command>."""

from __future__ import annotations

from pathlib import Path

import typer
from rich import print

app = typer.Typer(no_args_is_help=True, add_completion=False)


@app.command()
def generate(
    per_category: int = typer.Option(400, help="Examples per category"),
    out: Path = typer.Option(Path("data/generated/corpus.parquet")),
    seed: int = 20260801,
):
    """Generate a synthetic dual-label corpus (no LLM required)."""
    from ticketlens_ml.generate import class_balance_report, write_corpus

    df = write_corpus(out, per_category=per_category, seed=seed)
    bal = out.with_suffix(".balance.md")
    bal.write_text(class_balance_report(df), encoding="utf-8")
    print(f"[green]Wrote {len(df)} rows → {out}[/green]")
    print(f"Balance report → {bal}")


@app.command()
def split(
    corpus: Path = typer.Option(..., exists=True),
    out_dir: Path = typer.Option(Path("data/splits")),
    val_size: float = 0.1,
    test_size: float = 0.1,
    seed: int = 20260801,
):
    """Stratified train/val/test. Seed-sourced rows never enter train."""
    from ticketlens_ml.split import write_splits

    paths = write_splits(
        corpus, out_dir, val_size=val_size, test_size=test_size, seed=seed
    )
    for k, p in paths.items():
        print(f"{k}: {p}")


@app.command("prepare-eval")
def prepare_eval_cmd(
    n: int = 350,
    out: Path = typer.Option(Path("data/eval/ood_raw.csv")),
    seed: int = 20260801,
):
    """Sample HF OOD rows for manual relabeling (test-only)."""
    from ticketlens_ml.prepare_eval import prepare_eval

    path = prepare_eval(out, n=n, seed=seed)
    print(f"[green]Wrote {path}[/green] — fill category/priority per relabel_guide.md")


@app.command("external-english")
def external_english_cmd(
    raw_dir: Path = typer.Option(
        Path("data/external/customer-support-tickets/raw"), exists=True
    ),
    out_dir: Path = typer.Option(
        Path("data/external/customer-support-tickets/normalized")
    ),
    dedupe: bool = typer.Option(True, help="Drop rows whose body was already seen"),
):
    """Filter the external Kaggle corpus down to its English rows."""
    from ticketlens_ml.external_english import filter_english

    path, report = filter_english(raw_dir, out_dir, dedupe=dedupe)

    for f in report.files:
        langs = ", ".join(f"{k}={v:,}" for k, v in sorted(f.languages.items()))
        note = "" if f.english else "  [yellow](no English rows)[/yellow]"
        print(f"  {f.name}: {f.rows:,} rows → kept {f.kept:,}{note}")
        print(f"    languages: {langs}")

    print(
        f"[green]Wrote {report.total_kept:,} English rows → {path}[/green] "
        f"({report.total_duplicates:,} duplicates dropped)"
    )


@app.command()
def evaluate(
    test: Path = typer.Option(..., exists=True),
    out: Path = typer.Option(Path("data/splits/stub_baseline.json")),
):
    """Run the keyword stub as a baseline on a test split."""
    from ticketlens_ml.evaluate import evaluate_stub

    report = evaluate_stub(test, out)
    print(
        f"category macro-F1={report['category']['macro_f1']:.3f}  "
        f"priority macro-F1={report['priority']['macro_f1']:.3f}"
    )
    print(f"Wrote {out}")


@app.command()
def train(
    train: Path = typer.Option(..., exists=True),
    val: Path = typer.Option(..., exists=True),
    out: Path = typer.Option(Path("models/modernbert-multitask")),
    model_name: str = "answerdotai/ModernBERT-base",
    epochs: int = 3,
    batch_size: int = 16,
):
    """Fine-tune shared encoder + two heads."""
    from ticketlens_ml.train import TrainConfig
    from ticketlens_ml.train import train as run_train

    cfg = TrainConfig(model_name=model_name, epochs=epochs, batch_size=batch_size)
    path = run_train(train, val, out, cfg)
    print(f"[green]Checkpoint → {path}[/green]")


@app.command()
def calibrate(
    val: Path = typer.Option(..., exists=True),
    out: Path = typer.Option(Path("models/calibration.json")),
):
    """Temperature + threshold sweep using the stub proxy (or extend for torch)."""
    from ticketlens_ml.calibrate import calibrate_stub_proxy

    report = calibrate_stub_proxy(val, out)
    print(
        f"recommended CLASSIFIER_REVIEW_THRESHOLD="
        f"{report['recommended_review_threshold']}"
    )
    print(f"Wrote {out}")


@app.command()
def serve(
    host: str = "0.0.0.0",
    port: int = 8091,
    model_dir: Path | None = typer.Option(None, help="Checkpoint dir; stub if absent"),
):
    """Run the FastAPI /classify service."""
    from ticketlens_ml.serve import run

    run(host=host, port=port, model_dir=model_dir)


@app.command("list-candidates")
def list_candidates():
    from ticketlens_ml.train import list_candidates

    for c in list_candidates():
        print(f"[bold]{c['name']}[/bold] — {c['why']}")


if __name__ == "__main__":
    app()
