# CLAUDE.md

## Build & Test

- Use `make build` to compile the project.
- Use `make test` to run tests.
- Use `make run` to build and run the server.
- Use `make lint` to run the linter.
- Use `docker-compose up --build` to compile and run the app with its dependencies.

## Code Formatting

- Run `go fmt ./...` before staging Go files with `git add` to ensure consistent formatting.

## Reserved Slugs

When adding a new endpoint (e.g. `robots.txt`, `healthz`, `stats`), remember to:

1. Add the slug to the `reservedSlugs` map in `config.go` so it cannot be used as a redirect.
2. Update the reserved slugs list in `TestLoadConfig_ReservedSlug` in `config_test.go`.
