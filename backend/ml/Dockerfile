# TicketLens classifier image. Build context is backend/ml.
#
# Two stages so the compiler toolchain does not reach the runtime: pip needs
# build-essential for the source distributions in the dependency tree, and
# leaving it in the final image adds ~200 MB and a much larger attack surface
# for no runtime benefit.
#
# The trained checkpoint is copied in rather than mounted, so an image tag names
# both the code and the model it serves and a rollback returns to the model the
# previous release was validated against. With no checkpoint present the service
# still starts and serves the keyword stub (load_backend in serve.py) — useful
# for CI, never what a production tag should contain. Check which one is live:
#
#   curl -s localhost:8091/healthz   ->  {"backend": "TorchBackend", ...}

FROM python:3.12-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# A venv rather than the system site-packages: it gives the runtime stage one
# self-contained directory to copy, with no need to replay the install there.
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

WORKDIR /app

# torch comes from the CPU wheel index first, before anything can pull the
# default build. The default carries bundled CUDA libraries — roughly 2 GB that
# an inference container on a CPU host will never execute. The later install
# sees the requirement as already satisfied and leaves this wheel in place.
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir --index-url https://download.pytorch.org/whl/cpu torch

COPY pyproject.toml README.md ./
COPY src ./src

RUN pip install --no-cache-dir ".[serve]"

FROM python:3.12-slim

# Model version is a build input so the label can record which checkpoint is
# baked in; `docker inspect` then answers "what model is this tag serving?"
ARG MODEL_VERSION="none"
LABEL org.opencontainers.image.title="ticketlens-classifier" \
      org.opencontainers.image.description="TicketLens multi-task ticket classifier" \
      dev.ticketlens.model-version="${MODEL_VERSION}"

RUN useradd --create-home --uid 1001 appuser

COPY --from=builder /opt/venv /opt/venv

# Last, and on its own layer: the checkpoint is the largest and most frequently
# replaced input, so every layer above it stays cached when only the model
# changes. models/ is gitignored — CI fetches the checkpoint here before build.
COPY --chown=appuser:appuser models /models

ENV PATH="/opt/venv/bin:$PATH" \
    MODEL_DIR=/models \
    PYTHONUNBUFFERED=1

USER appuser

EXPOSE 8091

HEALTHCHECK --interval=15s --timeout=5s --start-period=40s --retries=5 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8091/healthz')" || exit 1

CMD ["sh", "-c", "ticketlens-ml serve --host 0.0.0.0 --port 8091 --model-dir ${MODEL_DIR}"]
