#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

bindir="$tmpdir/bin"
mkdir -p "$bindir"

cat > "$bindir/pg_dump" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
outfile=''
while [[ $# -gt 0 ]]; do
	case "$1" in
		--file)
			outfile="${2:-}"
			shift 2
			;;
		*)
			shift
			;;
	esac
done
printf 'backup-bytes' > "$outfile"
EOF
chmod +x "$bindir/pg_dump"

cat > "$bindir/pg_restore" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$PG_RESTORE_ARGS_FILE"
EOF
chmod +x "$bindir/pg_restore"

export PATH="$bindir:$PATH"
export DATABASE_URL_ADMIN='postgres://user:pass@localhost:5432/db?sslmode=disable'
export BACKUP_DIR="$tmpdir/backups"
export BACKUP_TIMESTAMP='2026-09-16T120000Z'
export PG_DUMP_OUTPUT_FILE="$tmpdir/backup.dump"
export PG_RESTORE_ARGS_FILE="$tmpdir/pg_restore.args"

output="$(bash scripts/restore-rehearsal.sh --backup-name rehearsal.dump --target-database-url 'postgres://user:pass@localhost:5432/restore?sslmode=disable')"

grep -q 'rehearsal ok' <<<"$output"
grep -q 'rehearsal.dump' <<<"$output"
grep -q -- '--dbname postgres://user:pass@localhost:5432/restore?sslmode=disable' "$PG_RESTORE_ARGS_FILE"
