#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8989}"
API_USER="${API_USER:-lcp_api_user}"
API_PASSWORD="${API_PASSWORD:-lcp_api_password}"

OUT_DIR="${OUT_DIR:-./.local/lcp-update-test}"
RESOURCE_DIR="${RESOURCE_DIR:-./.local/lcpserver/resources}"

LICENSE_SERVER_DIR="${LICENSE_SERVER_DIR:-$OUT_DIR/served-licenses}"
LICENSE_SERVER_BASE_URL="${LICENSE_SERVER_BASE_URL:-http://localhost:8991}"

PUBLICATION_ID="${PUBLICATION_ID:-$(python3 - <<'PY'
import uuid
print(uuid.uuid4())
PY
)}"

USER_ID="${USER_ID:-demo-user-001}"
USER_NAME="${USER_NAME:-Demo User}"
USER_EMAIL="${USER_EMAIL:-demo@example.org}"
PASSPHRASE="${PASSPHRASE:-123 456}"

PUBLICATION_TITLE="${PUBLICATION_TITLE:-Dummy EPUB Publication}"
PUBLICATION_FILENAME="${PUBLICATION_FILENAME:-demo-publication.epub}"

mkdir -p "$OUT_DIR" "$RESOURCE_DIR"

PUBLICATION_FILE="$RESOURCE_DIR/$PUBLICATION_FILENAME"
PUBLICATION_HREF="${BASE_URL%/}/resources/$PUBLICATION_FILENAME"

STATE_FILE="$OUT_DIR/state.env"
PUBLICATION_PAYLOAD="$OUT_DIR/publication-v1.json"
LICENSE_PAYLOAD="$OUT_DIR/license-request.json"
FRESH_LICENSE_PAYLOAD="$OUT_DIR/fresh-license-request.json"
LICENSE_FILE="$OUT_DIR/license-v1.lcpl"
STATUS_FILE="$OUT_DIR/status-v1.json"

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: $1 is required." >&2
    exit 1
  }
}

require python3
require curl

publish_latest_lcpl() {
  local license_id="$1"
  local lcpl_file="$2"

  mkdir -p "$LICENSE_SERVER_DIR"

  cp "$lcpl_file" "$LICENSE_SERVER_DIR/$license_id.lcpl"
  touch "$LICENSE_SERVER_DIR/$license_id.lcpl"

  echo "Published latest LCPL:"
  echo "  $LICENSE_SERVER_DIR/$license_id.lcpl"
  echo "  $LICENSE_SERVER_BASE_URL/licenses/$license_id"
}

echo "Generating dummy EPUB v1: $PUBLICATION_FILE"

python3 - "$PUBLICATION_FILE" "$PUBLICATION_ID" "v1" <<'PY'
from pathlib import Path
from zipfile import ZipFile, ZIP_STORED, ZIP_DEFLATED
from datetime import datetime, timezone
import sys

out_path = Path(sys.argv[1])
publication_id = sys.argv[2]
version = sys.argv[3]
modified = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

mimetype = "application/epub+zip"

container_xml = """<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0"
  xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/content.opf"
      media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
"""

content_opf = f"""<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf"
         version="3.0"
         unique-identifier="pub-id"
         xml:lang="en">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="pub-id">urn:uuid:{publication_id}</dc:identifier>
    <dc:title>Dummy EPUB Publication</dc:title>
    <dc:language>en</dc:language>
    <dc:creator>LCP Server quickstart</dc:creator>
    <meta property="dcterms:modified">{modified}</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>
"""

nav_xhtml = """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml"
      xmlns:epub="http://www.idpf.org/2007/ops"
      lang="en"
      xml:lang="en">
  <head>
    <title>Navigation</title>
  </head>
  <body>
    <nav epub:type="toc" id="toc">
      <h1>Table of Contents</h1>
      <ol>
        <li><a href="chapter.xhtml">Dummy Chapter</a></li>
      </ol>
    </nav>
  </body>
</html>
"""

chapter_xhtml = f"""<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml"
      lang="en"
      xml:lang="en">
  <head>
    <title>Dummy Chapter</title>
  </head>
  <body>
    <section>
      <h1>Dummy EPUB Publication</h1>
      <p>This is version {version}.</p>
      <p>This EPUB has exactly one XHTML document in the spine.</p>
    </section>
  </body>
</html>
"""

out_path.parent.mkdir(parents=True, exist_ok=True)

with ZipFile(out_path, "w") as zf:
    zf.writestr("mimetype", mimetype, compress_type=ZIP_STORED)
    zf.writestr("META-INF/container.xml", container_xml, compress_type=ZIP_DEFLATED)
    zf.writestr("EPUB/content.opf", content_opf, compress_type=ZIP_DEFLATED)
    zf.writestr("EPUB/nav.xhtml", nav_xhtml, compress_type=ZIP_DEFLATED)
    zf.writestr("EPUB/chapter.xhtml", chapter_xhtml, compress_type=ZIP_DEFLATED)
PY

read -r PUBLICATION_SIZE PUBLICATION_CHECKSUM < <(
  python3 - "$PUBLICATION_FILE" <<'PY'
from pathlib import Path
import hashlib
import sys

path = Path(sys.argv[1])
data = path.read_bytes()
print(path.stat().st_size, hashlib.sha256(data).hexdigest())
PY
)

