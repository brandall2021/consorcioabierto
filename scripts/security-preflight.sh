#!/usr/bin/env bash
set -euo pipefail

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

require() {
	local name="$1"
	if [[ -z "${!name:-}" ]]; then
		fail "$name es obligatoria"
	fi
}

if [[ "${APP_ENV:-local}" == "production" ]]; then
	require DATABASE_URL
	require DATABASE_URL_ADMIN
	require JWT_PRIVATE_KEY

	for item in STORAGE_DRIVER MAIL_DRIVER PSP_DRIVER SCAN_DRIVER; do
		value="${!item:-}"
		case "$value" in
			mock|mailpit|*mock*)
				fail "$item=$value está prohibido en production"
				;;
		esac
	done
fi

printf 'security preflight ok\n'
