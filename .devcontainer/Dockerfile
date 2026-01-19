FROM mcr.microsoft.com/devcontainers/go:1.25

RUN apt-get update \
    && apt-get install -y --no-install-recommends sqlite3 \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

ENV PATH="$PATH:/go/bin"
