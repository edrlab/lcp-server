---
layout: default
title: LCP Encrypt
nav_order: 7
---

The `lcpencrypt` utility is usable is different processing pipelines. 
It can be used as a command-line utility or as a standalone server.

This tool can encrypt different types of publications, and store them in a local or remote storage. It notifies the license server of the encryption key and URL of the encrypted publication.

# Usage

## Source formats
The `lcpencrypt` utility currently supports:

- EPUB 2
- EPUB 3 (with embedded multimedia content, media overlays, etc.)
- PDF
- Readium Packages, i.e. Readium Web Publications archived using the Readium Packaging Format. This includes Audiobooks, PDF files embedded in Readium Packages, Digital visual Narratives and (X)HTML-based Publications.
- W3C Audiobooks packaged as LPF files.

## Server mode:
- serve: if set, start the utility as a server; the uuid flag is ignored in this mode

## Required Parameters:
-input: source file locator (file path or url).
-storage: target location for the encrypted publication, without filename. It is either an S3 bucket or the path to a file system from which files are accessible via HTTP(s). See details below.
-url: base url associated with the storage. This must be a public URL.

## Optional Parameters:
-provider: provider URI of each license associated with the publication. If omitted, the provider URI set by server configuration is used. 
-uuid: unique identifier of the encrypted publication. If omitted, a uuid is generated. Used to update an existing publication.
-altid: alternative ID of the publication, used to update an existing publication.
-usefnas: if set to 'uuid' or 'altid', the source file name is used as publication uuid or alternative id.
-pdfnometa: if set, indicates that PDF metadata are not extracted from the source file.
-cover: if set, the cover extracted from an EPUB or PDF file is exported as an additional file (useful for feeding a CMS).
-verbose: boolean, the information sent to the LCP Server and CMS will be displayed.

## Parameters used to notify the LCP server of an encryption
-lcpsv: host name of the License Server to be notified; syntax http://username:password@example.com.
-v2: indicates that the License server is a V2 one (default true)

# S3-related parameters (e.g. Cloudflare R2, MinIO, Backblaze B2)
-s3endpoint: custom S3 endpoint for S3-compatible storage providers.
-s3pathstyle: if set, forces path style addressing for S3.
-s3disablessl: if set, disables SSL for S3, intended for local/dev S3-compatible servers.

# Corresponding environment variables
- LCPENCRYPT_INPUT_PATH
- LCPENCRYPT_PROVIDER_URI
- LCPENCRYPT_VERBOSE
- LCPENCRYPT_V2
- LCPENCRYPT_COVER
- LCPENCRYPT_PDF_NO_META
- LCPENCRYPT_USE_FILENAME_AS
- LCPENCRYPT_STORAGE_PATH
- LCPENCRYPT_STORAGE_URL
- LCPENCRYPT_LCPSERVER_URL
- LCPENCRYPT_CMS_URL
- LCPENCRYPT_S3_ENDPOINT
- LCPENCRYPT_S3_FORCE_PATH_STYLE
- LCPENCRYPT_S3_DISABLE_SSL


