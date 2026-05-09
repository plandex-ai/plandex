#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MODULES=(cli server shared)

is_race_supported() {
  local goos goarch
  goos="${GOOS:-$(go env GOOS)}"
  goarch="${GOARCH:-$(go env GOARCH)}"

  case "${goos}/${goarch}" in
    linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

main() {
  local race_enabled=0

  if [[ "${PLDX_FORCE_NO_RACE:-0}" == "1" ]]; then
    echo "[plandex-test] PLDX_FORCE_NO_RACE=1 -> running without -race"
  elif is_race_supported; then
    race_enabled=1
    echo "[plandex-test] race detector supported -> running with -race"
  else
    local goos goarch
    goos="${GOOS:-$(go env GOOS)}"
    goarch="${GOARCH:-$(go env GOARCH)}"
    echo "[plandex-test] race detector not supported on ${goos}/${goarch} -> running without -race"
  fi

  for module in "${MODULES[@]}"; do
    echo "== app/${module} =="
    pushd "${ROOT_DIR}/app/${module}" >/dev/null
    if [[ "${race_enabled}" == "1" ]]; then
      go test -race ./...
    else
      go test ./...
    fi
    popd >/dev/null
  done
}

main "$@"
