#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

bin="$PWD/bin/ns-apns"
old="$(pgrep -f "$bin run" || true)"
if [[ -n "$old" ]]; then
  echo "停掉旧进程 $old"
  kill $old || true
  sleep 0.3
fi

make build VERSION=dev
echo "设置页  http://127.0.0.1:8787/?debug"
exec "$bin" run "$@"
