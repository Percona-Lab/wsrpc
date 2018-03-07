# wsrpc

[![Build Status](https://travis-ci.org/Percona-Lab/wsrpc.svg)](https://travis-ci.org/Percona-Lab/wsrpc)
[![codecov](https://codecov.io/gh/Percona-Lab/wsrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/Percona-Lab/wsrpc)
[![GoDoc](https://godoc.org/github.com/Percona-Lab/wsrpc?status.svg)](https://godoc.org/github.com/Percona-Lab/wsrpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/Percona-Lab/wsrpc)](https://goreportcard.com/report/github.com/Percona-Lab/wsrpc)
[![CLA assistant](https://cla-assistant.io/readme/badge/Percona-Lab/wsrpc)](https://cla-assistant.io/Percona-Lab/wsrpc)

RPC-over-WebSocket prototype for PMM 2.0

## TODO

* Use several services over a single connection
* Register several services for a single connection
* Handle double registration of the same service / duplicate method names
* Handle errors from RPC methods
* Proper connection closing
* Authentication (expose HTTP headers?)
* Streaming
* Metrics
* Tweak constants
* More tests
* Fuzz testing
