# syntax=docker/dockerfile:1

################################################################################
# Create a stage for building the application.
ARG GO_VERSION=1.24
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# This is the architecture you're building for, which is passed in by the builder.
# Placing it here allows the previous steps to be cached across architectures.
ARG TARGETARCH

# Install any build dependencies that are needed to build the application.
RUN apt-get update && apt-get install -y gcc libc6-dev

# Copy source code to avoid mount issues with CGO and static libraries
COPY . /src

# The libuserkey.a in place must be the linux version, or PLCP must be used
# Enable CGO for MySQL driver and/or LCP support
# Build the application for Docker, with MySQL support but without the LCP production library
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH go build -tags "MYSQL" -o /app/server ./cmd/lcpserver
# Build the application with MySQL and LCP support
#RUN CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH go build -tags "PLCP,MYSQL" -o /app/server ./cmd/lcpserver

################################################################################
# Create a new stage for running the application that contains the minimal
# runtime dependencies for the application. 

# This stage uses the debian slim image as the foundation for running the app.
FROM debian:bookworm-slim AS final

# Install any runtime dependencies that are needed to run your application.
# Leverage a cache mount to /var/cache/apk/ to speed up subsequent builds.
#RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create a non-privileged user that the app will run under.
#ARG UID=10001
#RUN useradd --uid "${UID}" --user-group --system --no-log-init --create-home appuser

# This is the path to the LCP Server configuration file.
# By default, it is the sample configuration file provided with the codebase.
# It is copied into the root dir of the container, with a static name.
# This new file is set as LCPSERVER_CONFIG environment variable.
ARG CONFIG=./config/lcpserver-docker-config.yaml
COPY ${CONFIG} /config/lcpconfig.yaml
ENV LCPSERVER_CONFIG="/config/lcpconfig.yaml"

# Copy the X509 test certificate (to be replaced later by the production certificate)
COPY /config/cert-edrlab-test.pem ./config/
COPY /config/privkey-edrlab-test.pem ./config/

# create a database directory in the container
RUN mkdir /database
# the user of the container owns the database directory. 
#RUN chown -R appuser:appuser /database

# create a directory in the container for external resources
RUN mkdir /resources
# the user of the container owns the directory. 
#RUN chown -R appuser:appuser /resources

# from here on, the container runs as the non-privileged user
#USER appuser

# Copy the executable from the "build" stage.
COPY --from=build /app/server /app/

# Expose the port that the application listens on.
EXPOSE 8989

# What the container should run when it is started.
ENTRYPOINT [ "/app/server" ]
