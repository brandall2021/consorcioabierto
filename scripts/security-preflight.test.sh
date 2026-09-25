#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

run_ok() {
	APP_ENV=production \
	DATABASE_URL='postgres://user:pass@localhost:5432/db' \
	DATABASE_URL_ADMIN='postgres://user:pass@localhost:5432/db' \
	JWT_PRIVATE_KEY='key' \
	STORAGE_DRIVER=s3 \
	MAIL_DRIVER=smtp \
	PSP_DRIVER=mercadopago \
	SCAN_DRIVER=s3 \
	bash scripts/security-preflight.sh >"$tmpdir/out.ok"
	grep -q 'security preflight ok' "$tmpdir/out.ok"
}

run_fail_mock() {
	if APP_ENV=production \
		DATABASE_URL='postgres://user:pass@localhost:5432/db' \
		DATABASE_URL_ADMIN='postgres://user:pass@localhost:5432/db' \
		JWT_PRIVATE_KEY='key' \
		STORAGE_DRIVER=mock \
		MAIL_DRIVER=smtp \
		PSP_DRIVER=mercadopago \
		SCAN_DRIVER=s3 \
		bash scripts/security-preflight.sh >"$tmpdir/out.fail" 2>&1; then
		printf 'debía fallar con driver simulado\n' >&2
		exit 1
	fi
	grep -q 'STORAGE_DRIVER' "$tmpdir/out.fail"
}

run_fail_missing_secret() {
	if APP_ENV=production \
		DATABASE_URL='postgres://user:pass@localhost:5432/db' \
		DATABASE_URL_ADMIN='postgres://user:pass@localhost:5432/db' \
		STORAGE_DRIVER=s3 \
		MAIL_DRIVER=smtp \
		PSP_DRIVER=mercadopago \
		SCAN_DRIVER=s3 \
		bash scripts/security-preflight.sh >"$tmpdir/out.secret" 2>&1; then
		printf 'debía fallar por JWT_PRIVATE_KEY\n' >&2
		exit 1
	fi
	grep -q 'JWT_PRIVATE_KEY' "$tmpdir/out.secret"
}

run_ok
run_fail_mock
run_fail_missing_secret
