# Build the app
# https://docs.docker.com/build/building/multi-platform/#cross-compiling-a-go-application
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build
COPY . .
RUN RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o container-mon

# Run the app
FROM alpine:3

LABEL org.opencontainers.image.source=https://github.com/RafhaanShah/Container-Mon

WORKDIR /app
COPY --from=build /build/container-mon /app
ENTRYPOINT ["./container-mon"]
