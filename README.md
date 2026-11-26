# Shturval Alert Wall

Live multi-cluster **firing** alert dashboard for Shturval (VictoriaMetrics / Alertmanager). One Go binary embeds the React wall UI; no database.

## Architecture

- Discovers `VMAlertmanager` CRs (plus optional static URLs) and polls `GET /api/v2/alerts?active=true&silenced=false&inhibited=false` per cluster.
- Fallback mode `SOURCE_MODE=vmselect` queries `ALERTS{alertstate="firing"}` on VMSelect.
- Browser uses SSE (`/api/stream`) for snapshots and diff events.

Image: `ghcr.io/vutratenko/shturval-alert-wall`.

## GitOps (Argo CD)

- Overlay: `charts/shturval-alert-wall/values-cluster.yaml` (CI bumps `image.tag` on push to `main`).
- Application manifest: [deploy/application.yaml](deploy/application.yaml) (project `default`, namespace `shturval-alert-wall`).
- Private repo: deploy key + Secret `deploy/argocd-repo-secret.example.yaml` in `shturval-cd`.
- The image is a public GHCR package, so the namespace does not need a registry pull secret.

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
| GET | `/api/healthz` | Liveness (`status`, `service: shturval-alert-wall`) |
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
| `PUSH_STORE_PATH` | | Directory for VAPID keys and device subscriptions. Empty keeps them in memory |
| `PUSH_SUBJECT` | `mailto:alerts@sion2k.ru` | Contact URI sent in the VAPID token |

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

## PWA and Web Push

The wall is an installable PWA. Open [https://alerts.sion2k-cluster.sion2k.ru/wall](https://alerts.sion2k-cluster.sion2k.ru/wall) over HTTPS (`localhost` is enough for a local check). The installed app caches the UI shell only. Alert data still comes from the live SSE stream, and an offline reload shows an empty wall with the offline badge.

The header toggle enables Web Push for this browser only. Turning it on asks for notification permission and registers this device. Turning it off unsubscribes this device and leaves every other device as it was. A tap on a notification opens `/wall`. On iPhone, push works after the PWA is added to the Home Screen (iOS 16.4+).

`npm run build` emits the manifest and service worker. CI checks those files, and the image build runs the same command, so a push to `main` ships them inside the image. Argo CD rolls that image out as before. Subscriptions and the VAPID key pair live on a ReadWriteOnce PVC (`push.persistence` in the cluster values, mounted at `/data`) so a new image tag does not drop devices. Keep `replicas: 1` while that volume is in use.

## Browser add-on

`addon/` is one Manifest V3 extension for Chrome and Firefox. It polls each wall you add and shows a toolbar badge plus local notifications. Sign-in stays on that wall: the add-on sends the existing `alertwall_session` cookie and does not start its own OIDC client. Web Push is unchanged. The server image does not include the add-on.

```sh
cd addon && npm ci && npm test && npm run build
```

Load `addon/dist` as an unpacked extension. In the add-on options, paste the wall origin (`https://alerts.example.com` or a `/wall` URL). The browser asks for that origin only. `http://` is accepted for `localhost` and `127.0.0.1`, including a non-default port. The options page checks `GET /api/healthz` and keeps the address only when `service` is `shturval-alert-wall`.

The background alarm is 20 seconds. Chrome may delay packed-extension alarms to one minute. A click on a notification opens `/wall` for that installation. Publishing to the Chrome Web Store and AMO is separate from this build.