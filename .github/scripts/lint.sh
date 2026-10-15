#!/usr/bin/env bash
set -euo pipefail

# 与两个 workflow 安装的 v2.4.0 一致。先可靠下载，再离线校验，
# 避免 config verify 内置的 2 秒网站请求超时；不跳过配置校验。
schema="$(mktemp)"
trap 'rm -f "$schema"' EXIT
curl --fail --silent --show-error --location --retry 3 --retry-all-errors \
  --connect-timeout 10 --max-time 60 \
  'https://raw.githubusercontent.com/golangci/golangci-lint/v2.4.0/jsonschema/golangci.jsonschema.json' \
  --output "$schema"
printf '%s  %s\n' '1afda373ebb266e83d077acc2cebf0a8411d411cd69cde56537977562b488575' "$schema" | sha256sum --check --status

golangci-lint config verify --schema "$schema"
golangci-lint run --timeout 10m
