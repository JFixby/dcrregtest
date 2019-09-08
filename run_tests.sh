#!/usr/bin/env bash

# usage:
# ./run_tests.sh                         # local, go 1.12
# GOVERSION=1.11 ./run_tests.sh          # local, go 1.11
# ./run_tests.sh docker                  # docker, go 1.12
# GOVERSION=1.11 ./run_tests.sh docker   # docker, go 1.11
# ./run_tests.sh podman                  # podman, go 1.12
# GOVERSION=1.11 ./run_tests.sh podman   # podman, go 1.11

set -ex

# The script does automatic checking on a Go package and its sub-packages,
# including:
# 1. gofmt         (https://golang.org/cmd/gofmt/)
# 2. gosimple      (https://github.com/dominikh/go-simple)
# 3. unconvert     (https://github.com/mdempsky/unconvert)
# 4. ineffassign   (https://github.com/gordonklaus/ineffassign)
# 5. go vet        (https://golang.org/cmd/vet)
# 6. misspell      (https://github.com/client9/misspell)
# 7. race detector (https://blog.golang.org/race-detector)

# golangci-lint (github.com/golangci/golangci-lint) is used to run each each
# static checker.

# To run on docker on windows, symlink /mnt/c to /c and then execute the script
# from the repo path under /c.  See:
# https://github.com/Microsoft/BashOnWindows/issues/1854
# for more details.

# Default GOVERSION
[[ ! "$GOVERSION" ]] && GOVERSION=1.12
REPO=dcrregtest

testrepo () {
  GO=go

  $GO version
  dcrd --version
  dcrwallet --version

  # binary needed for RPC tests
  env CC=gcc $GO build

  # run tests on all modules
  export GO111MODULE=on
  go fmt ./...
  go build ./...
  go test ./...

  echo "------------------------------------------"
  echo "Tests completed successfully!"
}

testrepo
exit
