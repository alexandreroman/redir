# Redir

[![Build](https://github.com/alexandreroman/redir/actions/workflows/build.yml/badge.svg)](https://github.com/alexandreroman/redir/actions/workflows/build.yml)

A lightweight URL redirection service written in Go, with click tracking powered by Redis and automatic QR code generation.

## Features

- **URL Redirection** — define short slugs that redirect to full URLs via a simple TOML config file.
- **Click Tracking** — every redirect increments a counter in Redis, queryable via the `/stats` endpoint.
- **QR Code Generation** — append `.png` to any slug (e.g. `/github.png`) to get a QR code image pointing to that redirect.
- **Social Media Previews** — automatically fetches Open Graph meta tags from target URLs and serves them to social media crawlers (LinkedIn, Facebook, Twitter, Slack, Discord, etc.) for rich link previews.
- **Health Check** — `/healthz` endpoint for load balancers and orchestrators.

## Getting Started

### Prerequisites

- Go 1.25+
- Redis
- Docker & Docker Compose (optional)

### Configuration

Redirects are defined in a `redirects.toml` file:

```toml
[[redirects]]
slug = "github"
url = "https://github.com/alexandreroman"

[[redirects]]
slug = "linkedin"
url = "https://linkedin.com/in/alexandre-roman"
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `REDIR_CONFIG` | `redirects.toml` | Path to the TOML configuration file |
| `REDIR_ADDR` | `:4000` | Address the HTTP server listens on |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(empty)* | Redis password |
| `REDIS_TLS` | *(empty)* | Set to `true` to enable TLS for Redis connections |
| `REDIR_USER` | *(empty)* | Username for Basic Auth on `/stats` |
| `REDIR_PASSWORD` | *(empty)* | Password for Basic Auth on `/stats` |

### Run Locally

```bash
make build
make run
```

### Run with Docker Compose

```bash
docker compose up --build
```

This starts both the `redir` server and a Redis instance. The service is available at `http://localhost:4000`.

## API

| Endpoint | Description |
|---|---|
| `GET /` | Index page |
| `GET /<slug>` | Redirects to the configured URL (HTTP 302) |
| `GET /<slug>.png` | Returns a QR code image for the redirect |
| `GET /stats` | Returns click counts per slug as JSON (protected by Basic Auth when `REDIR_USER` and `REDIR_PASSWORD` are set) |
| `GET /healthz` | Health check (`OK`) |
| `GET /robots.txt` | Allows all crawlers |

### Example: `/stats` Response

```json
{
  "meta": { "version": 1, "date": "2026-03-10" },
  "slugs": { "github": 42, "linkedin": 7 }
}
```

## Development

```bash
make build        # Compile the binary
make test         # Run tests
make lint         # Run golangci-lint
make clean        # Remove the binary
make docker-build # Build the Docker image
```

## License

This project is licensed under the [Apache License 2.0](LICENSE).
