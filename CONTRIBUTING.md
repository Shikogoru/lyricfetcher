# Contributing to LyricFetcher

Thank you for your interest in contributing to LyricFetcher! We welcome issues, bug reports, feature requests, and pull requests.

## Code of Conduct

Please be respectful, collaborative, and considerate when interacting with the community and maintainers.

## Reporting Issues & Feature Requests

- **Bug Reports**: Please open an issue on GitHub detailing:
  - Your OS and LyricFetcher version (`lyricfetcher --version`)
  - The exact command run and CLI output
  - Steps to reproduce the problem
- **Feature Requests**: Describe the problem you are solving, the proposed CLI behavior or provider, and how it benefits users.

## Submitting Pull Requests

1. **Fork the repository** on GitHub.
2. **Clone your fork** and create a feature branch:
   ```bash
   git checkout -b feature/my-new-feature
   ```
3. **Make your changes**:
   - Adhere to idiomatic Go conventions.
   - Format all code with `gofmt`:
     ```bash
     gofmt -w .
     ```
   - Ensure `go vet` reports zero warnings:
     ```bash
     go vet ./...
     ```
   - Run tests (and maintain >70% test coverage):
     ```bash
     go test -race ./...
     ```
4. **Commit and push** your branch:
   ```bash
   git commit -m "feat: add support for new feature"
   git push origin feature/my-new-feature
   ```
5. **Open a Pull Request** against the `main` branch.

## Adding a New Language (i18n)

Adding a new language is simple and requires no Go code changes:
1. Create a new JSON file in `internal/i18n/locales/<lang_code>.json` (for example, `ja.json` or `es.json`).
2. Copy all keys from `internal/i18n/locales/en.json` and translate the values.
3. Keep all placeholders (`%s`, `%d`, `%v`, `%q`, `%.1fs`) identical.
4. Run `go test -v ./internal/i18n/...` to verify parity with existing locales.
