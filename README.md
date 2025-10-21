## Summary

This is a major evolution of the Readium LCP Server available on https://github.com/readium/readium-lcp-server. Because it is not an incremental evolution, and because this codebase is entirely maintained by EDRLab, we decided to create a new repository on the EDRLab Github space. 

**Note: This project is almost ready for production. We are currently testing it on a demo plateform before release.**

V2 requires go 1.16 or higher, due to the use of new features of the `os` package. It is currently developed using go 1.24. 

### lcpserver

The `lcpserver`:

- Receives notifications for each encryption of a publication.
- Serves LCP licenses for these publications. 
- Supports the License Status Document protocol for these licenses.

Licenses can be verified using a separate command line executable, developed in the same repository and named `lcpchecker`. 

### lcpencrypt

`lcpencrypt` aims to encrypt publications, publish encrypted publications at a location given as a parameter, and notify the lcpserver of the availability of this new asset. `lcpencrypt` is both available as a command line utility and a server tied to a watch folder. 

### lcpchecker

### Other tools

#### PubStore
This lightweight content management system named `pubstore` manages publications and users, the generation of LCP licenses when a user acquires publications, and the change of status of a license. It is useful for testing  `lcpserver` in a live environment.

#### LCP Server Dashboard
The lighweight dashboard server named `lcpdashboard` offers metrics on the LCP Server, displays oversharded licenses and allows admins to revoke overshared licenses. In can be used in production to manage an LCP Server.   


## Configuration

The configuration is similar to the v1 config, but largely simplified. 

The configuration of the server is kept both in a configuration file and in environment variables. It is possible to mix both sets;  environment variables are expressly recommended for confidential information. 

The configuration file is formatted as yaml, and can be located in any folder directly accessible from the application. The configuration file is found by the application via an environment variable named `EDRLAB_LCPSERVER_CONFIG`. Its value must be a file path.

Configuration properties are expressed in snake case in the configuration file, and all caps prefixed by `LCPSERVER` when expressed as environment variables. 
As an example, the `port` configuration property is mapped to the `LCPSERVER_PORT` environment variable, `public_base_url` becomes `LCPSERVER_PUBLICBASEURL`, and the `username` property of the `access` section becomes `LCPSERVER_ACCESS_USERNAME`.

For now, follow this example.

```yaml
# log level, can be "debug", "info", "warn", "error"
log_level: "debug"

# the public url of the server (used for setting links in the status document)
public_base_url: "http://localhost:8989"
# the port used by the server
port: 8989
# data source name of access to the chosen database
dsn: "sqlite3://file::memory:?cache=shared"

# username / password allowing access to the server API via http basic authentication
# for security reasons, it is much better to express these as environment variables (see the documentation)
# and docker secrets (https://docs.docker.com/compose/how-tos/use-secrets/)
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
  # url of a fresh license, served via a License Gateway 
  # must be templated using {license_id} as parameter
  fresh_license_link: "https://license_gateway.io/licenses/{license_id}"
  # allow renew on expired licenses
  allow_renew_on_expired_licenses: true
  # default number of days of extension of a license, see renew; can be overridden in the renew command
  renew_default_days: 7
  # max number of days of extension of a license, see the specification of the status document
  renew_max_days: 40
  # renew URL optionally managed by the provider, which then takes care of calling the license status server
  # must be templated using {license_id} as parameter
  renew_link: "http://localhost:8989/renew/{license_id}"

dashboard:
  # configurable threshold for licenses with excessive sharing (default is 6)
  excessive_sharing_threshold: 10
  # optional limit to last 12 months (default is false)
  limit_to_last_12_months: true

# path to the X509 certificate and private key used for signing licenses
certificate:
  cert:       "/Users/x/test/cert/cert-edrlab-test.pem"
  private_key: "/Users/x/test/cert/privkey-edrlab-test.pem"
```

The test certificate is provided in the /test/cert folder on the project. 

## Usage

For compiling and installing the application in the bin folder, use: 

