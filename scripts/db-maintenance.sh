#!/usr/bin/env bash
set -euo pipefail

usage() {
	printf 'Uso: %s backup | restore <archivo>\n' "$0" >&2
	exit 1
}

require_env() {
	local name="$1"
	if [[ -z "${!name:-}" ]]; then
		printf '%s es obligatoria\n' "$name" >&2
		exit 1
	fi
}

timestamp() {
	if [[ -n "${BACKUP_TIMESTAMP:-}" ]]; then
		printf '%s' "$BACKUP_TIMESTAMP"
		return
	fi
	date -u +%Y%m%dT%H%M%SZ
}

backup() {
	require_env DATABASE_URL_ADMIN
	local dir="${BACKUP_DIR:-./backups}"
	mkdir -p "$dir"
	local file="$dir/consorcioabierto-$(timestamp).dump"
	pg_dump --format=custom --file "$file" "$DATABASE_URL_ADMIN"
	printf '%s\n' "$file"
}

restore() {
	require_env DATABASE_URL_ADMIN
	local file="${1:-}"
	if [[ -z "$file" ]]; then
		usage
	fi
	if [[ ! -f "$file" ]]; then
		printf 'Archivo no encontrado: %s\n' "$file" >&2
		exit 1
	fi
	pg_restore --clean --if-exists --no-owner --dbname "$DATABASE_URL_ADMIN" "$file"
	printf 'restored:%s\n' "$file"
}

case "${1:-}" in
	backup)
		backup
		;;
	restore)
		shift
		restore "${1:-}"
		;;
	*)
		usage
		;;
esac
