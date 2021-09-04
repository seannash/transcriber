# syntax=docker/dockerfile:1
##
## Build
##
FROM golang:1.16-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd/ cmd/

RUN go build -o /main cmd/frontend/application.go 
##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /main /main

EXPOSE 5000

USER nonroot:nonroot

ENTRYPOINT ["/main"]
