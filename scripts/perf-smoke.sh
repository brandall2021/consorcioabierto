#!/usr/bin/env bash
set -euo pipefail

base_url='http://localhost:8090'
repetitions=5

usage() {
	printf 'Uso: %s [--base-url URL] [--repetitions N]\n' "$0" >&2
	exit 1
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--base-url)
			base_url="${2:-}"
			shift 2
			;;
		--repetitions)
			repetitions="${2:-}"
			shift 2
			;;
		-h|--help)
			usage
			;;
		*)
			usage
			;;
	esac
done

paths=(/healthz /metrics /api/v1/me /api/v1/portal)

printf 'perf smoke base_url=%s repeticiones=%s\n' "$base_url" "$repetitions"
for path in "${paths[@]}"; do
	total=0
	status=0
	for ((i = 0; i < repetitions; i++)); do
		read -r time_total http_code < <(curl -sS -o /dev/null -w '%{time_total} %{http_code}\n' "$base_url$path")
		status="$http_code"
		total="$(awk -v acc="$total" -v v="$time_total" 'BEGIN { printf "%.6f", acc + v }')"
	done
	avg="$(awk -v total="$total" -v reps="$repetitions" 'BEGIN { printf "%.3f", (total / reps) * 1000 }')"
	printf '%s avg_ms=%s status=%s\n' "$path" "$avg" "$status"
done
