# Contributing to Hatch

Thanks for wanting to improve Hatch — the embedded harness for a coding-agent squad.

## Develop

```bash
./scripts/onboard.sh   # build + a runnable demo (mock agent)
make test              # unit + integration tests
make lint              # go vet + gofmt check (CI enforces both)
make build             # bin/hatch + bin/hatch-mock
```

## Standards for Go changes

- [ ] `make lint` and `make test` pass (CI runs them on every PR)
- [ ] `go mod tidy` leaves `go.mod`/`go.sum` unchanged
- [ ] New behavior has a test; security-relevant paths are sanitized
- [ ] No secrets in code/config; credentials come from env vars only
- [ ] Keep packages small and documented; match the surrounding style

See [`docs/`](docs/) for the design and [`SECURITY.md`](SECURITY.md) for trust boundaries.

## How to Contribute

1. Fork the repo
2. Create a branch (`feat/<short-description>`)
3. Make your change with tests
4. Submit a PR with a brief description of the change and its rationale

## Bug Reports

Open an issue with:

- What you expected vs what happened
- Reproduction steps
- Any error messages or logs

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). In short: be kind, be helpful,
skip the drama.
