# Contributing to sinty-input

## Quick Start

```bash
git clone https://github.com/singularityos-lab/sinty-input
cd sinty-input
go build ./...
go test ./...
```

## Guidelines

- Keep changes small and tested. Every engine has tests that type real
  words; add the word that was wrong before fixing it.
- A key the engine does not use must reach the application unchanged.
- Match the surrounding style.
- By submitting a contribution you agree to the [CLA](CLA.md).

## License

GPL-3.0-only.
