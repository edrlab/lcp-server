#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="${STATE_FILE:-./.local/lcp-update-test/state.env}"

if [ ! -f "$STATE_FILE" ]; then
  echo "Error: state file not found: $STATE_FILE" >&2
  echo "Run scripts/test-create-publication-loan.sh first, or set STATE_FILE=/path/to/state.env." >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$STATE_FILE"

UPDATED_PUBLICATION_PAYLOAD="$OUT_DIR/publication-v2.json"
UPDATED_PUBLICATION_RESPONSE="$OUT_DIR/publication-v2-response.json"
UPDATED_STATUS_FILE="$OUT_DIR/status-v2.json"
FRESH_LICENSE_FILE="$OUT_DIR/fresh-license-v2.lcpl"

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: $1 is required." >&2
    exit 1
  }
}

require python3
require curl

echo "Regenerating EPUB v2 with changed content: $PUBLICATION_FILE"

python3 - "$PUBLICATION_FILE" "$PUBLICATION_ID" "v2" <<'PY'
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
      <p>This file intentionally has different bytes from v1.</p>
      <p>The publication record must be updated with the new ZIP byte length and SHA-256 hash.</p>
      <p>Extra payload to force a different ZIP length: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</p>
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

read -r UPDATED_PUBLICATION_SIZE UPDATED_PUBLICATION_CHECKSUM < <(
  python3 - "$PUBLICATION_FILE" <<'PY'
from pathlib import Path
import hashlib
import sys

path = Path(sys.argv[1])
data = path.read_bytes()
print(path.stat().st_size, hashlib.sha256(data).hexdigest())
PY
)

if [ "$UPDATED_PUBLICATION_SIZE" = "${INITIAL_PUBLICATION_SIZE:-}" ]; then
  echo "Warning: updated size is identical to initial size." >&2
fi

if [ "$UPDATED_PUBLICATION_CHECKSUM" = "${INITIAL_PUBLICATION_CHECKSUM:-}" ]; then
  echo "Error: updated checksum is identical to initial checksum; update test is invalid." >&2
  exit 1
fi

cat > "$UPDATED_PUBLICATION_PAYLOAD" <<EOF
{
  "uuid": "$PUBLICATION_ID",
  "title": "$PUBLICATION_TITLE",
  "encryption_key": "$ENCRYPTION_KEY",
  "href": "$PUBLICATION_HREF",
  "content_type": "application/epub+zip",
  "size": $UPDATED_PUBLICATION_SIZE,
  "checksum": "$UPDATED_PUBLICATION_CHECKSUM"
}
EOF

echo "Updating publication record with new size/checksum"

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X PUT "${BASE_URL%/}/publications/$PUBLICATION_ID" \
  --data @"$UPDATED_PUBLICATION_PAYLOAD" \
  > "$UPDATED_PUBLICATION_RESPONSE"

echo "Fetching LSD status document"

curl -fsS \
  "${BASE_URL%/}/status/$LICENSE_ID" \
  > "$UPDATED_STATUS_FILE"

FRESH_LICENSE_URL="$(python3 - "$UPDATED_STATUS_FILE" "$BASE_URL" "$LICENSE_ID" <<'PY'
import json
import sys
from urllib.parse import urljoin

status_file, base_url, license_id = sys.argv[1], sys.argv[2].rstrip("/"), sys.argv[3]

with open(status_file, "r", encoding="utf-8") as f:
    status = json.load(f)

fallback = f"{base_url}/licenses/{license_id}/"

license_links = [
    link for link in status.get("links", [])
    if link.get("rel") == "license"
]

if not license_links:
    print(fallback)
    raise SystemExit(0)

href = license_links[0].get("href", "")

# Expand the expected URI-template variable if present.
href = href.replace("{license_id}", license_id)

# Strip any remaining URI-template suffix.
href = href.split("{", 1)[0]

# Convert relative links to absolute URLs.
if href.startswith("/"):
    href = urljoin(base_url + "/", href.lstrip("/"))

# The current config-example may produce /license, which is not a server route.
# Fall back to the actual private fresh-license API route.
if href.rstrip("/") == f"{base_url}/license":
    print(fallback)
else:
    print(href)
PY
)"

cat > "$OUT_DIR/fresh-license-request-v2.json" <<EOF
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

echo "Fetching fresh license from LSD rel=license link"
echo "Fresh license URL: $FRESH_LICENSE_URL"

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "$FRESH_LICENSE_URL" \
  --data @"$OUT_DIR/fresh-license-request-v2.json" \
  > "$FRESH_LICENSE_FILE"

echo "Verifying fresh license publication link has updated length/hash"

python3 - "$FRESH_LICENSE_FILE" "$UPDATED_PUBLICATION_SIZE" "$UPDATED_PUBLICATION_CHECKSUM" <<'PY'
import json
import sys

license_file, expected_length, expected_hash = sys.argv[1], int(sys.argv[2]), sys.argv[3]

with open(license_file, "r", encoding="utf-8") as f:
    data = json.load(f)

publication_links = [link for link in data.get("links", []) if link.get("rel") == "publication"]
if not publication_links:
    raise SystemExit("No rel=publication link found in fresh license.")

publication = publication_links[0]

actual_length = publication.get("length")
actual_hash = publication.get("hash")

if actual_length != expected_length:
    raise SystemExit(f"Unexpected updated publication length: {actual_length} != {expected_length}")

if actual_hash != expected_hash:
    raise SystemExit(f"Unexpected updated publication hash: {actual_hash} != {expected_hash}")

print("OK: fresh license has updated publication length/hash.")
PY

cat <<EOF

Done.

Updated publication payload:
  $UPDATED_PUBLICATION_PAYLOAD

Updated LSD status:
  $UPDATED_STATUS_FILE

Fresh license:
  $FRESH_LICENSE_FILE

Updated publication length:
  $UPDATED_PUBLICATION_SIZE

Updated publication hash:
  $UPDATED_PUBLICATION_CHECKSUM

EOF
