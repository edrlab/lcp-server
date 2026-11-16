#!/usr/bin/env bash
set -euo pipefail

# Quickstart for local testing only.
# This builds lcpserver, creates a local SQLite config, and runs the server.
#
# Usage:
#   ./scripts/quickstart-lcpserver-sqlite.sh
#
# Optional overrides:
#   LCP_PORT=8989 ./scripts/quickstart-lcpserver-sqlite.sh
#   LCP_HOME=/tmp/lcp ./scripts/quickstart-lcpserver-sqlite.sh
#   RECREATE_CONFIG=1 ./scripts/quickstart-lcpserver-sqlite.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${SCRIPT_DIR}/go.mod" ]; then
  REPO_ROOT="${SCRIPT_DIR}"
elif [ -f "${SCRIPT_DIR}/../go.mod" ]; then
  REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
else
  echo "Error: this script must be run from the lcp-server repository, or from scripts/ inside it." >&2
  exit 1
fi

cd "${REPO_ROOT}"

: "${LCP_PORT:=8989}"
: "${LCP_LICENSE_SERVER_PORT:=8991}"
: "${LCP_PUBLIC_BASE_URL:=http://localhost:${LCP_PORT}}"
: "${LCP_LICENSE_SERVER_BASE_URL:=http://localhost:${LCP_LICENSE_SERVER_PORT}}"
: "${LCP_HOME:=${REPO_ROOT}/.local/lcpserver}"
: "${LCP_API_USER:=lcp_api_user}"
: "${LCP_API_PASSWORD:=lcp_api_password}"
: "${LCP_ADMIN_USER:=admin}"
: "${LCP_ADMIN_PASSWORD:=supersecret}"
: "${RECREATE_CONFIG:=0}"

BIN_DIR="${REPO_ROOT}/bin"
BIN_PATH="${BIN_DIR}/lcpserver"

CONFIG_DIR="${LCP_HOME}/config"
DB_DIR="${LCP_HOME}/db"
RESOURCES_DIR="${LCP_HOME}/resources"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
DB_FILE="${DB_DIR}/lcp.sqlite"

CERT_FILE="${REPO_ROOT}/config/cert-edrlab-test.pem"
PRIVATE_KEY_FILE="${REPO_ROOT}/config/privkey-edrlab-test.pem"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v gcc >/dev/null 2>&1 && ! command -v cc >/dev/null 2>&1; then
  echo "Error: SQLite builds require CGO and a C compiler such as gcc or cc." >&2
  exit 1
fi

if [ ! -f "${CERT_FILE}" ] || [ ! -f "${PRIVATE_KEY_FILE}" ]; then
  echo "Error: test certificate files were not found in ${REPO_ROOT}/config/." >&2
  echo "Expected:" >&2
  echo "  ${CERT_FILE}" >&2
  echo "  ${PRIVATE_KEY_FILE}" >&2
  exit 1
fi

mkdir -p "${BIN_DIR}" "${CONFIG_DIR}" "${DB_DIR}" "${RESOURCES_DIR}"

echo "Building lcpserver..."
CGO_ENABLED=1 go build -o "${BIN_PATH}" ./cmd/lcpserver
if [ ! -f "${CONFIG_FILE}" ] || [ "${RECREATE_CONFIG}" = "1" ]; then
  if command -v openssl >/dev/null 2>&1; then
    JWT_SECRET="$(openssl rand -base64 48)"
  else
    JWT_SECRET="quickstart-jwt-secret-change-me"
  fi

  echo "Creating local config: ${CONFIG_FILE}"

  cat > "${CONFIG_FILE}" <<EOF
# Local quickstart configuration for lcpserver.
# For test/development only. Do not use these credentials in production.

log_level: "debug"
public_base_url: "${LCP_PUBLIC_BASE_URL}"
port: ${LCP_PORT}

# The storage code expects a URI-like DSN.
# Use sqlite3 here so SQLite-specific initialization is applied.
dsn: "sqlite3://${DB_FILE}"

access:
  username: "${LCP_API_USER}"
  password: "${LCP_API_PASSWORD}"

certificate:
  cert: "${CERT_FILE}"
  private_key: "${PRIVATE_KEY_FILE}"

license:
  provider: "${LCP_PUBLIC_BASE_URL}"
  profile: "http://readium.org/lcp/basic-profile"
  hint_link: "${LCP_PUBLIC_BASE_URL}/hint"

status:
  fresh_license_link: "${LCP_LICENSE_SERVER_BASE_URL}/licenses/{license_id}"
  allow_renew_on_expired_licenses: true
  renew_default_days: 30
  renew_max_days: 365
  renew_link: "${LCP_PUBLIC_BASE_URL}/renew/{license_id}"

dashboard:
  excessive_sharing_threshold: 5
  limit_to_last_12_months: true

jwt:
  secret_key: "${JWT_SECRET}"
  admin:
    ${LCP_ADMIN_USER}: "${LCP_ADMIN_PASSWORD}"

resources: "${RESOURCES_DIR}"
EOF
else
  echo "Using existing config: ${CONFIG_FILE}"
  echo "Set RECREATE_CONFIG=1 to regenerate it."
fi

cat ${CONFIG_FILE}

cat <<EOF

LCP Server quickstart

Binary:    ${BIN_PATH}
Config:    ${CONFIG_FILE}
SQLite DB: ${DB_FILE}
Resources: ${RESOURCES_DIR}
Base URL:  ${LCP_PUBLIC_BASE_URL}

Dashboard credentials:
  ${LCP_ADMIN_USER} / ${LCP_ADMIN_PASSWORD}

API basic auth:
  ${LCP_API_USER} / ${LCP_API_PASSWORD}

Starting server...
Press Ctrl+C to stop.

EOF

exec env LCPSERVER_CONFIG="${CONFIG_FILE}" "${BIN_PATH}"
