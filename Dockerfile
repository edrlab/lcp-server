# syntax=docker/dockerfile:1

################################################################################
# Create a stage for building the application.
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# This is the OS you're building for, which is passed in by the builder.
ARG TARGETOS
RUN echo "Building for OS: $TARGETOS"

# This is the architecture you're building for, which is passed in by the builder.
# Placing it here allows the previous steps to be cached across architectures.
# Linux x64 ou macOS x64 (Intel) : amd64
# macOS ARM64 (Apple Silicon M1/M2/M3) : arm64
ARG TARGETARCH
RUN echo "Building for architecture: $TARGETARCH"

# Install any build dependencies that are needed to build the application.
RUN apt-get update && apt-get install -y gcc libc6-dev libmupdf-dev

# Copy source code to avoid mount issues with CGO and static libraries
COPY . /src

# Build lcpencrypt (architecture-independent)
RUN echo "Building lcpencrypt..." && \
    CGO_ENABLED=1 go build -o /app/lcpencrypt ./cmd/lcpencrypt

# Conditional copy and build based on architecture and LCP library availability
RUN if [ "$TARGETARCH" = "amd64" ]; then \
      echo "Building for AMD64 - checking for LCP library"; \
      if [ -f "./config/lib/linux-amd64/libuserkey.a" ]; then \
        echo "LCP library found, copying and building with PLCP support"; \
        cp ./config/lib/linux-amd64/libuserkey.a ./pkg/lic/; \
        CGO_ENABLED=1 go build -tags "PLCP,MYSQL" -o /app/lcpserver ./cmd/lcpserver; \
      else \
        echo "No LCP library found for AMD64, building without PLCP support"; \
        CGO_ENABLED=0 go build -tags "MYSQL" -o /app/lcpserver ./cmd/lcpserver; \
      fi; \
    else \
      echo "Building for $TARGETARCH without LCP library"; \
      CGO_ENABLED=0 go build -tags "MYSQL" -o /app/lcpserver ./cmd/lcpserver; \
    fi

################################################################################
# Create a new stage for running the application that contains the minimal
# runtime dependencies for the application. 

# This stage uses the debian slim image as the foundation for running the app.
#FROM debian:bookworm-slim AS final
FROM debian:trixie-slim AS final

# Install runtime dependencies
# mupdf-tools pulls in all the shared libraries (freetype, jbig2dec, etc.) needed by the CGO-linked binary
RUN apt-get update && apt-get install -y ca-certificates wget mupdf-tools && rm -rf /var/lib/apt/lists/*

# Create a non-privileged user that the app will run under.
ARG UID=10001
RUN useradd --uid "${UID}" --user-group --system --no-log-init --create-home appuser

# Copy the X509 test certificate (to be replaced later by the production certificate)
COPY /config/cert-edrlab-test.pem ./config/
COPY /config/privkey-edrlab-test.pem ./config/
# For production, use:
COPY /config/cert-production.pem ./config/
COPY /config/privkey-production.pem ./config/

# create a directory in the container for input files
RUN mkdir /input
# the user of the container owns the directory. 
RUN chown -R appuser:appuser /input

# create a directory in the container for external resources
RUN mkdir /resources
# the user of the container owns the directory. 
RUN chown -R appuser:appuser /resources

# from here on, the container runs as the non-privileged user
USER appuser

# Copy the executables from the "build" stage.
COPY --from=build /app/lcpserver /app/
COPY --from=build /app/lcpencrypt /app/

# Expose the port that the application listens on.
EXPOSE 8989

# What the container should run when it is started.
CMD ["/app/lcpserver"]