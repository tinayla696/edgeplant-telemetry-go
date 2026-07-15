#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

LOG_DIR="$ROOT_DIR/logs"
mkdir -p "$LOG_DIR"

WEB_LOG="$LOG_DIR/gamepadweb.log"
CAN_LOG="$LOG_DIR/gamepad_test_can.log"

cleanup() {
  if [[ -n "${WEB_PID:-}" ]]; then
    kill "$WEB_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

docker compose up -d mqtt-broker gpsd-mock telemetry

for _ in $(seq 1 30); do
  if docker compose ps telemetry | grep -q "Up" \
    && docker compose exec -T telemetry bash -lc "command -v candump >/dev/null && ip link show can0 >/dev/null 2>&1" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker compose exec -T telemetry bash -lc "command -v candump >/dev/null && ip link show can0 >/dev/null 2>&1" >/dev/null 2>&1; then
  echo "Telemetry container is not ready (candump/can0 missing)." >&2
  exit 1
fi

cd "$ROOT_DIR/src"
go run ./cmd/gamepadweb \
  -listen :8088 \
  -mqtt 127.0.0.1:1883 \
  -device vcan-e2e \
  -topic-prefix ctrl \
  -bus can0 \
  -map-config ../config/gamepad_mapping_all_inputs.yaml >"$WEB_LOG" 2>&1 &
WEB_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:8088/ >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

timeout 15s docker compose exec -T telemetry bash -lc "candump can0 -n 1" >"$CAN_LOG" &
CANDUMP_PID=$!
sleep 1

curl -fsS -X POST http://127.0.0.1:8088/api/gamepad \
  -H "Content-Type: application/json" \
  -d '{"axes":[0,-0.75],"buttons":[true]}' >/dev/null

wait "$CANDUMP_PID"

grep -Eiq 'can0\s+210' "$CAN_LOG"

echo "PASS: Gamepad sample flow published MQTT command and telemetry emitted CAN frame (0x210)."
