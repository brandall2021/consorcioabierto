#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

bindir="$tmpdir/bin"
mkdir -p "$bindir"

cat > "$bindir/pg_dump" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$PG_DUMP_ARGS_FILE"
printf 'backup-bytes' > "$PG_DUMP_OUTPUT_FILE"
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
export PG_DUMP_ARGS_FILE="$tmpdir/pg_dump.args"
export PG_DUMP_OUTPUT_FILE="$tmpdir/backup.dump"
export PG_RESTORE_ARGS_FILE="$tmpdir/pg_restore.args"

backup_output="$(bash scripts/db-maintenance.sh backup)"
test -f "$backup_output"
test "$(cat "$backup_output")" = 'backup-bytes'

grep -q -- '--format=custom' "$PG_DUMP_ARGS_FILE"
grep -q -- '--file' "$PG_DUMP_ARGS_FILE"

restore_output="$(bash scripts/db-maintenance.sh restore "$backup_output")"
test "$restore_output" = "restored:$backup_output"
grep -q -- '--clean' "$PG_RESTORE_ARGS_FILE"
grep -q -- '--if-exists' "$PG_RESTORE_ARGS_FILE"
