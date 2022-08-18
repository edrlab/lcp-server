## Rationale

This is a major evolution of the Readium LCP Server available on https://github.com/readium/readium-lcp-server. Because it is not an incremental evolution, we decided to create a new repository on the EDRLab Github space. When this software has been properly tested and widely used in production, we will certainly archive the codebase on the Readium Github. 

This server:

- Receives notifications for each encryption of a publication.
- Serves LCP licenses for these publications. 
- Supports the License Status Document protocol for these licenses.

It will soon be completed by two other open-source applications on the same EDRLab Github:

- **lcp-encryptor**: a new LCP encryption tool, available both as a command line tool and as a server, able to communicate with this lcp-server.
- **lcp-pubstore**: a lightweight content management server connected to the lcp-server, only useful for testing this lcp-server.

Before these tools are available, the current Encryption Tool and Test Frontend Server will be usable with this new lcp-server. 

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