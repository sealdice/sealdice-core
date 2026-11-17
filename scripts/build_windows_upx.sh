#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT_DIR/temp/build"
SHORT_HASH="$(git -C "$ROOT_DIR" rev-parse --short HEAD)"
BIN_NAME="sealdice-core_windows_amd64_${SHORT_HASH}.exe"
BIN_PATH="$OUT_DIR/$BIN_NAME"
UPX_PATH="$OUT_DIR/sealdice-core_windows_amd64_${SHORT_HASH}.upx.exe"

mkdir -p "$OUT_DIR"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags='-s -w' -o "$BIN_PATH" .

cp "$BIN_PATH" "$UPX_PATH"
upx --best --lzma "$UPX_PATH"

ls -lh "$BIN_PATH" "$UPX_PATH"
