FROM golang:1.22.3 AS base

WORKDIR /usr/src

COPY api ./api

COPY db/db.go ./db/
COPY db/go.mod ./db/

COPY executor ./executor

COPY configure_db.sql go.mod main.go ./

RUN go work init; \
  go work use api db executor .

RUN go mod download

FROM base AS test

CMD [ "go", "test", "executor" ]

FROM base AS build

RUN mkdir -p /usr/app

CMD [ "go", "build", "-o", "/usr/app/server.out", "main.go" ]
