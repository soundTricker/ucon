#!/bin/sh -eu

set +e
patch --unified --force node_modules/swagger-ui-dist/index.html < ./swagger-ui-index.patch
set -e

echo "open http://localhost:8080/"
go run *.go
