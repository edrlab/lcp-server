---
layout: default
title: LCP Server API
nav_order: 5
---

# LCP Server API

## Calls from the ebook delivery platform

### Generate a license

Access is protected by HTTP Basic Auth.

You can generate a license via:

POST {LCPServerURL}/licenses/ 

with a payload like: 

```json
{
    "publication_id": "c6abe80a-1681-4694-b6f4-80c165213780",
    "user_id": "552a6ffb-d79a-4ff2-bc66-6ebb08ccc4fe",
    "user_name": "John Doe",
    "user_email": "test@company.com",
    "user_encrypted": ["name","email"],
    "start": "2022-08-22T10:00:00Z",
    "end": "2022-08-30T10:00:00Z",
    "copy": 20000,
    "print": 100,
    "profile": "http://readium.org/lcp/basic-profile",
    "text_hint": "A textual hint for your passphrase.",
    "pass_hash": "FAEB00CA518BEA7CB11A7EF31FB6183B489B1B6EADB792BEC64A03B3F6FF80A8"
}
```

- `user_name` and `user_email` and `user_encrypted` are optional. `user_encrypted` is the list of user properties that will be encrypted in the LCP license. 
- `copy`, `print`, `start`, `end` are optional constraints. No value set means no constraint. 
- `profile`is optional. Allowed values are provided by EDRLab on request. A default value should be set in the LCP Server configuration.  

All other parameters are mandatory. 

About `pass_hash`: this is the user passphrase hashed using SHA256 and serialized as an hex-encoding string. For instance, the passphrase "123 456" becomes "4981AA0A50D563040519E9032B5D74367B1D129E239A1BA82667A57333866494" when hashed ([try with this online tool for testing](https://xorbin.com/tools/sha256-hash-calculator)). 

The publication identified by `publication_id` must be present in the server when a license is generated. 

In case of success the server returns a 201 code. 
The returned payload is the newly generated license. 

### Fetch a fresh license

Access is protected by HTTP Basic Auth.

You can fetch a fresh license via:

POST {LCPServerURL}/licenses/{licenseID}

with a payload like: 

```json
{
    "publication_id": "c6abe80a-1681-4694-b6f4-80c165213780",
    "user_id": "552a6ffb-d79a-4ff2-bc66-6ebb08ccc4fe",
    "user_name": "John Doe",
    "user_email": "test@company.com",
    "user_encrypted": ["name","email"],
    "profile": "http://readium.org/lcp/basic-profile",
    "text_hint": "A textual hint for your passphrase.",
    "pass_hash": "FAEB00CA518BEA7CB11A7EF31FB6183B489B1B6EADB792BEC64A03B3F6FF80A8"
}
```

The License Server does not store user information. This is why user information, including the textual hint and passphrase, must be repeated each time a fresh license is requested. 

## Other calls

### CRUD on a publication

One can add a publication to the server via:

POST {LCPServerURL}/publications/ 

with a payload like:

```json
{
    "uuid": "c6abe80a-1681-4694-b6f4-80c165213781",
    "title": "Voyage au centre de la terre",
    "encryption_key": "ZW5jcnlwdGlvbl9rZXkgeCBlbmNyeXB0aW9uX2tleQ==",
    "href": "https://edrlab.org/f/pub1.epub",
    "content_type": "application/epub+zip",
    "size": 769257,
    "checksum": "edce32ca54c36aa73da9075098fc592fa29ff3e12406d1442544535d99dc1b87" 
}
```

The publication will be identified by its `uuid` value.

You can also:

1. Get a list of publications via:

- GET {LCPServerURL}/publications/, with `page` and `per_page` pagination parameters.
- GET {LCPServerURL}/publications/search/ with a `format` parameter taking as a value: `epub`, `pdf`, `lcpdf`, `lcpaiu` or `lcpdi`. 

2. Fetch, update or delete (the info relative to) a publication via:

- GET {LCPServerURL}/publications/{publicationID} 
- PUT {LCPServerURL}/publications/{publicationID} (same payload as for a creation)
- DELETE {LCPServerURL}/publications/{publicationID} 

Where {publicationID} is the uuid used for the creation of the publication. 

`href` must be a public URL, accessible from any device on the internet. 

Note: because publications are submitted to a soft delete, the suppression of a publication does not impact the existing 
licenses associated with the publication. But no new license can be generated for a deleted publication. 


### Get a status document

This is a public route. 

Status is implemented as:

GET {LCPServerURL}/status/{licenseID} 

The returned payload is a fresh status document.


### Register / Renew / Return a license

Register, Renew and Return are public routes.

Register is implemented as:

POST {LCPServerURL}/register/{licenseID} 

Renew is implemented as: 

PUT {LCPServerURL}/renew/{licenseID}

Return is implemented as:

PUT {LCPServerURL}/return/{licenseID}

There is no payload associated with these routes, but two query parameters:

* id: a unique device identifier.
* name: a unique device name.

Renew can also take a third optional query parameter:

* end: the requested end date and time for the license, in W3C datetime format (YYYY-MM-DDThh:mm:ssTZD, cf https://www.w3.org/TR/NOTE-datetime)  

The returned payload is a fresh status document.


### Revoke a license

Renew is a private route. It is implemented as: 

PUT {LCPServerURL}/revoke/{licenseID}

with no payload.

### CRUD on license information

You can add raw license information to the server via:

POST {LCPServerURL}/licenseinfo/ 

with a payload like:

```json
{
    "uuid": "87ea1655-3973-4df4-983b-37144ed1b482",
    "user_id": "axv6rli8-1681-4694-b6f4-80c165213f56u",
    "publication_id": "c6abe80a-1681-4694-b6f4-80c165213781",
    "provider": "http://test-provider.com",
    "start": "2022-08-22T10:00:00Z",
    "end": "2022-08-30T10:00:00Z",
    "copy": 20000,
    "print": 100,
    "status": "ready"
}
```

The license will be identified by its `uuid` value.

You can also:

1. Get a list of licenses via:

- GET {LCPServerURL}/licenses/, with `page` and `per_page` pagination parameters.
- GET {LCPServerURL}/licenses/search/, with `user` (id), `pub` (id), `status` ("ready" etc.) or `count` query parameter. `count`takes a min:max tuple as value.

2. Fetch, update or delete a license (the info relative to) via:

- GET {LCPServerURL}/licenseinfo/{{LicenseID}} 
- PUT {LCPServerURL}/licenseinfo/{{LicenseID}} (same payload as for a creation)
- DELETE {LCPServerURL}/licenseinfo/{{LicenseID}} 

Where {{LicenseID}} is the uuid used for the creation of the license. 
