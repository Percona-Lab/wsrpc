all: gen test-race

# installs tools to $GOBIN (or $GOPATH/bin) which is expected to be in $PATH
init:
	go install -v -race ./vendor/github.com/golang/protobuf/protoc-gen-go

	go get -u github.com/AlekSi/gocoverutil

	go get -u gopkg.in/alecthomas/gometalinter.v2
	gometalinter.v2 --install

gen:
	go install -v -race ./cmd/protoc-gen-wsrpc
	rm -f example/api/*.pb.go example/api/*.wsrpc.go
	protoc example/api/*.proto --go_out=.
	protoc example/api/*.proto --wsrpc_out=debug=true:.

install:
	go install -v ./...
	go test -v -i ./...

install-race:
	go install -v -race ./...
	go test -v -race -i ./...

test: install
	go test -v ./...

test-race: install-race
	go test -v -race ./...

cover: install
	gocoverutil test -v ./...

check: install
	-gometalinter.v2 --tests --vendor --deadline=300s --sort=path ./...
