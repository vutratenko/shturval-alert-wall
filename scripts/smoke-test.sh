#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
NS="${SMOKE_NAMESPACE:-shturval-alert-wall}"
MODE="${MODE:-local}"
if [[ "$MODE" == "local" ]]; then
  export SOURCE_MODE=fake
  export FAKE_ALERTS_PATH="$ROOT/config/samples/fake-alerts.json"
  go run ./cmd/server &
  PID=$!
  trap 'kill $PID 2>/dev/null || true' EXIT
  sleep 2
else
  helm upgrade --install alert-wall charts/shturval-alert-wall --namespace "$NS" --create-namespace --wait --timeout 5m
  kubectl -n "$NS" port-forward svc/shturval-alert-wall 18080:8080 &
  PF=$!
  trap 'kill $PF 2>/dev/null || true' EXIT
  sleep 2
  export SMOKE_BASE=http://127.0.0.1:18080
fi
BASE="${SMOKE_BASE:-http://127.0.0.1:8080}"
curl -sf "$BASE/api/healthz" | grep -q ok
MOOD="$(curl -sf "$BASE/api/snapshot" | grep -o '"mood":"[^"]*"' | head -1)"
if [[ "$MOOD" != *worried* && "$MOOD" != *panic* && "$MOOD" != *calm* ]]; then
  echo "unexpected snapshot: $MOOD"
  exit 1
fi
echo "smoke ok: $MOOD"