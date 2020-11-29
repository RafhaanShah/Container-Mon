FROM golang:1.14-alpine AS build

LABEL org.opencontainers.image.source https://github.com/RafhaanShah/Container-Mon

WORKDIR /build
COPY . .
RUN go build app.go

FROM alpine:latest

WORKDIR /app
COPY --from=build /build/app /app 
ENTRYPOINT ["./app"]
