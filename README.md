# Shturval Alert Wall

Live multi-cluster **firing** alert dashboard for Shturval (VictoriaMetrics / Alertmanager). One Go binary embeds the React wall UI; no database.

## Architecture

- Discovers `VMAlertmanager` CRs (plus optional static URLs) and polls `GET /api/v2/alerts?active=true&silenced=false&inhibited=false` per cluster.
- Fallback mode `SOURCE_MODE=vmselect` queries `ALERTS{alertstate="firing"}` on VMSelect.
- Browser uses SSE (`/api/stream`) for snapshots and diff events.

Image: `registry.sion2k.ru/shturval/shturval-alert-wall`.

## GitOps (Argo CD)

- Overlay: `charts/shturval-alert-wall/values-cluster.yaml` (CI bumps `image.tag` on push to `main`).
- Application manifest: [deploy/application.yaml](deploy/application.yaml) (project `default`, namespace `shturval-alert-wall`).
- Private repo: deploy key + Secret `deploy/argocd-repo-secret.example.yaml` in `shturval-cd`.
- Copy registry pull secret into `shturval-alert-wall` as `regcred` (e.g. clone `registry-shturval-pull` from `shturval-ctf`).

## Local development

Requires Go 1.24+, Node 22+.

```sh
make test
SOURCE_MODE=fake FAKE_ALERTS_PATH=config/samples/fake-alerts.json make run
```

Open [http://localhost:8080/wall](http://localhost:8080/wall).

### API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/healthz` | Liveness |
| GET | `/api/snapshot` | Current aggregated state |
| GET | `/api/stream` | SSE (snapshot + events) |
| GET | `/metrics` | Prometheus metrics |

### Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `SOURCE_MODE` | `alertmanager` | `alertmanager`, `vmselect`, or `fake` |
| `VMSELECT_URL` | | Required for `vmselect` mode |
| `STATIC_SOURCES` | | `cluster1=http://...;cluster2=http://...` |
| `POLL_INTERVAL` | `1s` | Poll period |
| `FAKE_ALERTS_PATH` | | JSON file for `fake` mode |

## Helm

```sh
helm upgrade --install alert-wall charts/shturval-alert-wall \
  --namespace shturval-alert-wall --create-namespace \
  --set image.tag=latest
kubectl port-forward -n shturval-alert-wall svc/shturval-alert-wall 8080:8080
```

## Smoke test

```sh
MODE=local ./scripts/smoke-test.sh
```

## UI

Design tokens from Shturval ML webinar deck (Montserrat, `#1cf2de`, glass cards). Fox mascot states: calm / worried / panic driven by aggregate alert counts.