FROM golang:1.17-alpine AS builder

RUN apk add --no-cache git make build-base

ENV GO111MODULE=on 
ENV CGO_ENABLED=1
ENV GOOS=linux 
ENV GOARCH=amd64
ENV GOSUMDB=off
ENV GOPROXY=direct
ENV GOOGLE_CHROME_PATH = "/usr/bin/google-chrome"
ENV PATH="${GOOGLE_CHROME_PATH}:${PATH}"
ENV PATH_TO_DB="/pricewatcher.db"

WORKDIR /app

COPY go.mod .
COPY go.sum .
COPY pricewatcher.db .

RUN go mod download

COPY . .

RUN go build -o ./main

FROM alpine:3.14

RUN apk add --no-cache iputils busybox-extras curl

WORKDIR /app

COPY --from=builder  /app/main .

RUN chmod 777 /app/main

RUN mkdir /app/db
COPY pricewatcher.db /app/db/pricewatcher.db

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip .
ENV TZ=Europe/Moscow
ENV ZONEINFO=/app/zoneinfo.zip

RUN ls -l -R

ENTRYPOINT ["/app/main"]
