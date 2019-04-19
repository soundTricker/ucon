#!/bin/bash -eux

cd `dirname $0`

export PATH=$(pwd)/build-cmd:$PATH

goimports -w .
go generate ./...
go vet .
golint .
golint swagger
go test ./... $@