> go install cmd/lcpserver/server.go

From the `lcp-server` folder ...

For testing the lcpserver application you can use:

> cd cmd/lcpserver
> go run server.go router.go authenticator.go


or, if your forked the codebase:

> go build -o $GOPATH/bin  ./cmd/lcpserver2

## API calls

### CRUD on a publication

You can add a publication to the server via:

POST localhost:8989/publications/ 

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

- GET localhost:8989/publications/, with `page` and `per_page` pagination parameters.
- GET localhost:8989/publications/search/ with a `format` parameter taking as a value: `epub`, `pdf`, `lcpdf`, `lcpaiu` or `lcpdi`. 

2. Fetch, update or delete (the info relative to) a publication via:

- GET localhost:8989/publications/<PublicationID> 
- PUT localhost:8989/publications/<PublicationID> (same payload as for a creation)
- DELETE localhost:8989/publications/<PublicationID> 

Where <PublicationID> is the uuid used for the creation of the publication. 

`href` must be a public URL, accessible from any device on the internet. 

Note: because publications are submitted to a soft delete, the suppression of a publication does not impact the existing 
licenses associated with the publication. But no new license can be generated for a deleted publication. 

### Generate a license

This is a private route. 

You can generate a license via:

POST localhost:8989/licenses/ 

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

All other parameters are mandatory. 
The publication identified by `publication_id` must be present in the server when a license is generated. 

In case of success the server returns a 201 code. 
The returned payload is the newly generated license. 

### Fetch an existing (i.e. fresh) license

This is a private route. 

You can fetch a fresh license via:

POST localhost:8989/licenses/<licenseID>

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

The License Server does not store user information, as the entire user database would then replicated in the License Server at some point, which is not desirable. This is why user information, including the personal text hint and passphrase, must be repeated each time a fresh license is requested. 

### Get a status document

This is a public route. 

Status is implemented as:

GET localhost:8989/status/<licenseID> 

The returned payload is a fresh status document.


### Register / Renew / Return a license

Register, Renew and Return are public routes.

Register is implemented as:

POST localhost:8989/register/<licenseID> 

Renew is implemented as: 

PUT localhost:8989/renew/<licenseID>

Return is implemented as:

PUT localhost:8989/return/<licenseID>

There is no payload associated with these routes, but two query parameters:

* id: a unique device identifier.
* name: a unique device name.

Renew can also take a third optional query parameter:

* end: the requested end date and time for the license, in W3C datetime format (YYYY-MM-DDThh:mm:ssTZD, cf https://www.w3.org/TR/NOTE-datetime)  

The returned payload is a fresh status document.


### Revoke a license

Renew is a private route. It is implemented as: 

PUT localhost:8989/revoke/<licenseID>

with no payload.

### CRUD on license information

You can add raw license information to the server via:

POST localhost:8989/licenseinfo/ 

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

- GET localhost:8989/licenses/, with `page` and `per_page` pagination parameters.
- GET localhost:8989/licenses/search/, with `user` (id), `pub` (id), `status` ("ready" etc.) or `count` query parameter. `count`takes a min:max tuple as value.

2. Fetch, update or delete a license (the info relative to) via:

- GET localhost:8989/licenseinfo/<LicenseID> 
- PUT localhost:8989/licenseinfo/<LicenseID> (same payload as for a creation)
- DELETE localhost:8989/licenseinfo/<LicenseID> 

Where <LicenseID> is the uuid used for the creation of the license. 

### Dashboard


## Development choices
We wanted to develop this new version of the LCP Server around three principles:

- A performant and clean router (mux).
- Flexible access to different databases (SQLite, MySQL, Postgres, SQL Server).
- A complete set of unit tests.

### Chi
There are plenty of routers in Go land. Gin and Chi are two performant pieces of software among others. Gin seems to be the most popular these days, but we have finally chosen *Chi* for this development for its stability over time, its compatibility with net/http and its clean "render" helpers. Chi supports JWT authentication and OAuth2 autorisation, which will be useful.

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