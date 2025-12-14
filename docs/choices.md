---
layout: default
title: Choices of development
nav_order: 6
---

# Choices of development

We wanted to develop this new version of the LCP Server around three principles:

- A performant and clean router (mux).
- Flexible access to different databases (SQLite, MySQL, Postgres, SQL Server).
- A complete set of unit tests.

## Chi
There are plenty of routers in Go land. Gin and Chi are two performant pieces of software among others. Gin seems to be the most popular these days, but we have finally chosen *Chi* for this development for its stability over time, its compatibility with net/http and its clean "render" helpers. Chi supports JWT authentication and OAuth2 autorisation, which will be useful.

Project home: https://go-chi.io/#/

A side-by-side comparison of Gin and Chi: https://go.libhunt.com/compare-gin-vs-chi

On August 2022:
- Gin 1.8.1 was released in June 2022 ; 428 issues + 125 PR
- Chi 5.0.7 was released in Nov 2021 ; 19 issues + 9 PR

## GORM
Working with an ORM abstracts us from low-level storage code and is especially useful for software which must be adapted to different database solutions.  

Gorm officially supports the following databases: SQLite, MySQL, PostgreSQL, SQL Server, Oracle, GaussDB, TiDB, Clickhouse. Using a compatible databases (e.g. MariaDB) should not be an issue. 

Note: The proper driver must be included in the codebase and the codebase recompiled for a given database to be usable. See: [https://gorm.io/docs/connecting_to_the_database.html](https://gorm.io/docs/connecting_to_the_database.html) 
