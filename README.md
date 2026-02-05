## Summary

This is a major evolution of the Readium LCP Server available on https://github.com/readium/readium-lcp-server. Because it is not an incremental evolution, and this codebase is entirely maintained by EDRLab, we decided to create a new repository on the EDRLab Github space. 

**Note: This project is almost ready for production. We are currently testing it on a demo plateform before release.**

This project is made of three executables: 

### LCP Server (lcpserver)

The `lcpserver`:

- Receives notifications for each encryption of a publication.
- Serves LCP licenses for these publications. 
- Supports the License Status Document protocol for these licenses.

### LCP Encryption tool (lcpencrypt)

`lcpencrypt`:

- Is both available as a command line utility and a server associated with to a watch folder:
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

## Test Installation

Assuming a working Go installation (Go 1.24 or higher) ... 

You can install the different tools in Test mode using:

```sh
# fetch, build and install the different packages and their dependencies
go install github.com/edrlab/lcp-server/cmd/lcpserver@latest
go install github.com/edrlab/lcp-server/cmd/lcpencrypt@latest
go install github.com/edrlab/lcp-server/cmd/lcpchecker@latest
```

Before testing, the LCP Server requires proper configuration, expressed is a yaml config file and/or environment variables. [Read the documentation to create one](https://edrlab.github.io/lcp-server/).

After install, the LCP Server is launched by the `lcpserver` command. 

`lcpencrypt` and `lcpchecker` require command-line arguments. There again, the documentation is a useful read.

### Installing the LCP Server before moving to its Production mode

The quick install decribed above does not allow you to serve or check production-grade LCP licenses. 
For that, you'll need first to sign a contract with EDRLab and obtain confidential information and instructions. 

If you wish to prepare an installation in Production mode, you must first clone the software. 

Create a working folder (ex. lcp-server), and from this folder, enter:

```sh
git clone https://github.com/edrlab/lcp-server.git
```

Option 1: For testing the lcpserver application without compiling it, use:

```sh
# From the lcp-server directory
go run ./cmd/lcpserver/.
```

Option 2: For compiling the lcpserver application, use:

```sh
# Compile and create the binary in the Go bin folder
go build -o $GOPATH/bin/lcpserver  ./cmd/lcpserver
# Launch the application
lcpserver
```

Note: on a Linux Alpine server, the addition of the musl tag is required for building lcpserver. 

```sh
go build -tags musl -o $GOPATH/bin/lcpencrypt ./cmd/lcpencrypt
```

Note: the name of the executable is your choice. You can use `lcpserver2` to avoid a clash with the former version of the LCP Server executable. 

The open-source codebase is provided with **SQLite**, **MySQL** and **PostgresQL** drivers. The default is sqlite. It is up to integrators to replace it by the driver of their choice if sqlite does not fit their needs.

This is achieved by adding a tag at build time: 
> go build -tags MYSQL -o $GOPATH/bin/lcpserver2  ./cmd/lcpserver

# More

A detailed documentation is available at https://edrlab.github.io/lcp-server/ 
