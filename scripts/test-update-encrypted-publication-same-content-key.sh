#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="${STATE_FILE:-./.local/lcp-update-test/state.env}"

if [ ! -f "$STATE_FILE" ]; then
  echo "Error: state file not found: $STATE_FILE" >&2
  echo "Run the create-publication-loan script first, or set STATE_FILE=/path/to/state.env." >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$STATE_FILE"

LICENSE_SERVER_DIR="${LICENSE_SERVER_DIR:-$OUT_DIR/served-licenses}"
LICENSE_SERVER_BASE_URL="${LICENSE_SERVER_BASE_URL:-http://localhost:8991}"

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

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: $1 is required." >&2
    exit 1
  }
}

require go
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

BASE_URL="${BASE_URL%/}"
OUT_DIR="${OUT_DIR:-./.local/lcp-update-test}"
RESOURCE_DIR="${RESOURCE_DIR:-./.local/lcpserver/resources}"

mkdir -p "$OUT_DIR" "$RESOURCE_DIR" "$OUT_DIR/source-v2" "$REPO_ROOT/bin"

LCPENCRYPT_BIN="$REPO_ROOT/bin/lcpencrypt"

PUBLICATION_BEFORE="$OUT_DIR/publication-before-v2.json"
PUBLICATION_AFTER_LCPENCRYPT="$OUT_DIR/publication-after-lcpencrypt-v2.json"
PUBLICATION_UPDATE_PAYLOAD="$OUT_DIR/publication-v2-size-checksum-only.json"
PUBLICATION_AFTER_FINAL_UPDATE="$OUT_DIR/publication-after-final-v2.json"

SOURCE_EPUB="$OUT_DIR/source-v2/demo-publication-v2.epub"
BASELINE_LICENSE="$OUT_DIR/license-before-v2.lcpl"
UPDATED_STATUS_FILE="$OUT_DIR/status-v2.json"
FRESH_LICENSE_REQUEST="$OUT_DIR/fresh-license-request-v2.json"
FRESH_LICENSE_FILE="$OUT_DIR/fresh-license-v2.lcpl"

echo "Building lcpencrypt..."
CGO_ENABLED=1 go build -o "$LCPENCRYPT_BIN" ./cmd/lcpencrypt

echo "Fetching current publication..."
curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  "${BASE_URL}/publications/${PUBLICATION_ID}" \
  > "$PUBLICATION_BEFORE"

echo "Fetching baseline fresh license before update..."
cat > "$FRESH_LICENSE_REQUEST" <<EOF
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

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "${BASE_URL}/licenses/${LICENSE_ID}/" \
  --data @"$FRESH_LICENSE_REQUEST" \
  > "$BASELINE_LICENSE"

echo "Generating modified source EPUB v2..."

python3 - "$SOURCE_EPUB" "$PUBLICATION_ID" <<'PY'
from pathlib import Path
from zipfile import ZipFile, ZIP_STORED, ZIP_DEFLATED
from datetime import datetime, timezone
import sys

out_path = Path(sys.argv[1])
publication_id = sys.argv[2]
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

