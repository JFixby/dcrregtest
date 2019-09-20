#!/usr/bin/env bash

set -ex

# Default GOVERSION
[[ ! "$GOVERSION" ]] && GOVERSION=1.12
REPO=dcrregtest

testrepo () {
  GO=go
  PROJECT=decred
  NODE_REPO=dcrd
  WALLET_REPO=dcrwallet

  $GO version

  # binary needed for RPC tests
  env CC=gcc

  # run tests on all modules

  pushd ../../
  git clone --depth=50 --branch=release-v1.4 https://github.com/${PROJECT}/${NODE_REPO}.git ${PROJECT}/${NODE_REPO}
  git clone --depth=50 --branch=release-v1.4 https://github.com/${PROJECT}/${WALLET_REPO}.git ${PROJECT}/${WALLET_REPO}
  popd

  $GO fmt ./...
  $GO build ./...

  pushd ../../${PROJECT}/${NODE_REPO}
  $GO build ./...
  $GO install
  popd

  pushd ../../${PROJECT}/${WALLET_REPO}
  $GO build ./...
  $GO install
  popd

  GO111MODULE=on
  ${NODE_REPO} --version
  ${WALLET_REPO} --version
  $GO clean -testcache
  $GO build ./...
  $GO test ./...

  echo "------------------------------------------"
  echo "Tests completed successfully!"
}

testrepo
exit
