## Summary

This is a major evolution of the Readium LCP Server available on https://github.com/readium/readium-lcp-server. Because it is not an incremental evolution, and this codebase is entirely maintained by EDRLab, we decided to create a new repository on the EDRLab Github space. 

**Note: This project is now ready for production. Its tagged versions started at 2.0.0.**

This project is made of three executables: 

### LCP Server (lcpserver)

The `lcpserver`:

- Receives notifications for each encryption of a publication.
- Serves LCP licenses for these publications. 
- Supports the License Status Document protocol for these licenses.

### LCP Encryption tool (lcpencrypt)

`lcpencrypt`:

- Is both available as a command line utility and as a server triggered from a watch folder.
- Encrypts EPUBs, PDF documents, and packaged Web Publications.
- Stores encrypted publications at a location given as a parameter. This location can be a file system or a Cloud repository.
- Notifies the LCP Server of the availability of this new asset. 

### LCP checker (lcpchecker)

`lcpchecker` verifies the compliance of an LCP license with the LCP specification and the LSD protocol. It should be used by any LCP Server integrator to check their integration before they enter the EDRLab LCP certification phase.

### Other tools

These open-source tools are related to the LCP Server but maintained in different repositories: 

#### LCP Server Dashboard (lcpdashboard)
This SPA dashboard offers metrics on the LCP Server, displays oversharded licenses and allows admins to revoke overshared licenses. In can be used in production to manage an LCP Server. 

See https://github.com/edrlab/lcp-dashboard.

Note: We preferred developing it in a separated repository because it is a Node.js/React application: mixing it with a Go-based development would have drawbacks.

#### PubStore (pubstore)
This lightweight content management system has been developed for demonstration purpose. It manages publications and users, the generation of LCP licenses when a user acquires publications, and the change of status of a license. It is by no mean intended to be used in production. 

See https://github.com/edrlab/pubstore. 

## Quickstart

Assuming a working Go installation (Go 1.24 or higher), the project builds and runs with Go modules. `GOPATH` is not required.

The repository includes a default test configuration at `config/default.yaml`. It uses SQLite and the test certificate bundled with the project, so it is suitable for local development only.

Run the test suite:

```sh
go test ./...
```

Build the three executables locally:

```sh
mkdir -p build
go build -o ./build/lcpserver ./cmd/lcpserver
go build -o ./build/lcpencrypt ./cmd/lcpencrypt
go build -o ./build/lcpchecker ./cmd/lcpchecker
```

Run the LCP Server from the sources:

```sh
LCPSERVER_CONFIG=./config/default.yaml go run ./cmd/lcpserver
```

Or run the built server binary:

```sh
LCPSERVER_CONFIG=./config/default.yaml ./build/lcpserver
```

The server listens on `http://localhost:8989`. You can check it with:

```sh
curl http://localhost:8989/health
```

`lcpencrypt` and `lcpchecker` are command-line tools; run them with `-help` to list their options.

### Installing the LCP Server before moving to its Production mode

The quick install decribed above does not allow you to serve or check production-grade LCP licenses. 
For that, you'll need first to sign a contract with EDRLab and obtain confidential information and instructions. 

If you wish to prepare an installation in Production mode, you must first clone the software. 

Create a working folder (ex. `edrlab`) and, from this folder, enter:

```sh
git clone https://github.com/edrlab/lcp-server.git
```

Option 1: For testing the lcpserver application without compiling it, use:

```sh
# From the lcp-server directory
LCPSERVER_CONFIG=./config/default.yaml go run ./cmd/lcpserver
```

Option 2: For compiling the lcpserver application, use:

```sh
# Compile and create the binary in the local build folder
mkdir -p build
go build -o ./build/lcpserver ./cmd/lcpserver
# Launch the application
LCPSERVER_CONFIG=./config/default.yaml ./build/lcpserver
```

Note: on a Linux Alpine server, the addition of the musl tag is required for building lcpserver. 

```sh
go build -tags musl -o ./build/lcpencrypt ./cmd/lcpencrypt
```

Note: the name of the executable is your choice. You can use `lcpserver2` to avoid a clash with the former version of the LCP Server executable. 

The open-source codebase is provided with **SQLite**, **MySQL** and **PostgresQL** drivers. The default is sqlite. It is up to integrators to replace it by the driver of their choice if sqlite does not fit their needs.

This is achieved by adding a tag at build time: 
> go build -tags MYSQL -o ./build/lcpserver ./cmd/lcpserver

Compile lcpencrypt and lcpchecker using: 

```sh
# Compile and create the binaries in the local build folder
mkdir -p build
go build -o ./build/lcpencrypt ./cmd/lcpencrypt
go build -o ./build/lcpchecker ./cmd/lcpchecker
```

# More

A detailed documentation is available at https://edrlab.github.io/lcp-server/ 
