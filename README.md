## Summary

This is a major evolution of the Readium LCP Server available on https://github.com/readium/readium-lcp-server. Because it is not an incremental evolution, we decided to create a new repository on the EDRLab Github space. When this software has been properly tested and widely used in production, we will certainly archive the codebase on the Readium Github. 

V2 requires go 1.16 or higher, due to the use of new features of the `os` package.

The `lcpserver`:

- Receives notifications for each encryption of a publication.
- Serves LCP licenses for these publications. 
- Supports the License Status Document protocol for these licenses.

Licenses can be verified using a separate commande line executable, developed in the same repository and named `lcpchecker`. 

An additional executable will soon be added to this set, named `lcpencrypt`. This tool will be able to encrypt a publication, publish the encrypted publication at a location given as a parameter, and notify the lcpserver of the availability of this new asset. `lcpencrypt` will be both available as a command line utility and a web service. 

A lightweight content management server will also be released, named `pubstore`, which will be connected to the lcp-server and will be useful for testing  `lcpserver` in a live environment.

Before these tools are available, the current Encryption Tool and Test Frontend Server will be usable with this new lcp-server (their API will be adapted to do so).

## Configuration

The configuration is similar to the v1 config, but simplified. 

This is a yaml file that can be located in any folder directly accessible from the application. The configuration file is found by the application via an environment variable named EDRLAB_LCPSERVER_CONFIG. 

For now, follow this example.

```yaml
# the public url of the server (used for setting links in the status document)
public_base_url: "http://localhost:8081"
# the port used by the server (default is 8081)
port: 8081
# data source name of access to the chosen database
dsn: "sqlite3://file::memory:?cache=shared"

# admin access for private routes
access:
  username: "login"
  password: "password"

license:
  # provider identifier, as a url, set in every license
  provider: "http://edrlab.org"
  # LCP profile identifier, set in every license, can be overridden per license
  profile: "http://readium.org/lcp/basic-profile"
  # link to a hint page, can be templated using {license_id} as parameter
  hint_link: "https://www.edrlab.org/lcp-help/{license_id}"

status:
  # default number of days of extension of a license, see renew; can be overridden in the renew command
  renew_default_days: 7
  # max number of days of extension of a license, see the specification of the status document
  renew_max_days: 40
  # renew URL optionally managed by the provider, which then takes care of calling the license status server
  # must be templated using {license_id} as parameter
  renew_link: "http://localhost:8081/renew/{license_id}"

# path to the X509 certificate and private key used for signing licenses
certificate:
  cert:       "/Users/x/test/cert/cert-edrlab-test.pem"
  private_key: "/Users/x/test/cert/privkey-edrlab-test.pem"
```

The test certificate is provided in the /test/cert folder on the project. 

## Usage

From the `lcp-server` folder ...

For testing the lcpserver application you can simply use:

> go run cmd/lcpserver/server.go

For compiling and installing the application in the bin folder, use: 

> go install cmd/lcpserver/server.go

## API calls

### CRUD on a publication

You can add a publication to the server via:

POST localhost:8081/publications/ 

with a payload like:

```json
{
    "uuid": "c6abe80a-1681-4694-b6f4-80c165213781",
    "title": "Voyage au centre de la terre",
    "encryption_key": "ZW5jcnlwdGlvbl9rZXkgeCBlbmNyeXB0aW9uX2tleQ==",
    "location": "https://edrlab.org/f/pub1.epub",
    "content_type": "application/epub+zip",
    "size": 769257,
    "checksum": "edce32ca54c36aa73da9075098fc592fa29ff3e12406d1442544535d99dc1b87" 
}
```

The publication will be identified by the `uuid` value.

You can also:

1. Get a list of publications via:

- GET localhost:8081/publications/

2. Fetch, update or delete (the info relative to) a publication via:

- GET localhost:8081/publications/<PublicationID> 
- PUT localhost:8081/publications/<PublicationID> (same payload as for a creation)
- DELETE localhost:8081/publications/<PublicationID> 

Where <PublicationID> is the uuid used for the creation of the publication. 

