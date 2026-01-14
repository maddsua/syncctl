from docker.io/golang:1.24.11-alpine3.23 as builder

workdir /app/storage_service

copy . /app

run go mod download
run go build -v -ldflags "-s -w" -o s4-server ./cmd

from alpine:3.23

run apk add --no-cache ca-certificates

copy --from=builder /app/storage_service/s4-server /usr/bin/s4-server
copy ./s4server.yml /etc/syncctl/server.yml

entrypoint ["/usr/bin/s4-server"]
