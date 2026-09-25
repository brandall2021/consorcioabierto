#!/usr/bin/env bash
set -euo pipefail

backup_name='restore-rehearsal.dump'
target_database_url=''

usage() {
	printf 'Uso: %s --backup-name archivo.dump --target-database-url postgres://...\n' "$0" >&2
	exit 1
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--backup-name)
			backup_name="${2:-}"
			shift 2
			;;
		--target-database-url)
			target_database_url="${2:-}"
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

if [[ -z "$target_database_url" ]]; then
	usage
fi

backup_file="$(bash scripts/db-maintenance.sh backup)"
restore_dir="${RESTORE_REHEARSAL_DIR:-./backups/rehearsal}"
mkdir -p "$restore_dir"
target_backup="$restore_dir/$backup_name"
cp "$backup_file" "$target_backup"
	pg_restore --clean --if-exists --no-owner --dbname "$target_database_url" "$target_backup"
printf 'rehearsal ok: %s -> %s\n' "$backup_file" "$target_backup"