ENCRYPTION_KEY="$(python3 - <<'PY'
import base64
import os
print(base64.b64encode(os.urandom(32)).decode("ascii"))
PY
)"

# TODO: static
# "123 456"
PASS_HASH="4981AA0A50D563040519E9032B5D74367B1D129E239A1BA82667A57333866494"

read -r START END < <(
  python3 - <<'PY'
from datetime import datetime, timedelta, timezone

start = datetime.now(timezone.utc)
end = start + timedelta(days=14)

print(
    start.strftime("%Y-%m-%dT%H:%M:%SZ"),
    end.strftime("%Y-%m-%dT%H:%M:%SZ"),
)
PY
)

cat > "$PUBLICATION_PAYLOAD" <<EOF
{
  "uuid": "$PUBLICATION_ID",
  "title": "$PUBLICATION_TITLE",
  "encryption_key": "$ENCRYPTION_KEY",
  "href": "$PUBLICATION_HREF",
  "content_type": "application/epub+zip",
  "size": $PUBLICATION_SIZE,
  "checksum": "$PUBLICATION_CHECKSUM"
}
EOF

cat > "$LICENSE_PAYLOAD" <<EOF
{
  "publication_id": "$PUBLICATION_ID",
  "user_id": "$USER_ID",
  "user_name": "$USER_NAME",
  "user_email": "$USER_EMAIL",
  "user_encrypted": ["name", "email"],
  "start": "$START",
  "end": "$END",
  "copy": 20000,
  "print": 100,
  "profile": "http://readium.org/lcp/basic-profile",
  "text_hint": "Passphrase for this demo loan: $PASSPHRASE",
  "pass_hash": "$PASS_HASH"
}
EOF

cat > "$FRESH_LICENSE_PAYLOAD" <<EOF
{
  "user_id": "$USER_ID",
  "user_name": "$USER_NAME",
  "user_email": "$USER_EMAIL",
  "user_encrypted": ["name", "email"],
  "profile": "http://readium.org/lcp/basic-profile",
  "text_hint": "Passphrase for this demo loan: $PASSPHRASE",
  "pass_hash": "$PASS_HASH"
}
EOF

echo "Creating publication: $PUBLICATION_ID"

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "${BASE_URL%/}/publications/" \
  --data @"$PUBLICATION_PAYLOAD" \
  > "$OUT_DIR/publication-v1-response.json"

echo "Creating loan/license"

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "${BASE_URL%/}/licenses/" \
  --data @"$LICENSE_PAYLOAD" \
  > "$LICENSE_FILE"

LICENSE_ID="$(python3 - "$LICENSE_FILE" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

print(data["id"])
PY
)"

publish_latest_lcpl "$LICENSE_ID" "$LICENSE_FILE"

echo "Fetching LSD status document"

curl -fsS \
  "${BASE_URL%/}/status/$LICENSE_ID" \
  > "$STATUS_FILE"

echo "Verifying initial license publication link"

python3 - "$LICENSE_FILE" "$PUBLICATION_SIZE" "$PUBLICATION_CHECKSUM" <<'PY'
import json
import sys

license_file, expected_length, expected_hash = sys.argv[1], int(sys.argv[2]), sys.argv[3]

with open(license_file, "r", encoding="utf-8") as f:
    data = json.load(f)

publication_links = [link for link in data.get("links", []) if link.get("rel") == "publication"]
if not publication_links:
    raise SystemExit("No rel=publication link found in license.")

publication = publication_links[0]

if publication.get("length") != expected_length:
    raise SystemExit(f"Unexpected publication length: {publication.get('length')} != {expected_length}")

if publication.get("hash") != expected_hash:
    raise SystemExit(f"Unexpected publication hash: {publication.get('hash')} != {expected_hash}")

print("OK: initial license has expected publication length/hash.")
PY

cat > "$STATE_FILE" <<EOF
BASE_URL='$BASE_URL'
API_USER='$API_USER'
API_PASSWORD='$API_PASSWORD'
OUT_DIR='$OUT_DIR'
RESOURCE_DIR='$RESOURCE_DIR'
PUBLICATION_ID='$PUBLICATION_ID'
PUBLICATION_TITLE='$PUBLICATION_TITLE'
PUBLICATION_FILENAME='$PUBLICATION_FILENAME'
PUBLICATION_FILE='$PUBLICATION_FILE'
PUBLICATION_HREF='$PUBLICATION_HREF'
ENCRYPTION_KEY='$ENCRYPTION_KEY'
USER_ID='$USER_ID'
USER_NAME='$USER_NAME'
USER_EMAIL='$USER_EMAIL'
PASSPHRASE='$PASSPHRASE'
PASS_HASH='$PASS_HASH'
LICENSE_ID='$LICENSE_ID'
INITIAL_PUBLICATION_SIZE='$PUBLICATION_SIZE'
INITIAL_PUBLICATION_CHECKSUM='$PUBLICATION_CHECKSUM'
EOF

cat <<EOF

Done.

State:
  $STATE_FILE

Publication ID:
  $PUBLICATION_ID

License ID:
  $LICENSE_ID

Initial publication length:
  $PUBLICATION_SIZE

Initial publication hash:
  $PUBLICATION_CHECKSUM

EOF
