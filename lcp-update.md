# LCP / Publication Update

Local branch: `lcp-publication-update`

Target Thorium Reader PR: [edrlab/thorium-reader#3642](https://github.com/edrlab/thorium-reader/pull/3642)

Note: this `lcp-server` clone does not contain a local or remote reference named `develop`. The local diff was therefore analyzed against `origin/main`, which is the base of the current branch. The Thorium PR itself targets `edrlab/thorium-reader:develop`.

## Goal

This branch adds a local test environment to reproduce the case handled in Thorium Reader: an already imported LCP license is refreshed through LSD, and the refreshed LCPL points to an updated publication archive through its `rel="publication"` link.

The goal is to verify that Thorium correctly detects an updated publication when the `hash` or `length` of the `rel="publication"` link changes, then downloads the new archive and replaces the stored files without breaking the existing loan.

## LCP Server Changes

### License Refresh After Publication Updates

`pkg/api/publication_handler.go` was updated to detect publication changes that affect the LCPL document:

- `title`
- `encryption_key`
- `href`
- `content_type`
- `size`
- `checksum`

When one of these fields changes, all licenses linked to the publication are marked as updated.

Important detail: the change affects the license document, not the LSD status. If `StatusUpdated` is empty, it is initialized with `CreatedAt` before updating `Updated`. This lets the LSD document report that the license changed through `updated.license`, without incorrectly signaling that the status itself changed through `updated.status`.

Expected effect in Thorium: during the LSD refresh, Thorium sees that a fresh license is available, fetches the new LCPL, then compares its `rel="publication"` link with the previously known one.

### Added Scripts

The diff adds a set of test scripts under `scripts/`:

- `quickstart-lcpserver-sqlite.sh`: builds `lcpserver`, creates a local SQLite configuration, and starts the server.
- `license-file-server.py`: serves fresh LCPL files from a local directory through `/licenses/{license_id}`.
- `test-create-publication-loan.sh`: generates a minimal EPUB publication, creates it in `lcpserver`, creates an LCP loan, and publishes the initial LCPL for the file server.
- `test-update-publication-lsd-license.sh`: regenerates a v2 publication with different bytes, updates `size` and `checksum`, fetches a fresh LCPL, and checks that the `rel="publication"` link contains the new `length`/`hash`.
- `test-update-encrypted-publication-same-content-key.sh`: generates an encrypted v2 publication with the same content key, replaces the bytes served by the previous `href`, updates only `size`/`checksum`, publishes the fresh LCPL, and checks that semantic license data remains stable.
- `test-lcp-update-demo-flow.sh`: runs the create step, pauses for a Thorium demo/import check, then runs the update step.

## Thorium Validation Scenario

The useful flow for the Thorium PR is the following.

### 1. Start the Local LCP Server

```sh
RECREATE_CONFIG=1 ./scripts/quickstart-lcpserver-sqlite.sh
```

This script:

- builds `cmd/lcpserver`;
- creates a SQLite configuration under `.local/lcpserver/`;
- configures `status.fresh_license_link` to point to the local fresh-LCPL server;
- exposes EPUB resources through the LCP server.

### 2. Start the Fresh-LCPL Server

In a second terminal:

```sh
LICENSE_SERVER_DIR=.local/lcp-update-test/served-licenses LICENSE_SERVER_PORT=8991 ./scripts/license-file-server.py
```

This server serves the latest known LCPL for each loan through:

```txt
http://localhost:8991/licenses/{license_id}
```

The `quickstart` script configures LSD to point to this route. This simulates an LCPL returned by an LSD server after a renewal or refresh.

### 3. Create the v1 Publication and Initial Loan

In a third terminal:

```sh
./scripts/test-create-publication-loan.sh
```

This script:

- generates a minimal v1 EPUB;
- computes its byte length and SHA-256 hash;
- creates the publication in `lcpserver`;
- creates an LCP license with passphrase `123 456`;
- publishes the initial LCPL under `.local/lcp-update-test/served-licenses/`;
- writes the test state to `.local/lcp-update-test/state.env`.

At this point, Thorium can import the initial LCPL and open the publication.

### 4. Update the Publication

Recommended case for the Thorium PR:

```sh
./scripts/test-update-encrypted-publication-same-content-key.sh
```

This script produces an encrypted v2 publication while keeping the same content key. It then forces an update of the existing publication by changing only the served bytes and the integrity metadata (`size` and `checksum`).

It then checks that:

- the archive served by `href` has the new byte length;
- the publication keeps the same functional fields;
- the fresh LCPL exposes a `rel="publication"` link with the new `length` and `hash`;
- the rest of the license data remains stable, apart from the expected signature and cryptographic wrappers.

More direct, unencrypted alternative:

```sh
./scripts/test-update-publication-lsd-license.sh
```

This script is useful for quickly checking that the LCP server emits a fresh LCPL with an updated `rel="publication"` link after a `size`/`checksum` change.

### Demo Flow With Pause

To run the Thorium demo flow in one command after starting both servers:

```sh
bash ./scripts/test-lcp-update-demo-flow.sh
```

This wrapper runs `test-create-publication-loan.sh`, prints the LCPL file, license ID, publication ID and passphrase to use in Thorium, then waits before running `test-update-encrypted-publication-same-content-key.sh`.

When Thorium has imported and opened the v1 loan, press Enter in the terminal to publish the v2 update. Set `PAUSE_BEFORE_UPDATE=0` to run the whole flow without waiting.

## Expected Result in Thorium

After importing the initial LCPL, then running the update script:

1. Thorium refreshes the LSD status.
2. LSD indicates that a fresh license is available.
3. Thorium fetches the new LCPL.
4. The LCPL `rel="publication"` link contains a different `hash` or `length`.
5. Thorium downloads the new protected archive.
6. Thorium verifies the archive integrity.
7. Thorium injects the refreshed LCPL into the replacement archive.
8. Thorium replaces the stored publication files transactionally.
9. The publication opens with the v2 content.

The negative test to run on the Thorium side is to publish an LCPL with an invalid `hash`: Thorium must reject the replacement and keep the previous publication readable, without leaving partially replaced files behind.

## Local Diff Summary

Original branch diff analyzed against `origin/main`, before adding this demo-flow helper:

```txt
8 files changed, 2171 insertions(+)
```

Modified or added files:

```txt
M  pkg/api/publication_handler.go
A  scripts/demo-lcp-loan.sh
A  scripts/license-file-server.py
A  scripts/quickstart-lcpserver-sqlite.sh
A  scripts/quickstart-lcpserver-sqlite.sh.bak
A  scripts/test-create-publication-loan.sh
A  scripts/test-update-encrypted-publication-same-content-key.sh
A  scripts/test-update-publication-lsd-license.sh
```

Additional working-tree helper added for the Thorium demo flow:

```txt
A  scripts/test-lcp-update-demo-flow.sh
```
