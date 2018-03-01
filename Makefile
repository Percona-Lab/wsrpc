all:
	go install -race -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -race -v ./cmd/protoc-gen-wsrpc
	rm -f example/api/*.pb.go example/api/*.wsrpc.go
	protoc example/api/*.proto --go_out=.
	protoc example/api/*.proto --wsrpc_out=.
	go install -race -v ./...
	go test -race -v ./...
