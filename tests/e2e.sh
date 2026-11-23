#!/usr/bin/env bash
# DockNuke E2E: exercises the CLI against the local docker daemon.
# Creates real junk resources (stopped containers, dangling image + volume),
# verifies --dry-run leaves them alone and a real run removes exactly them.
set -euo pipefail
cd "$(dirname "$0")/.."

BIN=$(mktemp -d)/docknuke
go build -o "$BIN" ./cmd/docknuke

PFX="docknuke-e2e"
cleanup() {
  docker rm -f "${PFX}-a" "${PFX}-b" "${PFX}-keep" >/dev/null 2>&1 || true
  docker volume rm "${PFX}-vol" >/dev/null 2>&1 || true
  docker rmi "${PFX}-img:latest" >/dev/null 2>&1 || true
}
cleanup  # clean slate; and again on exit
trap cleanup EXIT

# --- create fixtures --------------------------------------------------------
docker run --name "${PFX}-a" busybox:1.36 true >/dev/null
docker run --name "${PFX}-b" busybox:1.36 true >/dev/null
docker run -d --name "${PFX}-keep" busybox:1.36 sleep 300 >/dev/null   # running, must survive
docker volume create "${PFX}-vol" >/dev/null
docker tag busybox:1.36 "${PFX}-img:latest" >/dev/null
docker rmi "${PFX}-img:latest" >/dev/null && docker build -qt "${PFX}-img" - <<'EOF' >/dev/null
FROM busybox:1.36
RUN echo junk > /junk
EOF
docker rmi "${PFX}-img" >/dev/null || true   # leaves a dangling layer? ensure a dangling image exists:
docker build -qt "${PFX}-img" - <<'EOF' >/dev/null
FROM busybox:1.36
RUN echo junk2 > /junk2
EOF
IMG_ID=$(docker images -q "${PFX}-img")
docker rmi "${PFX}-img" >/dev/null           # untag → dangling

assert() { if ! eval "$2"; then echo "FAIL: $1"; exit 1; fi; echo "ok: $1"; }

# --- dry-run ----------------------------------------------------------------
OUT=$("$BIN" --dry-run --volumes)
echo "$OUT"
assert "dry-run lists both stopped containers" 'grep -q "${PFX}-a" <<<"$OUT" && grep -q "${PFX}-b" <<<"$OUT"'
assert "dry-run lists the dangling volume"     'grep -q "${PFX}-vol" <<<"$OUT"'
assert "dry-run deletes nothing"               'docker inspect "${PFX}-a" >/dev/null && docker volume inspect "${PFX}-vol" >/dev/null'

# --- real run, scoped -------------------------------------------------------
"$BIN" --exclude keep --volumes | tee /tmp/dn-out.txt
assert "stopped container A removed"           '! docker inspect "${PFX}-a" >/dev/null 2>&1'
assert "stopped container B removed"           '! docker inspect "${PFX}-b" >/dev/null 2>&1'
assert "running container survives"            'docker inspect -f {{.State.Running}} "${PFX}-keep" | grep -q true'
assert "dangling volume removed"               '! docker volume inspect "${PFX}-vol" >/dev/null 2>&1'
assert "untagged image removed"                '! docker image inspect "$IMG_ID" >/dev/null 2>&1'
assert "busybox still present (in use)"        'docker image inspect busybox:1.36 >/dev/null'
assert "reclaimed bytes reported"              'grep -q "total reclaimed:" /tmp/dn-out.txt'

# --- idempotent -------------------------------------------------------------
"$BIN" --volumes | tee /tmp/dn-out2.txt
assert "second run removes zero"               'grep -q "containers removed: 0" /tmp/dn-out2.txt'

echo "E2E PASSED"
