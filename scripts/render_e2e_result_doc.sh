#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

MODE="${1:-}"
STATUS="${2:-Pass}"

case "$MODE" in
  mqtt|rabbitmq) ;;
  *)
    echo "usage: $0 <mqtt|rabbitmq> [Pass|Fail]" >&2
    exit 2
    ;;
esac

DOC_PATH="$ROOT_DIR/docs/test-results-${MODE}.md"
UPLINK_JSON="$ROOT_DIR/logs/e2e_${MODE}_uplink.json"
DOWNLINK_CAN="$ROOT_DIR/logs/e2e_${MODE}_downlink_can.log"
APP_LOG="$ROOT_DIR/logs/app.${MODE}.log"
SERVER_UPLINK_LOG="$ROOT_DIR/logs/server.${MODE}.uplink.log"
SERVER_DOWNLINK_LOG="$ROOT_DIR/logs/server.${MODE}.downlink.log"

BROKER_LABEL="MQTT"
if [[ "$MODE" == "rabbitmq" ]]; then
  BROKER_LABEL="RabbitMQ"
fi

EXECUTED_AT="${E2E_EXECUTED_AT:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
COMMAND_LINE="${E2E_COMMAND:-./scripts/e2e_vcan.sh $MODE}"
DEVICE_ID="${E2E_DEVICE_ID:-vcan-e2e}"
RUNNER_NAME="${RUNNER_NAME:-${HOSTNAME:-unknown}}"
GIT_SHA="${GITHUB_SHA:-$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)}"
BRANCH_NAME="${GITHUB_REF_NAME:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
WORKFLOW_NAME="${GITHUB_WORKFLOW:-local}"
RUN_LINK=""
if [[ -n "${GITHUB_SERVER_URL:-}" && -n "${GITHUB_REPOSITORY:-}" && -n "${GITHUB_RUN_ID:-}" ]]; then
  RUN_LINK="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
fi

emit_file_block() {
  local file_path="$1"
  local fallback="$2"

  if [[ -s "$file_path" ]]; then
    cat "$file_path"
  else
    printf '%s\n' "$fallback"
  fi
}

{
  printf '# %s E2E Test Result\n\n' "$BROKER_LABEL"
  printf -- '- Executed at: %s\n' "$EXECUTED_AT"
  printf -- '- Mode: %s\n' "$MODE"
  printf -- '- Device ID: %s\n' "$DEVICE_ID"
  printf -- '- Result: %s\n' "$STATUS"
  printf -- '- Command: `%s`\n' "$COMMAND_LINE"
  printf -- '- Runner: %s\n' "$RUNNER_NAME"
  printf -- '- Branch: `%s`\n' "$BRANCH_NAME"
  printf -- '- Commit: `%s`\n' "$GIT_SHA"
  printf -- '- Workflow: %s\n' "$WORKFLOW_NAME"
  if [[ -n "$RUN_LINK" ]]; then
    printf -- '- GitHub Actions Run: %s\n' "$RUN_LINK"
  fi
  if [[ -n "${GITHUB_RUN_ATTEMPT:-}" ]]; then
    printf -- '- Run Attempt: %s\n' "$GITHUB_RUN_ATTEMPT"
  fi

  printf '\n## Validation Scope\n\n'
  printf '1. Verify uplink JSON against CAN input and GPS mock output\n'
  printf '2. Verify downlink JSON command against emitted CAN frame\n'
  printf '3. Preserve app and compose logs for post-run inspection\n'

  printf '\n## Observed Uplink JSON\n\n'
  printf '```json\n'
  emit_file_block "$UPLINK_JSON" '{"status":"missing uplink log"}'
  printf '\n```\n'

  printf '\n## Observed Downlink CAN\n\n'
  printf '```text\n'
  emit_file_block "$DOWNLINK_CAN" 'missing downlink CAN log'
  printf '\n```\n'

  printf '\n## Saved Logs\n\n'
  printf -- '- `logs/app.%s.log`\n' "$MODE"
  printf -- '- `logs/server.%s.uplink.log`\n' "$MODE"
  printf -- '- `logs/server.%s.downlink.log`\n' "$MODE"
  printf -- '- `logs/e2e_%s_uplink.json`\n' "$MODE"
  printf -- '- `logs/e2e_%s_downlink_can.log`\n' "$MODE"

  printf '\n## Notes\n\n'
  if [[ "$STATUS" == "Pass" ]]; then
    printf 'The scenario completed successfully and the generated logs match the expected CAN/GPS round-trip flow.\n'
  else
    printf 'The scenario did not complete successfully. Review the saved logs listed above for the failing step.\n'
  fi
} >"$DOC_PATH"

printf 'Wrote %s\n' "$DOC_PATH"
