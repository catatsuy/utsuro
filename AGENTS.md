# Repository Guidelines

## Project Structure & Module Organization
- `cmd/utsuro/main.go`: application entrypoint.
- `internal/cli/`: CLI flags and startup wiring.
- `internal/server/`: memcached text protocol handling and TCP server logic.
- `internal/cache/`: in-memory cache and eviction/list implementations.
- `benchmark/`: separate Go module for comparative benchmarks (`benchmark/go.mod`).
- `.github/workflows/`: CI, security scans, and release automation.

Keep new production code under `internal/` unless it is a true public package. Place tests next to the code they verify (`*_test.go`).

## Build, Test, and Development Commands
- `make build`: builds `bin/utsuro`.
- `make run`: runs the server locally (`cmd/utsuro`).
- `make test`: runs all root-module tests (`go test ./... -count=1 -timeout=120s`).
- `make cover`: generates `coverage.out` and prints per-function coverage.
- `make fmt`: formats Go code (`go fmt ./...`).
- `make vet`: runs static checks (`go vet ./...`).
- `make tidy`: syncs `go.mod`/`go.sum`.
- `cd benchmark && go test ./... -count=1 -timeout=120s`: benchmark module tests used by CI.

## Coding Style & Naming Conventions
- Follow standard Go style; rely on `gofmt` (`make fmt`) before opening a PR.
- Use tabs/formatting produced by `gofmt`; do not hand-align.
- Package names are short, lowercase, no underscores.
- Exported identifiers use `CamelCase`; unexported helpers use `camelCase`.
- Logging in this repository is `log/slog`-first: use `log/slog` for application logs and do not introduce other logging frameworks.
- Prefer descriptive file names by concern (for example, `protocol_text.go`, `linked_list.go`).

## Testing Guidelines
- Use Go’s built-in `testing` package.
- Test files must end with `_test.go`; prefer table-driven tests for protocol/cache behavior.
- Add regression tests for bug fixes and edge cases (TTL expiry, numeric overflow, eviction boundaries).
- Run both root and benchmark-module tests before submitting.

## Commit & Pull Request Guidelines
- Commit messages in this repo are short, imperative, and concise (examples: `add GETS`, `refactor`, `mod README`).
- Keep commits focused; avoid mixing refactors with behavior changes.
- PRs should include a clear summary of behavior changes, linked issue(s) when applicable, test evidence (commands run), and protocol request/response examples when command behavior changes.

## Security & Configuration Tips
- This is a volatile in-memory store; do not use it for durable or sensitive data.
- Default listen address is local (`127.0.0.1:11211`); keep it restricted unless intentionally exposing it.
