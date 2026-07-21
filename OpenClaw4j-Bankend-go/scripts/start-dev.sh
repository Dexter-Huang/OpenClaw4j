#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd -- "$script_dir/.." && pwd)"
binary_dir="$project_root/bin"
binary_path="$binary_dir/openclaw4j-backend-go.exe"
rebuild=false
online=false

usage() {
  cat <<'EOF'
Usage: ./scripts/start-dev.sh [--rebuild] [--online]

  --rebuild  强制重新构建 Go 后端。
  --online   允许 Go 在构建时访问 GOPROXY 和 GOSUMDB。
EOF
}

while (($# > 0)); do
  case "$1" in
    --rebuild)
      rebuild=true
      ;;
    --online)
      online=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'Unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

cd -- "$project_root"

# .env 在运行时读取；只有参与 go build 的源码和模块清单变更才需要重新链接。
needs_build="$rebuild"
if [[ "$needs_build" != true && ! -f "$binary_path" ]]; then
  needs_build=true
fi

if [[ "$needs_build" != true ]]; then
  if find "$project_root" -path "$binary_dir" -prune -o -type f -name '*.go' ! -name '*_test.go' -newer "$binary_path" -print -quit | grep -q . \
    || [[ "$project_root/go.mod" -nt "$binary_path" ]] \
    || [[ "$project_root/go.sum" -nt "$binary_path" ]]; then
    needs_build=true
  fi
fi

if [[ "$needs_build" == true ]]; then
  mkdir -p -- "$binary_dir"
  printf 'Building Go backend...\n'

  if [[ "$online" == true ]]; then
    go build -o "$binary_path" ./cmd/openclaw4j-backend-go
  elif ! GOPROXY=off GOSUMDB=off go build -o "$binary_path" ./cmd/openclaw4j-backend-go; then
    printf 'Offline build failed. Run again with --online if the module cache is incomplete.\n' >&2
    exit 1
  fi
else
  printf 'Using current Go backend binary.\n'
fi

printf 'Starting Go backend: %s\n' "$binary_path"
exec "$binary_path"
