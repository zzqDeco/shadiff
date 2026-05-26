#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
PROJECT_NAME="${SHADIFF_E2E_PROJECT:-shadiff-e2e}"
RUN_ID="${SHADIFF_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
WORK_DIR="${SCRIPT_DIR}/.work/${RUN_ID}"
CONFIG_FILE="${WORK_DIR}/config.json"
SESSION_NAME="e2e-${RUN_ID}"

ASSERT=0
KEEP=0
RECORD_PID=""
REPLAY_PID=""

usage() {
  cat <<'USAGE'
Usage: examples/e2e/run.sh [--assert] [--keep]

Runs the official Shadiff Docker Compose E2E demo.

Options:
  --assert  Verify that diff.json contains expected SQL and MongoDB differences.
  --keep    Keep Docker Compose services running after the script exits.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --assert)
      ASSERT=1
      ;;
    --keep)
      KEEP=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

log() {
  printf '[shadiff-e2e] %s\n' "$*"
}

fail() {
  printf '[shadiff-e2e] error: %s\n' "$*" >&2
  exit 1
}

choose_compose() {
  if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose -f "${COMPOSE_FILE}" -p "${PROJECT_NAME}")
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE=(docker-compose -f "${COMPOSE_FILE}" -p "${PROJECT_NAME}")
    return
  fi
  fail "Docker Compose is required"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "$1 is required"
  fi
}

cleanup() {
  if [[ -n "${RECORD_PID}" ]] && kill -0 "${RECORD_PID}" >/dev/null 2>&1; then
    kill -TERM "${RECORD_PID}" >/dev/null 2>&1 || true
    wait "${RECORD_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${REPLAY_PID}" ]] && kill -0 "${REPLAY_PID}" >/dev/null 2>&1; then
    kill -TERM "${REPLAY_PID}" >/dev/null 2>&1 || true
    wait "${REPLAY_PID}" >/dev/null 2>&1 || true
  fi

  if [[ "${KEEP}" -eq 0 ]]; then
    "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  else
    log "keeping Docker Compose services for project ${PROJECT_NAME}"
  fi
}
trap cleanup EXIT

