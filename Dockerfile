# syntax=docker/dockerfile:1

##
## Build
##

FROM golang:1.16-buster AS build

WORKDIR /msh

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY lib/ ./lib/

RUN go build -o /msh-docker

##
## Deploy
##

FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY README.md ./
COPY msh-config.json ./
COPY --from=build /msh-docker /msh-docker

EXPOSE 25555

USER nonroot:nonroot

ENTRYPOINT ["/msh-docker"]