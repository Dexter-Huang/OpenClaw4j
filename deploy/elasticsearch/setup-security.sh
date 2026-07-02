#!/usr/bin/env bash
set -euo pipefail

ES_URL="${ES_URL:-http://elasticsearch:9200}"

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

wait_for_elasticsearch() {
  until curl -fsS -u "elastic:${ELASTIC_PASSWORD}" "${ES_URL}/_security/_authenticate" >/dev/null; do
    sleep 2
  done
}

es_api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"

  if [ -n "$body" ]; then
    curl -fsS -u "elastic:${ELASTIC_PASSWORD}" \
      -H "Content-Type: application/json" \
      -X "$method" \
      "${ES_URL}${path}" \
      -d "$body" >/dev/null
  else
    curl -fsS -u "elastic:${ELASTIC_PASSWORD}" \
      -X "$method" \
      "${ES_URL}${path}" >/dev/null
  fi
}

KIBANA_PASSWORD_JSON="$(json_escape "$KIBANA_SYSTEM_PASSWORD")"
LOGSTASH_PASSWORD_JSON="$(json_escape "$LOGSTASH_INTERNAL_PASSWORD")"

wait_for_elasticsearch

es_api POST "/_security/user/kibana_system/_password" \
  "{\"password\":\"${KIBANA_PASSWORD_JSON}\"}"

es_api PUT "/_security/role/openclaw4j_logstash_writer" '{
  "cluster": ["monitor"],
  "indices": [
    {
      "names": ["openclaw4j-logs-*"],
      "privileges": ["auto_configure", "create_index", "create", "write", "view_index_metadata"]
    }
  ]
}'

es_api PUT "/_security/user/logstash_internal" \
  "{\"password\":\"${LOGSTASH_PASSWORD_JSON}\",\"roles\":[\"openclaw4j_logstash_writer\"],\"full_name\":\"OpenClaw4j Logstash writer\"}"
