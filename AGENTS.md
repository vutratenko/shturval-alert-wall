# AGENTS — shturval-alert-wall

- Build/test: `make test` (npm tests + vite build + `go test ./...`).
- Local UI: `SOURCE_MODE=fake FAKE_ALERTS_PATH=config/samples/fake-alerts.json make run`.
- Helm: `helm lint charts/shturval-alert-wall`.
- Details: [README.md](README.md).