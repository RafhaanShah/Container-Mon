FROM golang:1.18-alpine AS build

WORKDIR /build
COPY . .
RUN go build app.go

FROM alpine:latest

LABEL org.opencontainers.image.source https://github.com/RafhaanShah/Container-Mon

WORKDIR /app
COPY --from=build /build/app /app 
ENTRYPOINT ["./app"]
