#!/bin/sh -eux

goimports -w .
go generate ./...
go vet .
golint .
golint swagger
go test ./... $@
