# LCP Server — Project Context for Claude Code

## What This Is

This is the [EDRLab Readium LCP v2 server](https://edrlab.github.io/lcp-server/) — a Go-based DRM licensing server for encrypted EPUB/PDF/Web Publications. It is **v2**, not v1. The architecture and API are substantially different from the original lcp-server project. Do not conflate with older LCP documentation found on the internet.

This repo is being integrated into a larger ebook distribution platform. The platform is a separate NestJS/Astro web app (not in this repo).

## The Stack

- **Web app**: NestJS (API) + Astro (static frontend), Node.js
- **Reader clients**: Mobile apps (out of scope for now)
- **Infrastructure**: Azure
- **Team**: 2 people. Josh is the only one with infra experience.

## Architecture Decisions (Settled)

### Hosting: Azure Container Apps (no VMs)
- LCP server runs as a Docker container (image already in this repo)
- Encrypt worker runs as a separate container
- NestJS app runs as a separate container
- All within the same Container Apps Environment (internal networking is Azure-managed)
- **No custom VNet required** at this stage. Use Container Apps internal ingress + Managed Identity for security.

### File Storage: Azure Blob Storage
- The LCP server does NOT store files. It stores an `href` URL pointing to where the encrypted file lives.
- Two Blob containers:
  - `staging`: private. Raw (unencrypted) uploads land here temporarily.
  - `content`: readable. Encrypted `.lcpdf`/`.lcpa` files live here permanently. Public read is acceptable because LCP encryption is the DRM.
- Auth via **Managed Identity** — no credentials in code.

### Database: Azure Database for PostgreSQL Flexible Server
- LCP server configured with `dsn: "postgres://..."`
- Managed service, no VM/DB management overhead.

### Secrets: Azure Key Vault
- LCP signing certificate (cert + private key)
- LCP API credentials (`access.txt`)
- JWT secret key
- Mounted into containers as secrets (Container Apps has native Key Vault integration).

### Encryption Pipeline: Queue-based Encrypt Worker
The `lcpencrypt` binary (compiled into the LCP server Docker image) is wrapped in a separate worker container. Triggered by Azure Storage Queue — **no public ingress on the worker**.

Flow:
1. NestJS receives upload → writes raw file to `staging` Blob → enqueues message to Storage Queue
2. Encrypt worker dequeues message → reads from `staging` → runs `lcpencrypt` → writes encrypted file to `content` Blob → `POST /publications/` to LCP server
3. LCP server stores publication metadata (including Blob URL as `href`)

### What Is Public vs. Internal

| Component | Ingress |
|---|---|
| NestJS app | Public |
| LCP server `/status`, `/register`, `/renew`, `/return` | Public |
| LCP server private API (`/publications/`, `/licenses/`, etc.) | Internal (or public with HTTP Basic Auth) |
| Encrypt worker | None (queue-driven) |
| Blob `content` container | Public read (or CDN) |
| Blob `staging` container | Private (Managed Identity only) |
| Storage Queue | Private (Managed Identity only) |
| PostgreSQL | Azure-services-only access |

### License Gateway (Required NestJS Work)
Per LCP spec, reader apps fetch "fresh" licenses from the **platform**, not directly from the LCP server. NestJS must implement a License Gateway endpoint:
- Reader app: `GET /licenses/{license_id}` → NestJS
- NestJS looks up user record (passphrase hash, hint) by license_id
- NestJS: `POST /licenses/{license_id}` → LCP server
- Returns fresh license JSON to reader app

### User Model Extensions Required
NestJS user model needs two LCP-specific fields:
- `lcp_pass_hash`: SHA256 hex string of the user's LCP passphrase
- `lcp_hint`: textual hint shown by reader app when passphrase is requested
- Platform must store `license_id` on the user-publication transaction record

## Implementation Phases

### Phase 1: LCP Server on Azure
- [ ] Build Docker image, push to Azure Container Registry
- [ ] Deploy to Container Apps
- [ ] Provision PostgreSQL Flexible Server, wire up DSN
- [ ] Provision Key Vault, store cert + credentials, mount into container
- [ ] Verify `/health` endpoint publicly reachable
- [ ] Verify private endpoints protected by Basic Auth

### Phase 2: Blob + Encrypt Worker
- [ ] Create Blob Storage account, `staging` and `content` containers
- [ ] Create Storage Queue
- [ ] Build encrypt worker container (wraps lcpencrypt, reads queue)
- [ ] Assign Managed Identities, configure RBAC roles
- [ ] Deploy encrypt worker to Container Apps (no ingress)
- [ ] End-to-end test: upload EPUB → queue → encrypt → Blob → LCP server publication registered

### Phase 3: NestJS Integration
- [ ] Extend user model with `lcp_pass_hash`, `lcp_hint`
- [ ] License generation endpoint: on purchase, `POST /publications/` then `POST /licenses/`
- [ ] Store `license_id` on transaction record
- [ ] Implement License Gateway endpoint

### Phase 4: Public Exposure + Hardening
- [ ] Configure Container Apps custom domains + managed TLS
- [ ] Optionally add Azure CDN in front of Blob `content` container
- [ ] Review ingress rules, ensure LCP private API not unnecessarily exposed
- [ ] Load test encrypt worker queue throughput

## Key LCP Server Facts to Remember

- Config is via env vars + Docker secrets (not a config file) when running in Docker
- `lcpencrypt` binary is compiled into the same Docker image as `lcpserver`
- The `href` field on a publication is just a URL — it can be any publicly accessible URL
- LCP signing certificate must be a valid X.509 cert issued by EDRLab's CA for production; test cert in `/test/` for development
- HTTP Basic Auth credentials for private routes come from `/config/access.txt` (Docker secret)
- Dashboard uses JWT auth (separate credentials)
- List endpoints cap at 1000 results
- Soft deletes on publications (licenses remain valid after publication is "deleted")