chapter_xhtml = """<?xml version="1.0" encoding="UTF-8"?>
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
      <p>This is version 2 of the same publication.</p>
      <p>The content has changed, but the LCP content-key must remain identical.</p>
      <p>Extra bytes to force a different encrypted publication length and SHA-256 hash.</p>
      <p>xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</p>
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

LCP_SERVER_AUTH_URL="$(python3 - "$BASE_URL" "$API_USER" "$API_PASSWORD" <<'PY'
from urllib.parse import urlsplit, urlunsplit, quote
import sys

base_url, user, password = sys.argv[1], sys.argv[2], sys.argv[3]
parts = urlsplit(base_url)

netloc = f"{quote(user)}:{quote(password)}@{parts.netloc}"
print(urlunsplit((parts.scheme, netloc, parts.path, parts.query, parts.fragment)))
PY
)"

echo "Encrypting v2 with the existing content-key..."
echo "Publication ID: $PUBLICATION_ID"

"$LCPENCRYPT_BIN" \
  -input "$SOURCE_EPUB" \
  -uuid "$PUBLICATION_ID" \
  -provider "$BASE_URL" \
  -storage "$RESOURCE_DIR" \
  -url "${BASE_URL}/resources" \
  -lcpsv "$LCP_SERVER_AUTH_URL" \
  -v2=true \
  -cover=false \
  -verbose

echo "Fetching publication after lcpencrypt notification..."
curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  "${BASE_URL}/publications/${PUBLICATION_ID}" \
  > "$PUBLICATION_AFTER_LCPENCRYPT"

echo "Replacing the publication bytes at the existing href..."

read -r ENCRYPTED_OUTPUT_FILE TARGET_PUBLICATION_FILE < <(
  python3 - "$PUBLICATION_BEFORE" "$PUBLICATION_AFTER_LCPENCRYPT" "$RESOURCE_DIR" "$PUBLICATION_ID" <<'PY'
import json
import sys
from pathlib import Path
from urllib.parse import urlparse

before_file, after_file, resource_dir, publication_id = sys.argv[1], sys.argv[2], Path(sys.argv[3]), sys.argv[4]

with open(before_file, "r", encoding="utf-8") as f:
    before = json.load(f)

with open(after_file, "r", encoding="utf-8") as f:
    after = json.load(f)

def href_to_resource_file(href):
    if not href:
        return None
    name = Path(urlparse(href).path).name
    if not name:
        return None
    return resource_dir / name

# File currently referenced by the original publication href.
target = href_to_resource_file(before.get("href"))

if target is None:
    raise SystemExit("Could not resolve original publication href to a resource file.")

# Candidate encrypted outputs produced by lcpencrypt.
candidates = []

after_target = href_to_resource_file(after.get("href"))
if after_target:
    candidates.append(after_target)

candidates.append(resource_dir / f"{publication_id}.epub")
candidates.append(resource_dir / Path(urlparse(before.get("href", "")).path).name)

encrypted = None
for candidate in candidates:
    if candidate and candidate.exists():
        encrypted = candidate
        break

if encrypted is None:
    raise SystemExit("Could not find encrypted output file generated by lcpencrypt.")

print(encrypted, target)
PY
)

if [ "$ENCRYPTED_OUTPUT_FILE" != "$TARGET_PUBLICATION_FILE" ]; then
  cp "$ENCRYPTED_OUTPUT_FILE" "$TARGET_PUBLICATION_FILE"
fi

echo "Encrypted output file:"
echo "  $ENCRYPTED_OUTPUT_FILE"

echo "Publication href file replaced:"
echo "  $TARGET_PUBLICATION_FILE"

echo "Computing final publication size/checksum from the file served by href..."

read -r UPDATED_PUBLICATION_SIZE UPDATED_PUBLICATION_CHECKSUM < <(
  python3 - "$TARGET_PUBLICATION_FILE" <<'PY'
from pathlib import Path
import hashlib
import sys

path = Path(sys.argv[1])
data = path.read_bytes()
print(path.stat().st_size, hashlib.sha256(data).hexdigest())
PY
)

echo "Updated length: $UPDATED_PUBLICATION_SIZE"
echo "Updated hash:   $UPDATED_PUBLICATION_CHECKSUM"

# echo "Computing encrypted publication size/checksum..."
# 
# read -r UPDATED_PUBLICATION_SIZE UPDATED_PUBLICATION_CHECKSUM ENCRYPTED_OUTPUT_FILE < <(
#   python3 - "$PUBLICATION_BEFORE" "$PUBLICATION_AFTER_LCPENCRYPT" "$RESOURCE_DIR" "$PUBLICATION_ID" <<'PY'
# import hashlib
# import json
# import sys
# from pathlib import Path
# from urllib.parse import urlparse
# 
# before_file, after_file, resource_dir, publication_id = sys.argv[1], sys.argv[2], Path(sys.argv[3]), sys.argv[4]
# 
# with open(before_file, "r", encoding="utf-8") as f:
#     before = json.load(f)
# 
# with open(after_file, "r", encoding="utf-8") as f:
#     after = json.load(f)
# 
# # lcpencrypt normally stores the file as <publication UUID>.epub when no storage filename is imposed.
# candidates = []
# 
# after_href = after.get("href") or ""
# if after_href:
#     candidates.append(resource_dir / Path(urlparse(after_href).path).name)
# 
# before_href = before.get("href") or ""
# if before_href:
#     candidates.append(resource_dir / Path(urlparse(before_href).path).name)
# 
# candidates.append(resource_dir / f"{publication_id}.epub")
# 
# encrypted_path = None
# for candidate in candidates:
#     if candidate.exists():
#         encrypted_path = candidate
#         break
# 
# if encrypted_path is None:
#     raise SystemExit("Could not find encrypted output file in resource directory.")
# 
# data = encrypted_path.read_bytes()
# print(encrypted_path.stat().st_size, hashlib.sha256(data).hexdigest(), encrypted_path)
# PY
# )
# 
# echo "Encrypted output file: $ENCRYPTED_OUTPUT_FILE"
# echo "Updated length: $UPDATED_PUBLICATION_SIZE"
# echo "Updated hash:   $UPDATED_PUBLICATION_CHECKSUM"

echo "Re-applying publication update: preserve all previous fields except size/checksum..."

python3 - \
  "$PUBLICATION_BEFORE" \
  "$PUBLICATION_AFTER_LCPENCRYPT" \
  "$PUBLICATION_UPDATE_PAYLOAD" \
  "$UPDATED_PUBLICATION_SIZE" \
  "$UPDATED_PUBLICATION_CHECKSUM" <<'PY'
import json
import sys

before_file, after_file, out_file, size, checksum = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4]), sys.argv[5]

with open(before_file, "r", encoding="utf-8") as f:
    before = json.load(f)

with open(after_file, "r", encoding="utf-8") as f:
    after = json.load(f)

# Keep the same content-key as the encrypted v2 notification used.
# It should be identical to before; this check makes the intent explicit.
if before.get("encryption_key") != after.get("encryption_key"):
    raise SystemExit("Content-key changed after lcpencrypt update; test aborted.")

payload = {
    "uuid": before.get("uuid"),
    "alt_id": before.get("alt_id", ""),
    "provider": before.get("provider", ""),
    "title": before.get("title"),
    "description": before.get("description", ""),
    "authors": before.get("authors", ""),
    "publishers": before.get("publishers", ""),
    "cover_url": before.get("cover_url", ""),
    "encryption_key": before.get("encryption_key"),
    "href": before.get("href"),
    "content_type": before.get("content_type"),
    "size": size,
    "checksum": checksum,
}

with open(out_file, "w", encoding="utf-8") as f:
    json.dump(payload, f, indent=2)
    f.write("\n")
PY

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X PUT "${BASE_URL}/publications/${PUBLICATION_ID}" \
  --data @"$PUBLICATION_UPDATE_PAYLOAD" \
  > "$PUBLICATION_AFTER_FINAL_UPDATE"

echo "Verifying downloaded publication length matches updated metadata..."

DOWNLOADED_PUBLICATION="$OUT_DIR/downloaded-publication-v2.epub"

curl -fsS \
  "$PUBLICATION_HREF" \
  > "$DOWNLOADED_PUBLICATION"

DOWNLOADED_SIZE="$(python3 - "$DOWNLOADED_PUBLICATION" <<'PY'
from pathlib import Path
import sys
print(Path(sys.argv[1]).stat().st_size)
PY
)"

if [ "$DOWNLOADED_SIZE" != "$UPDATED_PUBLICATION_SIZE" ]; then
  echo "Error: downloaded publication length mismatch." >&2
  echo "Expected: $UPDATED_PUBLICATION_SIZE" >&2
  echo "Got:      $DOWNLOADED_SIZE" >&2
  exit 1
fi

echo "OK: downloaded publication length matches updated metadata."

echo "Verifying publication changed only size/checksum..."

python3 - "$PUBLICATION_BEFORE" "$PUBLICATION_AFTER_FINAL_UPDATE" "$UPDATED_PUBLICATION_SIZE" "$UPDATED_PUBLICATION_CHECKSUM" <<'PY'
import json
import sys

before_file, after_file, expected_size, expected_checksum = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4]

with open(before_file, "r", encoding="utf-8") as f:
    before = json.load(f)

with open(after_file, "r", encoding="utf-8") as f:
    after = json.load(f)

fields_to_compare = [
    "uuid",
    "alt_id",
    "provider",
    "title",
    "description",
    "authors",
    "publishers",
    "cover_url",
    "encryption_key",
    "href",
    "content_type",
]

for field in fields_to_compare:
    if before.get(field, "") != after.get(field, ""):
        raise SystemExit(f"Unexpected publication field change: {field}")

if after.get("size") != expected_size:
    raise SystemExit(f"Unexpected publication size: {after.get('size')} != {expected_size}")

if after.get("checksum") != expected_checksum:
    raise SystemExit(f"Unexpected publication checksum: {after.get('checksum')} != {expected_checksum}")

print("OK: publication metadata is unchanged except size/checksum.")
PY

echo "Fetching LSD status document..."

curl -fsS \
  "${BASE_URL}/status/$LICENSE_ID" \
  > "$UPDATED_STATUS_FILE"

# FRESH_LICENSE_URL="$(python3 - "$UPDATED_STATUS_FILE" "$BASE_URL" "$LICENSE_ID" <<'PY'
# import json
# import sys
# from urllib.parse import urljoin
# 
# status_file, base_url, license_id = sys.argv[1], sys.argv[2].rstrip("/"), sys.argv[3]
# 
# with open(status_file, "r", encoding="utf-8") as f:
#     status = json.load(f)
# 
# fallback = f"{base_url}/licenses/{license_id}/"
# 
# license_links = [
#     link for link in status.get("links", [])
#     if link.get("rel") == "license"
# ]
# 
# if not license_links:
#     print(fallback)
#     raise SystemExit(0)
# 
# href = license_links[0].get("href", "")
# href = href.replace("{license_id}", license_id)
# href = href.split("{", 1)[0]
# 
# if href.startswith("/"):
#     href = urljoin(base_url + "/", href.lstrip("/"))
# 
# # Tolerate the current config-example value /license, which is not a working route.
# if href.rstrip("/") == f"{base_url}/license":
#     print(fallback)
# else:
#     print(href)
# PY
# )"
# 

FRESH_LICENSE_URL="${BASE_URL}/licenses/${LICENSE_ID}/" 
echo "Fetching fresh license from: $FRESH_LICENSE_URL"

curl -fsS \
  -u "$API_USER:$API_PASSWORD" \
  -H "Content-Type: application/json" \
  -X POST "$FRESH_LICENSE_URL" \
  --data @"$FRESH_LICENSE_REQUEST" \
  > "$FRESH_LICENSE_FILE"

publish_latest_lcpl "$LICENSE_ID" "$FRESH_LICENSE_FILE"

echo "Verifying fresh license publication hash/length and stable semantic data..."

python3 - \
  "$BASELINE_LICENSE" \
  "$FRESH_LICENSE_FILE" \
  "$UPDATED_PUBLICATION_SIZE" \
  "$UPDATED_PUBLICATION_CHECKSUM" <<'PY'
import copy
import json
import sys

before_file, after_file, expected_length, expected_hash = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4]

with open(before_file, "r", encoding="utf-8") as f:
    before = json.load(f)

with open(after_file, "r", encoding="utf-8") as f:
    after = json.load(f)

def publication_link(doc):
    links = [link for link in doc.get("links", []) if link.get("rel") == "publication"]
    if not links:
        raise SystemExit("No rel=publication link found.")
    return links[0]

before_pub = publication_link(before)
after_pub = publication_link(after)

if after_pub.get("length") != expected_length:
    raise SystemExit(f"Fresh license length mismatch: {after_pub.get('length')} != {expected_length}")

if after_pub.get("hash") != expected_hash:
    raise SystemExit(f"Fresh license hash mismatch: {after_pub.get('hash')} != {expected_hash}")

# The signature must change when the license payload changes.
# Some encrypted fields may also be non-deterministic depending on IV generation.
# Normalize cryptographic wrappers, then assert that the semantic payload is stable
# except publication hash/length.
def normalized(doc):
    doc = copy.deepcopy(doc)
    doc.pop("signature", None)

    if "encryption" in doc:
        doc["encryption"].setdefault("content_key", {}).pop("encrypted_value", None)
        doc["encryption"].setdefault("user_key", {}).pop("key_check", None)

    for link in doc.get("links", []):
        if link.get("rel") == "publication":
            link["length"] = "__NORMALIZED_LENGTH__"
            link["hash"] = "__NORMALIZED_HASH__"

    return doc

if normalized(before) != normalized(after):
    print("Baseline normalized license:")
    print(json.dumps(normalized(before), indent=2, sort_keys=True))
    print("Updated normalized license:")
    print(json.dumps(normalized(after), indent=2, sort_keys=True))
    raise SystemExit("Fresh license changed beyond publication hash/length and cryptographic wrappers.")

print("OK: fresh license publication link has updated hash/length; semantic data is otherwise unchanged.")
PY

cat <<EOF

Done.

Updated encrypted publication:
  $ENCRYPTED_OUTPUT_FILE

Publication update payload:
  $PUBLICATION_UPDATE_PAYLOAD

LSD status:
  $UPDATED_STATUS_FILE

Fresh license:
  $FRESH_LICENSE_FILE

Updated publication length:
  $UPDATED_PUBLICATION_SIZE

Updated publication hash:
  $UPDATED_PUBLICATION_CHECKSUM

EOF
