#!/usr/bin/env bash
# One-time: copy Shturval registry credentials into shturval-alert-wall.
set -euo pipefail
SRC_NS="${SRC_NS:-shturval-ctf}"
SRC_SECRET="${SRC_SECRET:-registry-shturval-pull}"
DST_NS="${DST_NS:-shturval-alert-wall}"
kubectl get secret "$SRC_SECRET" -n "$SRC_NS" -o yaml \
  | sed "s/namespace: $SRC_NS/namespace: $DST_NS/" \
  | sed "s/name: $SRC_SECRET/name: regcred/" \
  | grep -v 'resourceVersion:\|uid:\|creationTimestamp:' \
  | kubectl apply -f -