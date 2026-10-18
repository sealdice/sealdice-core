#!/usr/bin/env bash
set -euo pipefail

# 与两个 workflow 安装的 v2.15.0 一致，使用内置 schema 校验配置。
golangci-lint config verify
golangci-lint run --timeout 10m
