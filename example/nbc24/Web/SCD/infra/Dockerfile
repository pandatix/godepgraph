FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt /tmp/requirements.txt

COPY flag.txt /flag.txt

RUN apt-get update && \
    apt-get install -y --no-install-recommends python3-venv && \
    rm -rf /var/lib/apt/lists/*

RUN python -m venv /app/venv

RUN /app/venv/bin/pip install --no-cache-dir -r /tmp/requirements.txt

RUN useradd -m flask

COPY app /app

RUN chown -R flask:flask /app

USER flask

EXPOSE 5000

CMD ["/app/venv/bin/python", "app.py"]
