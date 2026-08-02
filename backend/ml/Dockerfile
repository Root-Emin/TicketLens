FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml README.md ./
COPY src ./src

RUN pip install --no-cache-dir -e .

ENV MODEL_DIR=/models
EXPOSE 8091

CMD ["sh", "-c", "ticketlens-ml serve --host 0.0.0.0 --port 8091 --model-dir ${MODEL_DIR}"]
