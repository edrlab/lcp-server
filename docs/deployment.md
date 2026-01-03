---
layout: default
title: Deployment
nav_order: 3
---

# Deployment 

# Docker

## Configuration
Using a configuration file is uneasy when setting up Docker images. 

The configuration of the LCP Server therefore goes through two steps:
- Setting environment variables
- Setting Docker secrets

### Setting environment variables
Information that may change over time but is not secret should be stored as environment variable, via a `.env` file stored on the root folder of the project.

Values unset in `.env` are mapped to empty string in the configuration codebase, and are treated as such by the application. 

### Setting Docker secrets
Access to private endpoints of the LCP Server via HTTP Basic Authentication and usernames and passwords used to access the server using JWT are defined in `/config/access.txt`.

This file defines colon separated usernames and passwords allowing access to the server API private endpoints and dashboard.

Please update /config/access.txt with the data you want to use for accessing the LCP Server:
- the first line is the username and password used to access the server using basic auth (private routes),
- the next lines are usernames and passwords used to access the server using JWT (dashboard).

Do not forget to replace the sample values by secure values.

### Building and running your application

Build the image of an LCP Server using SQLite (this is the default value) and start the containers:
`docker compose up -d`
or
`docker compose up`
if you prefer seeing application logs in the terminal.

The image is named `lcp-server`. 

Note: to force the image to be rebuilt, type:
`docker compose up --build`

And to restart and rebuild with a new configuration:
`docker compose down && docker compose up --build -d`

To check the logs
`docker compose logs server --tail=15`

To simply restart
`docker compose restart server`

Your application will be available at http://localhost:8989.

### Launch LCP Encrypt
The lcpencrypt binary is part of the container. You can launch it in a one-shot mode using: 

`docker compose exec server /app/lcpencrypt` with parameters

More 

### Alternative builds, with a MySQL database
Build the image of an LCP server using SQLite by typing:
`docker compose build --tag lcp-server:sqlite .`

Build the image of an LCP server using MySQL by typing:
`docker compose build --tag lcp-server:mysql .`

### Run in detached mode

If you choose to expose the default port, use:
`docker run --detach --publish 8989:8989 lcp-server`

### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t lcp-server .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t lcp-server .`

Then, push it to your registry, e.g. `docker push myregistry/lcp-server`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)

# MySQL

# Required Files

- `access.txt` - Basic authentication credentials for LCP server API
- `mysql-root-password.txt` - MySQL root password
- `mysql-password.txt` - Password for the `lcp_user` database user

## MySQL Setup

The Docker Compose configuration creates:
- A MySQL 8.4.7 Community server
- Database: `lcpserver`
- User: `lcp_user` with access to the `lcpserver` database
- Data stored in Docker volume: `db-data`, mapped to /var/lib/mysql

## Security Notes

**IMPORTANT**: Change the default passwords in the `.txt` files before running in production!

1. Edit `mysql-root-password.txt` with a strong root password (the default is `your_secure_root_password_change_this`)
2. Edit `mysql-password.txt` with a strong password for the lcp_user (the default is `lcp_user_password_change_this`)
3. Make sure these files have restricted permissions (600)

## Connection String Example

For the LCP server configuration, use a DSN like:
```
dsn: "lcp_user:password@tcp(mysql:3306)/lcpserver"
```

Where `password` matches the content of `mysql-password.txt`.

