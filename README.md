# wsrpc
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
* WS pinger, ponger
* TCP keep alives
* Tweak constants
* More tests
* Fuzz testing
