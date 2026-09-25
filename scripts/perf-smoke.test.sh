#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

bindir="$tmpdir/bin"
mkdir -p "$bindir"

cat > "$bindir/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url="${@: -1}"
case "$url" in
  */healthz) printf '0.010 200\n' ;;
  */metrics) printf '0.020 200\n' ;;
  */api/v1/me) printf '0.030 200\n' ;;
  */api/v1/portal) printf '0.040 200\n' ;;
  *) printf '0.050 404\n' ;;
esac
EOF
chmod +x "$bindir/curl"

export PATH="$bindir:$PATH"

output="$(bash scripts/perf-smoke.sh --base-url http://example.test --repetitions 3)"

grep -q 'perf smoke' <<<"$output"
grep -q '/healthz' <<<"$output"
grep -q '/api/v1/portal' <<<"$output"
grep -q 'repeticiones=3' <<<"$output"
