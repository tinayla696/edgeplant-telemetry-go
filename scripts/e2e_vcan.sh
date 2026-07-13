#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

MODE="${1:-mqtt}"
case "$MODE" in
  mqtt|rabbitmq) ;;
  *)
    echo "usage: $0 [mqtt|rabbitmq]" >&2
    exit 2
    ;;
esac

APP_LOG="$ROOT_DIR/logs/app.log"
APP_MODE_LOG="$ROOT_DIR/logs/app.${MODE}.log"
SERVER_LOG="$ROOT_DIR/logs/server.log"
SERVER_UPLINK_LOG="$ROOT_DIR/logs/server.${MODE}.uplink.log"
SERVER_DOWNLINK_LOG="$ROOT_DIR/logs/server.${MODE}.downlink.log"
UPLINK_JSON="$ROOT_DIR/logs/e2e_${MODE}_uplink.json"
DOWNLINK_CAN="$ROOT_DIR/logs/e2e_${MODE}_downlink_can.log"
RESULT_STATUS="Fail"

export E2E_BROKER_TYPE="$MODE"
if [[ "$MODE" == "mqtt" ]]; then
  export E2E_MQTT_ENDPOINT="127.0.0.1:1883"
else
  export E2E_RABBITMQ_URL="amqp://guest:guest@127.0.0.1:5672/"
fi

capture_server_log() {
  local out="$1"
  docker compose logs --no-color >"$out" 2>&1 || true
  cp "$out" "$SERVER_LOG" 2>/dev/null || true
}

subscribe_uplink() {
  if [[ "$MODE" == "mqtt" ]]; then
    timeout 20s docker compose exec -T mqtt-tools sh -lc \
      "mosquitto_sub -h 127.0.0.1 -p 1883 -t '/tx/vcan-e2e/state' | grep -m1 '\"can0\"'" >"$UPLINK_JSON"
    return
  fi

  timeout 25s docker compose exec -T amqp-tools sh -lc "python3 - <<'PY' > /workspace/logs/.amqp_uplink_tmp.json
import json
import pika

params = pika.URLParameters('amqp://guest:guest@127.0.0.1:5672/')
conn = pika.BlockingConnection(params)
ch = conn.channel()
ch.exchange_declare(exchange='amq.topic', exchange_type='topic', durable=True)
result = ch.queue_declare(queue='', exclusive=True, auto_delete=True)
queue_name = result.method.queue
ch.queue_bind(exchange='amq.topic', queue=queue_name, routing_key='/tx/vcan-e2e/state')
for method, props, body in ch.consume(queue_name, inactivity_timeout=20, auto_ack=True):
    if body is None:
        continue
    text = body.decode()
    if '\"can0\"' in text:
        print(text)
        break
ch.cancel()
conn.close()
PY
cat /workspace/logs/.amqp_uplink_tmp.json" >"$UPLINK_JSON"
  rm -f "$ROOT_DIR/logs/.amqp_uplink_tmp.json"
}

publish_downlink() {
  local payload='{"Timestamp":"2024-06-10T15:30:30+09:00","bus_id":"can0","frame_id":"0x2","signals":{"ControlCommand":true,"ControlValue":-10.0}}'
  if [[ "$MODE" == "mqtt" ]]; then
    docker compose exec -T mqtt-tools sh -lc "mosquitto_pub -h 127.0.0.1 -p 1883 -t '/rx/vcan-e2e/ctrl' -m '$payload'"
    return
  fi

  docker compose exec -T amqp-tools sh -lc "python3 - <<'PY'
import pika

payload = '''$payload'''
params = pika.URLParameters('amqp://guest:guest@127.0.0.1:5672/')
conn = pika.BlockingConnection(params)
ch = conn.channel()
ch.exchange_declare(exchange='amq.topic', exchange_type='topic', durable=True)
ch.basic_publish(exchange='amq.topic', routing_key='/rx/vcan-e2e/ctrl', body=payload.encode())
conn.close()
PY"
}

cleanup() {
  docker compose exec -T telemetry bash -lc "chmod -R a+rX /workspace/logs" >/dev/null 2>&1 || true
  capture_server_log "$SERVER_DOWNLINK_LOG"
  "$ROOT_DIR/scripts/render_e2e_result_doc.sh" "$MODE" "$RESULT_STATUS" >/dev/null 2>&1 || true
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Ensure a clean state from previous runs.
docker compose down -v --remove-orphans >/dev/null 2>&1 || true
rm -f "$APP_LOG" "$APP_MODE_LOG" "$SERVER_LOG" "$SERVER_UPLINK_LOG" "$SERVER_DOWNLINK_LOG" "$UPLINK_JSON" "$DOWNLINK_CAN"

# Start infra and app.
docker compose up -d mqtt-broker rabbitmq gpsd-mock telemetry mqtt-tools amqp-tools

for _ in $(seq 1 30); do
  if docker compose ps telemetry | grep -q "Up" \
    && docker compose exec -T telemetry bash -lc "command -v cansend >/dev/null && ip link show can0 >/dev/null 2>&1 && test -f /workspace/logs/app.log" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! docker compose exec -T telemetry bash -lc "command -v cansend >/dev/null && ip link show can0 >/dev/null 2>&1 && test -f /workspace/logs/app.log" >/dev/null 2>&1; then
  echo "Telemetry test tools, vcan, or app log not ready" >&2
  exit 1
fi

docker compose exec -T telemetry bash -lc "chmod -R a+rX /workspace/logs"

# 1) Verify uplink publish value against CAN input and GPS mock.
subscribe_uplink &
SUB_PID=$!
sleep 1

docker compose exec -T telemetry bash -lc "cansend can0 001#0001020304050607"

wait "$SUB_PID"
capture_server_log "$SERVER_UPLINK_LOG"

grep -q '"speed":2.56' "$UPLINK_JSON"
grep -q '"left_turn":0' "$UPLINK_JSON"
grep -q '"right_turn":1' "$UPLINK_JSON"
grep -q '"steering_deg":102.7' "$UPLINK_JSON"
grep -q '"brake":2' "$UPLINK_JSON"
grep -q '"latitude":35.6812' "$UPLINK_JSON"
grep -q '"longitude":139.7671' "$UPLINK_JSON"
grep -q '"speed":10' "$UPLINK_JSON"

# 2) Verify subscribe value against CAN output.
timeout 15s docker compose exec -T telemetry bash -lc "candump can0 -n 1" >"$DOWNLINK_CAN" &
CANDUMP_PID=$!
sleep 1

publish_downlink
wait "$CANDUMP_PID"

grep -Eiq 'can0\s+002' "$DOWNLINK_CAN"
grep -Eiq '01 9C FF 00 00 00 00 00|019CFF0000000000' "$DOWNLINK_CAN"

grep -q 'Successfully connected to GPSD' "$APP_LOG"
cp "$APP_LOG" "$APP_MODE_LOG"
RESULT_STATUS="Pass"

echo "E2E OK ($MODE): publish/CAN consistency, subscribe/CAN consistency, and GPS mock verification passed."