wait_http() {
  local url="$1"
  local label="$2"
  local deadline=$((SECONDS + 90))

  while (( SECONDS < deadline )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  fail "timed out waiting for ${label}: ${url}"
}

wait_tcp() {
  local host="$1"
  local port="$2"
  local label="$3"
  local deadline=$((SECONDS + 30))

  while (( SECONDS < deadline )); do
    if (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  fail "timed out waiting for ${label}: ${host}:${port}"
}

wait_process_exit() {
  local pid="$1"
  local label="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))

  while kill -0 "${pid}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      log "forcing ${label} to stop"
      kill -KILL "${pid}" >/dev/null 2>&1 || true
      break
    fi
    sleep 1
  done

  wait "${pid}" >/dev/null 2>&1 || true
}

wait_log_marker() {
  local log_file="$1"
  local marker="$2"
  local pid="$3"
  local label="$4"
  local timeout_seconds="$5"
  local deadline=$((SECONDS + timeout_seconds))

  while true; do
    if [[ -f "${log_file}" ]] && grep -q "${marker}" "${log_file}"; then
      return
    fi
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      wait "${pid}" >/dev/null 2>&1 || true
      [[ -f "${log_file}" ]] && grep -q "${marker}" "${log_file}" && return
      fail "${label} exited before completion marker: ${marker}"
    fi
    if (( SECONDS >= deadline )); then
      fail "timed out waiting for ${label} completion marker: ${marker}"
    fi
    sleep 1
  done
}

write_config() {
  mkdir -p "${WORK_DIR}/data" "${WORK_DIR}/logs" "${WORK_DIR}/artifacts"
  cat >"${CONFIG_FILE}" <<EOF
{
  "capture": {
    "listenAddr": "127.0.0.1:18080",
    "maxBodySize": 10485760
  },
  "replay": {
    "concurrency": 1,
    "timeout": "30s",
    "retryCount": 0
  },
  "diff": {
    "ignoreHeaders": ["Date", "X-Request-Id", "X-Trace-Id", "Server", "Content-Length"],
    "ignoreOrder": false,
    "maxDiffs": 1000
  },
  "storage": {
    "dataDir": "${WORK_DIR}/data",
    "maxSessions": 10
  },
  "log": {
    "level": "info",
    "logDir": "${WORK_DIR}/logs"
  }
}
EOF
}

build_shadiff() {
  if [[ -n "${SHADIFF_BIN:-}" ]]; then
    if [[ ! -x "${SHADIFF_BIN}" ]]; then
      fail "SHADIFF_BIN is not executable: ${SHADIFF_BIN}"
    fi
    return
  fi

  require_command go
  mkdir -p "${ROOT_DIR}/build/bin"
  SHADIFF_BIN="${ROOT_DIR}/build/bin/shadiff"
  log "building ${SHADIFF_BIN}"
  (cd "${ROOT_DIR}" && go build -o "${SHADIFF_BIN}" .)
}

assert_demo() {
  local diff_json="${WORK_DIR}/artifacts/diff.json"
  local report_html="${WORK_DIR}/artifacts/report.html"
  local records_file
  local replay_file

  records_file="$(find "${WORK_DIR}/data/sessions" -name records.jsonl -print -quit)"
  replay_file="$(find "${WORK_DIR}/data/sessions" -name replay-records.jsonl -print -quit)"

  [[ -s "${records_file}" ]] || fail "record stage did not write records.jsonl"
  [[ -s "${replay_file}" ]] || fail "replay stage did not write replay-records.jsonl"
  [[ -s "${diff_json}" ]] || fail "diff stage did not write diff.json"
  [[ -s "${report_html}" ]] || fail "report stage did not write report.html"

  grep -Eq '"totalCount": [1-9]' "${diff_json}" || fail "diff summary has no records"
  grep -Eq '"diffCount": [1-9]' "${diff_json}" || fail "diff summary has no expected differences"
  grep -q '"kind": "db_query"' "${diff_json}" || fail "diff.json does not contain a SQL side-effect difference"
  grep -q '"kind": "mongo_op"' "${diff_json}" || fail "diff.json does not contain a MongoDB side-effect difference"
  grep -q '"kind": "redis_command"' "${diff_json}" || fail "diff.json does not contain a Redis side-effect difference"
}

require_command docker
require_command curl
choose_compose
write_config
build_shadiff

log "work directory: ${WORK_DIR}"
log "starting Docker Compose services"
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
"${COMPOSE[@]}" up -d --build

wait_http "http://127.0.0.1:18081/health" "old-api"
wait_http "http://127.0.0.1:18082/health" "new-api"

log "starting record stage"
"${SHADIFF_BIN}" --config "${CONFIG_FILE}" record \
  --target http://127.0.0.1:18081 \
  --listen 127.0.0.1:18080 \
  --session "${SESSION_NAME}" \
  --duration 60s \
  --db-proxy "mysql://:13306->127.0.0.1:33306" \
  --db-proxy "postgres://:15432->127.0.0.1:35432" \
  --db-proxy "mongo://:27018->127.0.0.1:37017" \
  --db-proxy "redis://:16379->127.0.0.1:36379" \
  >"${WORK_DIR}/artifacts/record.log" 2>&1 &
RECORD_PID=$!

wait_tcp 127.0.0.1 18080 "record HTTP proxy"
curl -fsS "http://127.0.0.1:18080/users/1" -o "${WORK_DIR}/artifacts/record-response.json"

kill -TERM "${RECORD_PID}" >/dev/null 2>&1 || true
wait_process_exit "${RECORD_PID}" "record stage" 10
RECORD_PID=""

log "starting replay stage"
"${SHADIFF_BIN}" --config "${CONFIG_FILE}" replay \
  --session "${SESSION_NAME}" \
  --target http://127.0.0.1:18082 \
  --db-proxy "mysql://:13316->127.0.0.1:33306" \
  --db-proxy "postgres://:15442->127.0.0.1:35432" \
  --db-proxy "mongo://:27028->127.0.0.1:37017" \
  --db-proxy "redis://:16389->127.0.0.1:36379" \
  >"${WORK_DIR}/artifacts/replay.log" 2>&1 &
REPLAY_PID=$!
wait_log_marker "${WORK_DIR}/artifacts/replay.log" "Replay summary:" "${REPLAY_PID}" "replay stage" 60
kill -TERM "${REPLAY_PID}" >/dev/null 2>&1 || true
wait_process_exit "${REPLAY_PID}" "replay stage" 10
REPLAY_PID=""

log "starting diff stage"
"${SHADIFF_BIN}" --config "${CONFIG_FILE}" diff \
  --session "${SESSION_NAME}" \
  --output json \
  --output-file "${WORK_DIR}/artifacts/diff.json" \
  --fail-on none \
  >"${WORK_DIR}/artifacts/diff.log" 2>&1

log "generating HTML report"
"${SHADIFF_BIN}" --config "${CONFIG_FILE}" report \
  --session "${SESSION_NAME}" \
  --format html \
  --output "${WORK_DIR}/artifacts/report.html" \
  >"${WORK_DIR}/artifacts/report.log" 2>&1

if [[ "${ASSERT}" -eq 1 ]]; then
  log "asserting expected E2E artifacts and differences"
  assert_demo
fi

log "demo complete"
log "artifacts: ${WORK_DIR}/artifacts"