`location` must be a public URL, accessible from any device on the internet. 

Note: because publications are submitted to a soft delete, the suppression of a publication does not impact the existing 
licenses associated with the publication. But no new license can be generated for a deleted publication. 

### Generate a license

This is a private route. 

You can generate a license via:

POST localhost:8081/licenses/ 

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

`user_name` and `user_email` and `user_encrypted` are optional.
`copy`, `print`, `start`, `end` are optional constraints. No value set implies no constraint. 
`profile`is optional. A default value should be set in the configuration.  

All other paramaters are mandatory. 
The publication identified by `publication_id` must be present in the server when a license is generated. 

In case of success the server returns a 201 code. 
The returned payload is the newly generated license. 

### Fetch an existing (i.e. fresh) license

This is a private route. 

You can fetch a fresh license via:

POST localhost:8081/licenses/<licenseID>

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

The License Server does not store user information, as it would be the your entire user database is replicated in the License Server at some point, which is not desirable. This is why user information, including the personal text hint and passphrase, must be repeated each time a fresh license is requested. 

### Get a status document

This is a public route. 

Status is implemented as:

GET localhost:8081/status/<licenseID> 

The returned payload is a fresh status document.


### Register / Renew / Return a license

Register, Renew and Return are public routes.

Register is implemented as:

POST localhost:8081/register/<licenseID> 

Renew is implemented as: 

PUT localhost:8081/renew/<licenseID>

Return is implemented as:

PUT localhost:8081/return/<licenseID>

There is no payload associated with these routes, but two query parameters:

* id: a unique device identifier.
* name: a unique device name.

Renew can also take a third optional query parameter:

* end: the requested end date and time for the license, in W3C datetime format (YYYY-MM-DDThh:mm:ssTZD, cf https://www.w3.org/TR/NOTE-datetime)  

The returned payload is a fresh status document.


### Revoke a license

Renew is a private route. It is implemented as: 

PUT localhost:8081/revoke/<licenseID>

with no payload.

### CRUD on license information

You can add raw license information to the server via:

POST localhost:8081/licenseinfo/ 

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

The license will be identified by the `uuid` value.

You can also:

1. Get a list of licenses via:

- GET localhost:8081/licenses/

2. Fetch, update or delete a license (the info relative to) via:

- GET localhost:8081/licenseinfo/<LicenseID> 
- PUT localhost:8081/licenseinfo/<LicenseID> (same payload as for a creation)
- DELETE localhost:8081/licenseinfo/<LicenseID> 

Where <LicenseID> is the uuid used for the creation of the license. 

## Development choices
We wanted to develop this new version of the LCP Server around three principles:

- A performant and clean router (mux).
- Flexible access to different databases (SQLite, MySQL, Postgres, SQL Server).
- A complete set of unit tests.

### Chi
There are plenty of routers in Go land. Gin and Chi are two performant pieces of software among others. Gin seems to be the most popular these days, but we have finally chosen *Chi* for this development for its stability over time, its compatibility with net/http and its clean "render" helpers. Note that Chi supports JWT authentication and OAuth2 autorisation, which will be useful later.

Project home: https://go-chi.io/#/

A side-by-side comparison of Gin and Chi: https://go.libhunt.com/compare-gin-vs-chi

On August 2022:
- Gin 1.8.1 was released in June 2022 ; 428 issues + 125 PR
- Chi 5.0.7 was released in Nov 2021 ; 19 issues + 9 PR

### GORM
Working with an ORM abstracts us from low-level storage code and is especially useful for software which must be adapted to different database solutions.  

Gorm officially supports the following databases: SQLite, MySQL, PostgreSQL, SQL Server. An Oracle driver is available as an open PR on the Gorm Github (may 2021). 

Note: The proper driver must be included in the codebase and the codebase recompiled for a given database to be usable. See: [https://gorm.io/docs/connecting_to_the_database.html](https://gorm.io/docs/connecting_to_the_database.html) 

The open-source codebase is provided with an **sqlite** driver. It is up to integrators to replace it by the driver of their choice if sqlite does not fit their needs.