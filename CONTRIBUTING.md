# Contributing to siyi-sdk-go

Thank you for your interest in contributing to `siyi-sdk-go`! We welcome contributions of all kinds, including bug reports, feature suggestions, documentation enhancements, protocol fixes, and pull requests.

---

## Reporting Issues

When reporting an issue, please include:
1. **Camera Model**: (e.g., UniPod MT11, ZT30, ZT6, A8 mini, ZR30, ZR10, A2 mini, R1M).
2. **Firmware Version**: (queried via `client.GetFirmwareVersion()` or `siyi-cli -cmd info`).
3. **Transport Used**: (UDP, TCP, or UART Serial, plus baud rate/ports).
4. **Go Version & OS**: (e.g., `go version`, Linux/macOS/Windows).
5. **Minimal Reproducible Example**: A short code snippet or step-by-step instructions reproducing the behavior.

---

## Development Setup & Verification

Before submitting a pull request, ensure your code satisfies all quality gates:

```sh
# 1. Run standard unit and integration tests
go test ./...

# 2. Run race condition detector (must pass with 0 data races)
go test -race ./...

# 3. Run static analysis
go vet ./...

# 4. Run golangci-lint (must pass with 0 errors)
golangci-lint run ./...

# 5. Build CLI and Web dashboard binaries
go build -v ./cmd/siyi-cli
go build -v ./cmd/siyi-web
```

---

## Coding Guidelines

1. **Idiomatic Go**: Follow standard Go formatting (`gofmt`/`goimports`). Keep functions focused and return structured errors.
2. **SOLID Principles**: Adhere to clean package boundaries:
   - `pkg/protocol`: Low-level binary framing, CRC-16, and state-machine parser.
   - `pkg/models`: Strongly-typed request/response models and enums.
   - `pkg/commands`: Command encoders and decoders.
   - `pkg/transport`: Pluggable network and serial transports (`Transport` interface).
   - `pkg/stream`: Pure-Go RTSP stream demuxing (no CGo/GStreamer dependencies).
   - `pkg/siyi`: High-level client coordinating lifecycle, retries, and subscriptions.
3. **Thread Safety & Race Freedom**: All stateful abstractions must be thread-safe. Use `sync.Mutex`, `sync.RWMutex`, or `atomic` primitives where appropriate. Never send to a closed channel.
4. **Mock Testing**: Use `transport.NewMockTransport()` and `mock.SetOnSend()` for deterministic unit testing without requiring physical camera hardware.

---

## Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages and PR titles:

- `feat:` A new feature or protocol command
- `fix:` A bug fix or timing correction
- `docs:` Documentation changes only
- `style:` Formatting, whitespace, or asset layout
- `refactor:` Code restructuring without behavior changes
- `test:` Adding or updating tests
- `ci:` Changes to GitHub Actions or build scripts

---

## Versioning & Releases

`siyi-sdk-go` adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Release tags use the `vMAJOR.MINOR.PATCH` format.

To propose a release:
1. Update `const Version` in `version.go`.
2. Add a new `## [vX.Y.Z] - YYYY-MM-DD` section with release notes to `CHANGELOG.md`.
3. Open a pull request against `main`.
4. Once merged, create and push the matching Git tag:
   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```
5. The automated GitHub Actions release workflow will verify the tag against `version.go` and `CHANGELOG.md`, extract the release notes, execute test suites, and publish the GitHub Release.

---

## Licensing

Contributions intentionally submitted for inclusion in `siyi-sdk-go` are licensed under the MIT License as described in `LICENSE`, unless explicitly stated otherwise in a separate written agreement.
