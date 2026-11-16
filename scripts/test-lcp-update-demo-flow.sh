#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${SCRIPT_DIR}/../go.mod" ]; then
  REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
elif [ -f "./go.mod" ]; then
  REPO_ROOT="$(pwd)"
else
  echo "Error: this script must be run from the lcp-server repository or scripts/ inside it." >&2
  exit 1
fi

cd "$REPO_ROOT"

: "${BASE_URL:=http://localhost:8989}"
: "${OUT_DIR:=./.local/lcp-update-test}"
: "${LICENSE_SERVER_BASE_URL:=http://localhost:8991}"
: "${LICENSE_SERVER_DIR:=$OUT_DIR/served-licenses}"
: "${UPDATE_SCRIPT:=test-update-encrypted-publication-same-content-key.sh}"
: "${PAUSE_BEFORE_UPDATE:=1}"

STATE_FILE="$OUT_DIR/state.env"
CREATE_SCRIPT="${SCRIPT_DIR}/test-create-publication-loan.sh"

case "$UPDATE_SCRIPT" in
  /*) UPDATE_SCRIPT_PATH="$UPDATE_SCRIPT" ;;
  *) UPDATE_SCRIPT_PATH="${SCRIPT_DIR}/${UPDATE_SCRIPT}" ;;
esac

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: $1 is required." >&2
    exit 1
  }
}

display_path() {
  local path="$1"

  if command -v cygpath >/dev/null 2>&1; then
    cygpath -w "$path"
    return
  fi

  local dir
  local base
  dir="$(cd "$(dirname "$path")" && pwd)"
  base="$(basename "$path")"
  printf "%s/%s\n" "$dir" "$base"
}

require bash
require curl

if [ ! -f "$CREATE_SCRIPT" ]; then
  echo "Error: create script not found: $CREATE_SCRIPT" >&2
  exit 1
fi

if [ ! -f "$UPDATE_SCRIPT_PATH" ]; then
  echo "Error: update script not found: $UPDATE_SCRIPT_PATH" >&2
  exit 1
fi

cat <<EOF
LCP publication update demo flow

Expected services:
  LCP server:        ${BASE_URL%/}
  Fresh LCPL server: ${LICENSE_SERVER_BASE_URL%/}

If they are not running yet, start them in two other terminals:
  RECREATE_CONFIG=1 ./scripts/quickstart-lcpserver-sqlite.sh
  LICENSE_SERVER_DIR=$LICENSE_SERVER_DIR LICENSE_SERVER_PORT=8991 ./scripts/license-file-server.py

EOF

if ! curl -fsS "${LICENSE_SERVER_BASE_URL%/}/health" >/dev/null 2>&1; then
  cat <<EOF
Warning: the fresh-LCPL server health check failed.
Thorium will need this server during the LSD refresh:
  ${LICENSE_SERVER_BASE_URL%/}/licenses/{license_id}

EOF
fi

echo "Step 1/2: creating the v1 publication and initial loan..."
BASE_URL="$BASE_URL" \
OUT_DIR="$OUT_DIR" \
LICENSE_SERVER_BASE_URL="$LICENSE_SERVER_BASE_URL" \
LICENSE_SERVER_DIR="$LICENSE_SERVER_DIR" \
bash "$CREATE_SCRIPT"

if [ ! -f "$STATE_FILE" ]; then
  echo "Error: expected state file was not created: $STATE_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$STATE_FILE"

INITIAL_LCPL_FILE="$OUT_DIR/license-v1.lcpl"
INITIAL_LCPL_DISPLAY="$(display_path "$INITIAL_LCPL_FILE")"
FRESH_LCPL_URL="${LICENSE_SERVER_BASE_URL%/}/licenses/$LICENSE_ID"

cat <<EOF

Demo pause

Import this initial LCPL in Thorium:
  $INITIAL_LCPL_DISPLAY

Loan details:
  License ID:     $LICENSE_ID
  Publication ID: $PUBLICATION_ID
  Passphrase:     $PASSPHRASE

Fresh LCPL URL used by LSD:
  $FRESH_LCPL_URL

Open the publication in Thorium now and confirm that the v1 content is readable.
EOF

if [ "$PAUSE_BEFORE_UPDATE" != "0" ]; then
  printf "\nPress Enter when you are ready to publish the v2 update, or Ctrl+C to stop here. "
  read -r _
else
  echo
  echo "PAUSE_BEFORE_UPDATE=0, continuing without waiting."
fi

echo
echo "Step 2/2: publishing the v2 publication update..."
STATE_FILE="$STATE_FILE" \
LICENSE_SERVER_BASE_URL="$LICENSE_SERVER_BASE_URL" \
LICENSE_SERVER_DIR="$LICENSE_SERVER_DIR" \
bash "$UPDATE_SCRIPT_PATH"

FRESH_LCPL_DISPLAY="$(display_path "$OUT_DIR/fresh-license-v2.lcpl")"
UPDATED_STATUS_DISPLAY="$(display_path "$OUT_DIR/status-v2.json")"
STATE_FILE_DISPLAY="$(display_path "$STATE_FILE")"

cat <<EOF

Demo update completed.

Now refresh or reopen the same loan in Thorium. It should fetch the fresh LCPL,
detect the changed rel="publication" length/hash, download the v2 archive, and
keep the loan readable with the updated content.

Useful artifacts:
  Initial LCPL:  $INITIAL_LCPL_DISPLAY
  Fresh LCPL:   $FRESH_LCPL_DISPLAY
  Updated LSD:  $UPDATED_STATUS_DISPLAY
  State file:   $STATE_FILE_DISPLAY

EOF
