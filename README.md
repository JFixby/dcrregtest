Decred regression testing
=======
[![Build Status](http://img.shields.io/travis/jfixby/dcrregtest.svg)](https://travis-ci.org/jfixby/dcrregtest)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Harbours a pre-configured test setup and unit-tests to run RPC-driven node tests.

Builds a testing harness crafting and executing integration tests by driving a `dcrd` and `dcrwallet` instances via the `RPC` interface.

## Build 

```
set GO111MODULE=on
go build ./...
go clean -testcache
go test ./...
 ```
 
 ## Tip
 
 The `master` branch is WIP. Use the latest working build from travis: [travis-ci.org/jfixby/dcrregtest/builds](https://travis-ci.org/jfixby/dcrregtest/builds)
 
 ## License
 This code is licensed under the [copyfree](http://copyfree.org) ISC License.
